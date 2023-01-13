package dispatcher

import (
	"fmt"
	"sync"
	"sync/atomic"

	"go.lepak.sg/playground/slidingwindow"
)

const (
	defaultWindow = 100
)

// Keyer is the interface of items that Lazy accepts.
type Keyer interface {
	Key() string
}

// Acceptor is the interface that Lazy will route Keyers to.
type Acceptor interface {
	Accept(Keyer) error
	Close()
}

// only hold on to the key, not the dispatched item
// to avoid keeping it alive for too long
// also because Keyer should not imply comparable
type counter interface {
	Observe(string)
}

type acceptorEntry struct {
	acceptor Acceptor
	// refcount is atomically incremented when retrieving
	// this acceptorEntry from Lazy.active
	// then decremented after acceptor is used
	refCount int64
}

type Lazy struct {
	active  map[string]*acceptorEntry
	window  counter
	factory func(string) (Acceptor, error)
	// protects active and window (the reference to LockedCounter)
	// not allowed to hold this and then call window methods
	lock sync.RWMutex
}

// It should be possible to chain Lazys together
var _ Acceptor = (*Lazy)(nil)

// NewLazy creates a lazy dispatcher.
// factory is the function called to create a new Acceptor for
// a given key. windowSize is the number of items accepted without using
// an Acceptor before it is considered idle and closed. keyCardinality
// is a guess for how many unique keys there are (it may be 0).
//
// Note that factory may be called from multiple goroutines at once.
// Also, an Acceptor may be created and then closed without ever
// being used.
// Once an Acceptor is no longer needed, its Close method is called.
func NewLazy(
	factory func(string) (Acceptor, error), windowSize, keyCardinality int,
) *Lazy {
	ld := &Lazy{
		active:  make(map[string]*acceptorEntry),
		factory: factory,
	}

	if windowSize < 1 {
		windowSize = defaultWindow
	}

	ld.window = slidingwindow.NewLocked(slidingwindow.NewCounter(
		windowSize, keyCardinality, ld.cleanup))

	return ld
}

func (ld *Lazy) newAcceptor(key string) (ac Acceptor, err error) {
	defer func() {
		switch r := recover().(type) {
		case error:
			err = fmt.Errorf("factory paniced: %w", r)
		case nil:
			if err != nil {
				err = fmt.Errorf("factory: %w", err)
			}
		default:
			err = fmt.Errorf("factory paniced: %v", r)
		}
	}()
	ac, err = ld.factory(key)
	return
}

// Accept accepts a keyable item for dispatching.
// Any error from the acceptor is returned.
func (ld *Lazy) Accept(item Keyer) error {
	key := item.Key()

	ld.lock.RLock()
	if ld.window == nil {
		ld.lock.RUnlock()
		panic("closed")
	}

	dest, ok := ld.active[key]
	if ok {
		// prevent ld.window.Observe -> (evict) -> ld.cleanup
		// from closing this acceptor in another goroutine.
		// ld.cleanup acquires the full r/w lock before checking refcount,
		// but this increment happens while ld.Accept holds the rlock,
		// so it's not possible for ld.cleanup to see refcount = 0
		// while ld.Accept is using the acceptor
		atomic.AddInt64(&dest.refCount, 1)
	}
	ld.lock.RUnlock()

	if !ok {
		// avoid calling the factory while holding lock
		acceptor, err := ld.newAcceptor(key)
		if err != nil {
			return err
		}

		ld.lock.Lock()
		if ld.window == nil {
			ld.lock.Unlock()
			panic("closed")
		}

		dest, ok = ld.active[key]
		if !ok {
			dest = &acceptorEntry{
				acceptor: acceptor,
				refCount: 1, // using it now
			}
			ld.active[key] = dest
		} else {
			atomic.AddInt64(&dest.refCount, 1)
		}
		ld.lock.Unlock()

		if ok {
			// close the unnecessary acceptor that was
			// just created
			acceptor.Close()
		}
	}

	ld.window.Observe(key)
	err := dest.acceptor.Accept(item)
	atomic.AddInt64(&dest.refCount, -1)
	return err
}

func (ld *Lazy) Close() {
	ld.lock.Lock()
	defer ld.lock.Unlock()

	ld.window = nil
	for _, dest := range ld.active {
		dest.acceptor.Close()
	}
}

func (ld *Lazy) cleanup(key string) {
	ld.lock.Lock()
	dest, ok := ld.active[key]
	if !ok {
		ld.lock.Unlock()
		panic("key already removed")
	}
	if atomic.LoadInt64(&dest.refCount) != 0 {
		ld.lock.Unlock()
		// This is tricky: the key has already left the window
		// but the acceptor should not be removed from ld.active.
		// Trust that the other goroutine will Observe this key also,
		// therefore putting the key back in the window.
		// print("warning: refcount was not zero", dest)
		return
	}
	delete(ld.active, key)
	ld.lock.Unlock()

	if dest != nil {
		dest.acceptor.Close()
	}
}
