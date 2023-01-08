package doneq

import (
	"context"
	"sync"
	"time"

	"go.lepak.sg/playground/batcher"
)

type Batched[T any] struct {
	done   *Done[T]
	wg     sync.WaitGroup
	c      chan T
	cbatch chan []T
	mark   func(T)
}

// NewBatched creates a new done queue. `max`, the maximum number
// of tasks in flight, must be at least 1.
// Unlike standard New, it will only call the mark
// function periodically - every `threshold` tasks, or when
// `interval` elapses, whichever happens first.
//
// This is not suitable for applications where every task
// must be marked.
func NewBatched[T any](
	max int, mark func(T), threshold int, interval time.Duration,
) *Batched[T] {
	if mark == nil {
		panic("mark must not be nil")
	}

	d := &Batched[T]{
		c:      make(chan T, threshold),
		cbatch: make(chan []T, 1),
		mark:   mark,
	}

	d.done = New(max, func(progress T) {
		d.c <- progress
	})

	batcher.Start(d.c, d.cbatch, &d.wg, batcher.Params{
		Threshold: threshold,
		Interval:  interval,
	})

	d.wg.Add(1)
	go d.watch()

	return d
}

// Start creates a task with the provided progress indicator.
// Start blocks until either the task is accepted or the context
// is canceled, in which case the context error is returned.
// Tasks are accepted when there are less tasks in flight than
// the maximum passed to NewBatched.
//
// To block indefinitely until the task is accepted, pass a
// context.Background() as the context.
func (d *Batched[T]) Start(ctx context.Context, progress T) (*Task[T], error) {
	return d.done.Start(ctx, progress)
}

func (d *Batched[T]) watch() {
	defer d.wg.Done()
	for batch := range d.cbatch {
		d.mark(batch[len(batch)-1])
	}
}

// ShutdownWait shuts down the done queue and returns once
// all tasks in flight are processed. Start must not be called
// after ShutdownWait. ShutdownWait must not be called while
// any goroutine is blocked in Start.
func (d *Batched[T]) ShutdownWait() {
	d.done.ShutdownWait()
	close(d.c)
	d.wg.Wait()
}
