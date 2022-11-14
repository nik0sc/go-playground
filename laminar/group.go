// Package laminar manages goroutines that depend on each other,
// respecting dependency relationships and context cancellation.
// It combines an errgroup.Group with a DAG, and executes tasks
// in topological order.
//
// Create a new group, which encapsulates a dependency graph,
// with laminar.NewGroup. Then add tasks to the group and declare
// dependencies with Group.NewTask().After(). Once all tasks have
// been added, start the group asynchronously with Group.Start,
// and wait for the group to finish with Group.Wait.
package laminar

// Some other libraries that do similar things:
// https://github.com/natessilva/dag
// https://github.com/kamildrazkiewicz/go-flow

// TODO list:
// - group restart? what is the use case? when should this be allowed?
// - pass return values from one task to another? could get messy
// - expose task status directly? encourages racy behaviour as only
//   taskFinished is useful outside the group. only allow status to be
//   read after Group.Wait returns?
//   - subset: expose task success/fail status directly? to recover
//     from failure?
// - what to do about disconnected components?
// - improve test coverage of edge cases??

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"go.lepak.sg/playground/chops"
	"go.lepak.sg/playground/graph"
	"golang.org/x/sync/errgroup"
)

const (
	// NoLimit indicates that the Group can run any number of goroutines at once.
	NoLimit int = -1
)

const (
	taskCreated uint64 = iota
	taskDequeued
	taskWaitingForErrgroup
	taskWaitingForDependents
	taskRunning
	taskFinished
)

type Task struct {
	g       *Group
	name    string
	f       func(context.Context) error
	wg      sync.WaitGroup
	wgChan  <-chan struct{}
	waitFor []<-chan struct{}

	// atomic access only
	// state may be read at any time by String method
	// while the task is in flight
	state uint64
}

type Group struct {
	eg        *errgroup.Group
	egCtx     context.Context
	starterWg sync.WaitGroup

	// protected by starterWg
	savedCtxErr error

	// protected by lock
	lock    sync.Mutex
	graph   graphInterface
	started bool
}

// Any graph implementation must support these 5 methods
type graphInterface interface {
	AddNode(*Task) bool
	AddEdge(*Task, *Task)
	Neighbours(*Task) ([]*Task, bool)
	TopologicalOrder() ([]*Task, error)
	String() string
}

// NewGroup creates a new Group. It accepts a context from which
// another context is derived and passed to tasks in the group.
// limit is the maximum number of goroutines that can run
// simultaneously. Pass NoLimit to disable the limit.
func NewGroup(ctx context.Context, limit int) *Group {
	eg, ctx := errgroup.WithContext(ctx)
	if limit > 0 {
		eg.SetLimit(limit)
	}

	gr := graph.NewAdjacencyListDigraph[*Task]()

	return &Group{
		eg:    eg,
		egCtx: ctx,
		graph: gr,
	}
}

// NewTask creates a new Task in this Group. f accepts a context that is
// canceled after any other f passed to NewTask returns an error.
// name is shown when g.String() is called. There is no "get task by name"
// method: instead, retain the *Task and use it directly.
// NewTask must not be called after Start.
func (g *Group) NewTask(name string, f func(context.Context) error) *Task {
	t := &Task{
		g:    g,
		name: name,
		f:    f,
	}

	g.lock.Lock()
	if g.started {
		g.lock.Unlock()
		panic("Group already started")
	}

	g.graph.AddNode(t)
	g.lock.Unlock()

	return t
}

// After establishes an ordering: all befores complete
// before this task starts.
// After must not be called after the parent Group has started.
// Only one goroutine may call After on any given Task at once.
// However multiple goroutines may call After on different Tasks.
// After returns its Task to make this pattern possible:
//
//	task := group.NewTask(...).After(beforeTask)
//
// This should feel familiar to users of libraries like gomock.
func (t *Task) After(befores ...*Task) *Task {
	for _, before := range befores {
		if t == before {
			panic("Task cannot depend on itself")
		}

		if t.g != before.g {
			panic("Tasks not created from the same Group")
		}
	}

	t.g.lock.Lock()
	if t.g.started {
		t.g.lock.Unlock()
		panic("Group already started")
	}

	for _, before := range befores {
		t.g.graph.AddEdge(before, t)
	}
	t.g.lock.Unlock()

	return t
}

