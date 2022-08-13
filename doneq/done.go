// Package doneq provides a first-in, first-out done queue.
// This is useful for ensuring tasks that are fanned out to
// a worker pool are recorded as finished in the same order
// as they were started, which in turn is useful for implementing
// a checkpoint/resume feature in a batch data process.
//
// Call doneq.New to create a new done queue, and pass in a
// callback function that will record task completion.
// When a task is read from a data source, call Done.Start
// and pass the returned *doneq.Task to the worker.
// When the task is finished in the worker, call Task.Done.
package doneq

import (
	"sync"
)

type Done[T any] struct {
	c    chan *Task[T]
	mark func(T)
	wg   sync.WaitGroup
}

// New creates a new done queue. It supports a maximum of `max`
// tasks in flight.
//
// `mark` will be called once for every task started,
// in the same order that the tasks were started, regardless of
// the order that they were finished.
// `mark` runs in its own goroutine, so it is not necessary
// to synchronize access to any shared memory within `mark`,
// as long as no other goroutines are accessing that memory.
func New[T any](max int, mark func(T)) *Done[T] {
	d := &Done[T]{
		c:    make(chan *Task[T], max),
		mark: mark,
	}

	d.wg.Add(1)
	go d.watch()

	return d
}

// Start creates a task with the provided progress indicator.
// Start can block if there are more tasks in flight than
// the maximum passed to New.
func (d *Done[T]) Start(progress T) *Task[T] {
	t := &Task[T]{
		progress: progress,
	}

	t.doing.Lock()

	d.c <- t

	return t
}

func (d *Done[T]) watch() {
	defer d.wg.Done()
	for t := range d.c {
		t.doing.Lock()

		d.mark(t.progress)
	}
}

// ShutdownWait shuts down the done queue and returns once
// all tasks in flight are processed. Start must not be called
// after ShutdownWait.
func (d *Done[T]) ShutdownWait() {
	close(d.c)
	d.wg.Wait()
}
