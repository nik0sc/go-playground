package donepipeline

import (
	"sync"
)

type Done[T any] struct {
	c    chan *Task[T]
	mark func(T)
	wg   sync.WaitGroup
}

func New[T any](cap int, mark func(T)) *Done[T] {
	d := &Done[T]{
		c:    make(chan *Task[T], cap),
		mark: mark,
	}

	d.wg.Add(1)
	go d.watch()

	return d
}

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

func (d *Done[T]) ShutdownWait() {
	close(d.c)
	d.wg.Wait()
}