// String returns the string representation of this Task.
func (t *Task) String() string {
	var stateString string
	switch atomic.LoadUint64(&t.state) {
	case taskCreated:
		stateString = "created"
	case taskDequeued:
		stateString = "dequeued"
	case taskWaitingForErrgroup:
		stateString = "waiting for errgroup"
	case taskWaitingForDependents:
		stateString = "waiting for dependents"
	case taskRunning:
		stateString = "running"
	case taskFinished:
		stateString = "finished"
	default:
		stateString = "<unknown>"
	}

	return fmt.Sprintf("%s [%s]", t.name, stateString)
}

// Start starts the Tasks added to the Group in their dependency order.
// It returns an error if the order cannot be established because there
// is a cyclic dependency.
// Start must not be called twice.
//
// The actual task execution order is not guaranteed to be the same across
// multiple starts of the same group, created anew with the same
// dependency graph.
func (g *Group) Start() error {
	g.lock.Lock()

	if g.started {
		g.lock.Unlock()
		panic("Group already started")
	}

	// prevents concurrent Start, but also prevents
	// dependency graph from changing under our feet
	g.started = true
	g.lock.Unlock()

	// now graph will not change
	order, err := g.graph.TopologicalOrder()
	if err != nil {
		return err
	}

	g.starterWg.Add(1)
	go g.starter(order)

	return nil
}

func (g *Group) starter(order []*Task) {
	defer g.starterWg.Done()
	for _, task := range order {
		atomic.StoreUint64(&task.state, taskDequeued)

		select {
		case <-g.egCtx.Done():
			// save the error here, so that if all previous tasks
			// have entered the taskRunning state and don't return
			// the context error, we'll know in Wait that at least
			// one task was prevented from starting
			g.savedCtxErr = g.egCtx.Err()
			return
		default:
		}

		task := task

		dependents, ok := g.graph.Neighbours(task)
		if !ok {
			panic("order and neighbours inconsistent")
		}

		task.wg.Add(1)

		for _, dependent := range dependents {
			if task.wgChan == nil {
				task.wgChan = chops.Wait(&task.wg)
			}

			// this is thread-safe:
			// there is only one goroutine appending to any waitFor slice
			// (this one, running starter)
			// there may be multiple goroutines reading from waitFor,
			// depending on the group limit, but any goroutine will always
			// read after this write has finished, because of the
			// topological order
			dependent.waitFor = append(dependent.waitFor, task.wgChan)
		}

		atomic.StoreUint64(&task.state, taskWaitingForErrgroup)
		g.eg.Go(func() error {
			defer task.wg.Done()

			atomic.StoreUint64(&task.state, taskWaitingForDependents)
			// unstable order in waitFor
			// won't deadlock since circular dependency isn't possible
			for _, doneCh := range task.waitFor {
				select {
				case <-g.egCtx.Done():
					// this is different from errgroup
					return g.egCtx.Err()
				case <-doneCh:
				}
			}

			atomic.StoreUint64(&task.state, taskRunning)
			defer atomic.StoreUint64(&task.state, taskFinished)
			err := task.f(g.egCtx)
			if err != nil {
				err = fmt.Errorf("%s: %w", task.name, err)
			}
			return err
		})
	}
}

// Wait waits for all goroutines started in the Group to exit.
// Wait returns the first error returned from any task, or else
// any context error that prevented all tasks from starting.
// If Wait returns nil, all tasks completed successfully.
func (g *Group) Wait() error {
	g.starterWg.Wait()
	err := g.eg.Wait()
	if err == nil {
		// g.egCtx will be canceled by g.eg.Wait so we can't
		// return g.egCtx.Err directly, it will be too late
		err = g.savedCtxErr
	}
	return err
}

// String returns the string representation of this Group.
func (g *Group) String() string {
	// locking in String is not optimal
	g.lock.Lock()
	started := g.started
	graphStr := g.graph.String()
	g.lock.Unlock()

	return fmt.Sprintf("Group: started=%t\n%s", started, graphStr)
}
