// Package laminar manages goroutines that depend on each other.
package laminar

import (
	"context"
	"fmt"
	"sync"

	"go.lepak.sg/playground/chops"
	"go.lepak.sg/playground/graph"
	"golang.org/x/sync/errgroup"
)

const (
	// NoLimit indicates that the Group can run any number of goroutines at once.
	NoLimit int = -1
)

type Task struct {
	g       *Group
	name    string
	f       func(context.Context) error
	wg      sync.WaitGroup
	waitFor []*sync.WaitGroup

	// wgChan <-chan struct{}
	// waitFor []<-chan struct{}
}

type Group struct {
	eg        *errgroup.Group
	egCtx     context.Context
	starterWg sync.WaitGroup

	// protected by lock
	lock    sync.Mutex
	graph   *graph.AdjacencyListDigraph[*Task]
	started bool
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
// name is shown when g.String() is called.
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

// After establishes an ordering: before happens before this task.
// After must not be called after the parent Group has started.
func (t *Task) After(before *Task) {
	if t == before {
		panic("Task cannot depend on itself")
	}

	if t.g != before.g {
		panic("Tasks not created from the same Group")
	}

	t.g.lock.Lock()
	if t.g.started {
		t.g.lock.Unlock()
		panic("Group already started")
	}

	t.g.graph.AddEdge(before, t)
	t.g.lock.Unlock()

	// possible to double-add, it should not affect correctness though
	t.waitFor = append(t.waitFor, &before.wg)
}

func (t *Task) String() string {
	return t.name
}

// Start starts the Tasks added to the Group in their dependency order.
// It returns an error if the order cannot be established because there
// is a cyclic dependency.
// Start must not be called twice.
func (g *Group) Start() error {
	var order []*Task
	var err error
	g.lock.Lock()

	func() {
		defer g.lock.Unlock()

		if g.started {
			panic("Group already started")
		}

		g.started = true

		order, err = g.graph.TopologicalOrder()
		if err != nil {
			return
		}
	}()

	if err != nil {
		return err
	}

	g.starterWg.Add(1)
	go func() {
		defer g.starterWg.Done()
		for _, task := range order {
			select {
			case <-g.egCtx.Done():
				return
			default:
			}

			task := task

			task.wg.Add(1)
			g.eg.Go(func() error {
				defer task.wg.Done()
				// unstable order in waitFor: will it deadlock?
				for _, wg := range task.waitFor {
					select {
					case <-g.egCtx.Done():
						return nil // ctx err will be returned anyway
					case <-chops.Wait(wg):
						// may be inefficient to do this for tasks that have high outdegree
						// conversely, efficient for groups where only few tasks depend on others
						// perhaps it can be a tunable option
					}
				}

				return task.f(g.egCtx)
			})
		}
	}()

	return nil
}

// Wait waits for all goroutines started in the Group to exit.
// Note: even after this method returns, not all the Tasks may have run.
func (g *Group) Wait() error {
	g.starterWg.Wait()
	return g.eg.Wait()
}

// String returns the string representation of this Group.
func (g *Group) String() string {
	g.lock.Lock()
	started := g.started
	graphStr := g.graph.String()
	g.lock.Unlock()

	return fmt.Sprintf("Group: started=%t\n%s", started, graphStr)
}
