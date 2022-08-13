package doneq

import "sync"

// Task is returned from a call to Done.Start or Last.Start.
type Task[T any] struct {
	doing    sync.Mutex
	progress T
}

// Done marks the Task as completed and ready to be marked.
// Done returns immediately. Done must not be called twice.
func (t *Task[T]) Done() {
	t.doing.Unlock()
}

func (t *Task[T]) T() T {
	return t.progress
}
