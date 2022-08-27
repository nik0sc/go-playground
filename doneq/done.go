// Package doneq provides a first-in, first-out done queue.
// This is useful for ensuring tasks that are fanned out to
// a worker pool are recorded as finished in the same order
// as they were started. Some higher-level features which
// could be implemented with this are a checkpoint/resume
// feature in a batch data process, or
//
// Call doneq.New to create a new done queue, and pass in a
// callback function that will record task completion.
// When a task is read from a data source, call Done.Start
// and pass the returned *doneq.Task to the worker.
// When the task is finished in the worker, call Task.Done.
package doneq

import (
	"context"
	"sync"
)

type Done[T any] struct {
	c    chan *Task[T]
	mark func(T)
	wg   sync.WaitGroup

	// pool discipline: tasks in the pool
	// must be unlocked and T must be its zero value,
	// so if T is a pointer type, *T is not kept alive
	pool  sync.Pool // of *Task[T]
	zeroT T
}

// New creates a new done queue. It supports a maximum of `max`
// tasks in flight.
//
// `mark` will be called once for every task started,
// in the same order that the tasks were started, regardless of
// the order that they were finished.
// `mark` runs in its own goroutine.
func New[T any](max int, mark func(T)) *Done[T] {
	if max < 1 {
		panic("max must be >= 1")
	}

	if mark == nil {
		panic("mark must not be nil")
	}

	d := &Done[T]{
		// cap is max-1 as one task can be waiting in watch
		// while the rest are in this channel
		c:    make(chan *Task[T], max-1),
		mark: mark,
	}

	d.pool.New = func() any {
		return &Task[T]{}
	}

	d.wg.Add(1)
	go d.watch()

	return d
}

// Start creates a task with the provided progress indicator.
// Start blocks until either the task is accepted or the context
// is canceled, in which case the context error is returned.
// Tasks are accepted when there are less tasks in flight than
// the maximum passed to New.
//
// To block indefinitely until the task is accepted, pass a
// context.Background() as the context.
func (d *Done[T]) Start(ctx context.Context, progress T) (*Task[T], error) {
	t := d.pool.Get().(*Task[T])
	t.progress = progress
	t.doing.Lock()

	select {
	case d.c <- t:
		return t, nil
	case <-ctx.Done():
		t.progress = d.zeroT
		t.doing.Unlock()
		d.pool.Put(t)
		return nil, ctx.Err()
	}
}

func (d *Done[T]) watch() {
	defer d.wg.Done()
	for t := range d.c {
		t.doing.Lock()

		d.mark(t.progress)

		t.progress = d.zeroT
		t.doing.Unlock()
		d.pool.Put(t)
	}
}

// ShutdownWait shuts down the done queue and returns once
// all tasks in flight are processed. Start must not be called
// after ShutdownWait.
func (d *Done[T]) ShutdownWait() {
	close(d.c)
	d.wg.Wait()
}
