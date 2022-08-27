package doneq

import (
	"context"
	"sync"
	"time"

	"go.lepak.sg/playground/batcher"
)

type Last[T any] struct {
	c      chan *Task[T]
	cbatch chan []*Task[T]
	mark   func(T)
	wg     sync.WaitGroup
}

// NewLast creates a new done queue. It supports a maximum of `max`
// tasks in flight. Unlike standard New, it will only call the mark
// function periodically - every `threshold` tasks, or when
// `interval` elapses, whichever happens first.
//
// This is not suitable for applications where every task
// must be marked.
func NewLast[T any](
	max int, mark func(T), threshold int, interval time.Duration,
) *Last[T] {
	d := &Last[T]{
		c:      make(chan *Task[T], max),
		cbatch: make(chan []*Task[T], 1),
		mark:   mark,
	}

	batcher.Start(d.c, d.cbatch, threshold, interval, false, &d.wg)

	d.wg.Add(1)
	go d.watch()

	return d
}

// Start creates a task with the provided progress indicator.
// Start can block if there are more tasks in flight than
// the maximum passed to NewLast.
func (l *Last[T]) Start(ctx context.Context, progress T) (*Task[T], error) {
	// TODO use Done and integrate batcher on top of it
	t := &Task[T]{
		progress: progress,
	}

	t.doing.Lock()

	l.c <- t

	return t, nil
}

func (l *Last[T]) watch() {
	defer l.wg.Done()
	for batch := range l.cbatch {
		t := batch[len(batch)-1]

		t.doing.Lock()

		l.mark(t.progress)
	}
}

// ShutdownWait shuts down the done queue and returns once
// all tasks in flight are processed. Start must not be called
// after ShutdownWait.
func (l *Last[T]) ShutdownWait() {
	close(l.c)
	l.wg.Wait()
}
