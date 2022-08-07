package donepipeline

import "sync"

type Task[T any] struct {
	doing    sync.Mutex
	progress T
}

func (t *Task[T]) Done() {
	t.doing.Unlock()
}
