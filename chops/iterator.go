package chops

// Iterator describes some iterator over a data structure.
// It must not require closing at the end of iteration,
// as CoIterate may abandon it at any time.
type Iterator[T any] interface {
	Next() bool
	Item() T
}

// CoIterate starts coroutine-style iteration.
// The usage is as follows:
//
//	var x SomeDataStructure
//	items, stop := CoIterate(x.Iterator())
//	for i := range items {
//		... do stuff with i ...
//		if i meets some stopping condition {
//			close(stop)
//		}
//	}
//
// Note: CoIterate starts a goroutine, which exits when either
// the chan struct{} is closed or the iteration is finished.
// If you follow the usage above, the goroutine will not live beyond
// the end of the for-range loop.
func CoIterate[T any](iterator Iterator[T]) (<-chan T, chan<- struct{}) {
	out := make(chan T)
	stop := make(chan struct{})

	go func(out chan<- T, stop <-chan struct{}, i Iterator[T]) {
		defer close(out)
		for i.Next() {
			select {
			case out <- i.Item():
			case <-stop:
				return
			}
		}
	}(out, stop, iterator)

	return out, stop
}
