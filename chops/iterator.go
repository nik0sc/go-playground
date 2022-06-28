package chops

// Iterator describes some iterator over a data structure.
// It must not require closing at the end of iteration,
// as CoIterate may abandon it at any time.
type Iterator[T any] interface {
	Next() bool
	Item() T
}

// CoIterator is returned from CoIterate and abstracts
// communication with the iterating goroutine.
type CoIterator[T any] struct {
	items <-chan T
	stop  chan<- struct{}
}

// Items returns a channel on which the items from the iterator
// will be sent.
func (c CoIterator[T]) Items() <-chan T {
	return c.items
}

// Stop stops the iteration. This must not be called more than once.
// If the Items channel is closed, this doesn't need to be called.
//
// If you need to stop from multiple goroutines, use a sync.Once:
//
//	var once sync.Once
//	co := CoIterate[T](...)
//	for i := 0; i < 10; i++ {
//		go func() {
//			for item := range co.Items() {
//				if item meets some stopping condition {
//					once.Do(co.Stop)
//				}
//			}
//		}()
//	}
func (c CoIterator[T]) Stop() {
	close(c.stop)
}

// CoIterate starts coroutine-style iteration.
// The usage is as follows:
//
//	var x SomeDataStructure[T]
//	// x.Iterator() returns something that implements Iterator[T]
//	co := CoIterate[T](x.Iterator())
//	for i := range co.Items() {
//		... do stuff with i ...
//		if i meets some stopping condition {
//			co.Stop()
//		}
//	}
//
// If you might pass a typed nil pointer into CoIterate,
// make sure your underlying type's methods can handle
// being called with a nil receiver.
//
// Note: CoIterate starts a goroutine, which exits when either
// Stop() is called or the iteration is finished.
// If you follow the usage above, the goroutine will not live beyond
// the end of the for-range loop.
//
// Another way to use CoIterate without exhausting the underlying Iterator
// but also without explicit Stopping is to use runtime.SetFinalizer:
//
//	co := CoIterate[T](x.Iterator())
//	runtime.SetFinalizer(&co, (*CoIterator[T]).Stop)
//	for i := range co.Items() {
//		... do stuff with i ...
//		if i meets some stopping condition {
//			break
//		}
//	}
//	runtime.KeepAlive(&co)
//	// after this point, the GC may run co.Stop for you
//
// This approach is taken from the Ranger example in the generics proposal:
// https://go.googlesource.com/proposal/+/refs/heads/master/design/43651-type-parameters.md#channels.
// The explicit Stop approach should be preferred where practical, as
// the lifetime of the iterating goroutine is very obvious, and
// using runtime.SetFinalizer has some performance impact.
func CoIterate[T any](iterator Iterator[T]) CoIterator[T] {
	out := make(chan T)
	stop := make(chan struct{})
	co := CoIterator[T]{
		items: out,
		stop:  stop,
	}

	if iterator == nil {
		close(out)
		return co
	}

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

	return co
}
