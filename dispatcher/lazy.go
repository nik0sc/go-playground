package dispatcher

import (
	"sync"

	"go.lepak.sg/playground/slidingwindow"
)

const (
	defaultWindow = 100
)

type Keyer interface {
	Key() string
}

type Acceptor[T Keyer] interface {
	Accept(T)
	Close()
}

type Lazy[T Keyer] struct {
	lock   sync.RWMutex
	active map[string]Acceptor[T]
	// only hold on to the key, not the dispatched item
	// to avoid keeping it alive for too long
	// also because Keyer should not imply comparable
	window  *slidingwindow.Counter[string]
	factory func(string) Acceptor[T]
}

func NewLazy[T Keyer](
	factory func(string) Acceptor[T], windowSize, keyCardinality int,
) *Lazy[T] {
	ld := &Lazy[T]{
		active:  make(map[string]Acceptor[T]),
		factory: factory,
	}

	if windowSize < 1 {
		windowSize = defaultWindow
	}

	ld.window = slidingwindow.NewCounter(windowSize, keyCardinality, ld.cleanup)

	return ld
}

func (ld *Lazy[T]) Accept(item T) {
	key := item.Key()

	ld.lock.RLock()
	if ld.window == nil {
		ld.lock.RUnlock()
		panic("closed")
	}

	dest, ok := ld.active[key]
	if ok {
		// XXX Multiple rlock holders should not access window?
		ld.window.Observe(key)
	}
	ld.lock.RUnlock()

	if !ok {
		ld.lock.Lock()
		if ld.window == nil {
			ld.lock.Unlock()
			panic("closed")
		}

		dest, ok = ld.active[key]
		if !ok {
			dest = ld.factory(key)
			ld.active[key] = dest
		}
		ld.window.Observe(key)
		ld.lock.Unlock()
	}

	dest.Accept(item)
}

func (ld *Lazy[T]) Close() {
	ld.lock.Lock()
	defer ld.lock.Unlock()

	ld.window = nil
	for _, dest := range ld.active {
		dest.Close()
	}
}

func (ld *Lazy[T]) cleanup(key string) {
	// full write lock must be held! do this in the background
	// otherwise Lazy.Accept will deadlock with its own cleanup
	go func() {
		ld.lock.Lock()
		dest, ok := ld.active[key]
		delete(ld.active, key)
		ld.lock.Unlock()

		if ok && dest != nil {
			dest.Close()
		}
	}()
}
