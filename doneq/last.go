package doneq

import (
	"context"
	"sync"
	"time"

	"go.lepak.sg/playground/batcher"
)

type Last[T any] struct {
	done   *Done[T]
	wg     sync.WaitGroup
	c      chan T
	cbatch chan []T
	mark   func(T)
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
	if mark == nil {
		panic("mark must not be nil")
	}

	l := &Last[T]{
		c:      make(chan T, threshold),
		cbatch: make(chan []T, 1),
		mark:   mark,
	}

	l.done = New(max, func(progress T) {
		l.c <- progress
	})

	batcher.Start(l.c, l.cbatch, threshold, interval, false, &l.wg)

	l.wg.Add(1)
	go l.watch()

	return l
}

// Start creates a task with the provided progress indicator.
// Start can block if there are more tasks in flight than
// the maximum passed to NewLast.
func (l *Last[T]) Start(ctx context.Context, progress T) (*Task[T], error) {
	return l.done.Start(ctx, progress)
}

func (l *Last[T]) watch() {
	defer l.wg.Done()
	for batch := range l.cbatch {
		l.mark(batch[len(batch)-1])
	}
}

// ShutdownWait shuts down the done queue and returns once
// all tasks in flight are processed. Start must not be called
// after ShutdownWait.
func (l *Last[T]) ShutdownWait() {
	l.done.ShutdownWait()
	close(l.c)
	l.wg.Wait()
}
