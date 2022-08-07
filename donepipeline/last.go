package donepipeline

import (
	"sync"
	"time"

	"go.lepak.sg/playground/batcher"
)

type Last[T any] struct {
	c    chan *Task[T]
	cb   chan []*Task[T]
	mark func(T)
	wg   sync.WaitGroup
}

func NewLast[T any](
	cap int, mark func(T), threshold int, interval time.Duration,
) *Last[T] {
	d := &Last[T]{
		c: make(chan *Task[T], cap)
		cb: make(chan []*Task[T], 1)
		mark: mark,
	}

	batcher.Start(d.c, d.cb, threshold, interval, false, &d.wg)

	d.wg.Add(1)
	go d.watch()

	return d
}

func (l *Last[T]) Start(progress T) *Task[T] {
	t := &Task[T]{
		progress: progress,
	}

	t.doing.Lock()

	d.c <- t

	return t
}

func (l *Last[T]) watch() {
	defer l.wg.Done()
	for batch := range l.cb {
		t := batch[len(batch)-1]

		t.doing.Lock()

		l.mark(t.progress)
	}
}

func (d *Done[T]) ShutdownWait() {
	close(d.c)
	d.wg.Wait()
}
