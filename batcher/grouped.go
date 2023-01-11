package batcher

import (
	"fmt"
	"io"
	"runtime"
	"sync"
	"sync/atomic"

	"go.lepak.sg/playground/slidingwindow"
)

/*
Useful properties:
- On input channel close, everything in flight should be output,
then the output channel should be closed
- Empty batches should not be output
- Any batch will contain only items with the same key
- Items should be output in the same order as they are input,
within the same key, across all batches.
  This implies that an evicted sub-batcher should exit
  *before* a new one for the same key is created!
*/

type GroupedParams struct {
	// Params is used to create sub-batchers.
	Params

	// SubChannelCap is the capacity of sub-channels
	// created as the input to the sub-batchers.
	// If this is 0, the lesser of the input channel capacity
	// and the threshold will be used instead.
	// To use an unbuffered channel, set this to -1.
	SubChannelCap int

	// Lifetime is the number of items received
	// that do not match some key before an idle sub-batcher
	// for that key is stopped.
	//
	// For example, if this is 10, and an item with the key 'A'
	// is received, then if the next 10 items do not have
	// the key 'A', the sub-batcher for 'A' is stopped.
	//
	// If this is 0, sub-batchers will be kept alive
	// forever. If the key cardinality is high,
	// this may cause high memory usage.
	Lifetime int

	// KeyCardinalityHint is a hint of the key cardinality,
	// i.e. the number of distinct values of the key.
	// This is used for performance reasons; see the
	// documentation of [slidingwindow.NewCounter] for details.
	// If this is 0, a default will be inferred
	// based on the key type.
	KeyCardinalityHint int
}

type window[K comparable] interface {
	Observe(K)
	Lifetime() int
}

type noopWindow[K any] struct{}

func (noopWindow[K]) Observe(K)     {}
func (noopWindow[K]) Lifetime() int { return 0 }

const (
	stateRunning uint64 = iota
	stateClosing
	stateDeleted
)

type sub[T any, K comparable] struct {
	key   K
	ch    chan T         // to sub-batcher
	wg    sync.WaitGroup // sub-batcher decrements this
	state uint64
}

type grouped[T any, K comparable] struct {
	active map[K]*sub[T, K]

	// window records uses of sub-batchers
	// so that inactive ones can be stopped
	// if lifetime = 0, this is a dummy
	window window[*sub[T, K]]

	in  <-chan T
	out chan<- []T

	// incremented by StartGrouped
	// and decremented when accept/cleanup exit
	wg *sync.WaitGroup

	subParams Params
	subChCap  int

	keyer func(T) K

	// evictq connects the window and the cleanup goroutine
	// if lifetime = 0, since window does nothing,
	// evictq will be nil
	evictq chan *sub[T, K]

	debug io.Writer

	maplock sync.Mutex // protects active
	// incremented when creating a new sub-batcher
	// and decremented when one exits
	subWg sync.WaitGroup
}

func (m *grouped[T, K]) accept() {
	for item := range m.in {
		m.acceptOne(item)
	}

	// shutdown sub-batchers in any order
	for _, dest := range m.active {
		close(dest.ch)
	}

	// shutdown cleanup
	// if lifetime = 0, evictq is nil, and
	// cleanup was never started
	if m.evictq != nil {
		close(m.evictq)
	}

	// once all sub-batchers have exited,
	// close m.out on their behalf
	m.subWg.Wait()
	close(m.out)

	// cleanup will decrement m.wg as well
	m.wg.Done()
}

func (m *grouped[T, K]) acceptOne(item T) {
	key := m.keyer(item)
	// later on, if this is not nil,
	// we have to wait for this sub-batcher
	// to exit before creating its replacement
	var oldRec *sub[T, K]

	m.maplock.Lock()
	subRec, ok := m.active[key]
	m.maplock.Unlock()

	// this would have become true in enqueueForCleanup,
	// which runs in this goroutine too, so there is no race
	if ok && atomic.LoadUint64(&subRec.state) >= stateClosing {
		ok = false
		oldRec = subRec
	}

	if !ok {
		if m.debug != nil {
			fmt.Fprintf(m.debug, "create: key=%v lifetime=%d\n",
				key, m.window.Lifetime())
		}

		subRec = &sub[T, K]{
			key: key,
			ch:  make(chan T, m.subChCap),
		}
		subRec.wg.Add(1)
		m.subWg.Add(1)
		go m.runSub(subRec)

		// spinwait if needed
		if oldRec != nil {
			oldRec.wg.Wait() // but don't spin for too long
			for atomic.LoadUint64(&oldRec.state) != stateDeleted {
				// probably won't spend too long in here
				// accept isn't holding the lock
				// so cleanup can grab it immediately
				runtime.Gosched()
			}
			// once here, cleanup has removed the old
			// sub-batcher from m.active, so accept can
			// go ahead and add the new sub-batcher
		}

		m.maplock.Lock()
		m.active[key] = subRec
		m.maplock.Unlock()
	}

	subRec.ch <- item
	// will trigger eviction via enqueueForCleanup if needed
	m.window.Observe(subRec)
}

func (m *grouped[T, K]) runSub(subRec *sub[T, K]) {
	defer subRec.wg.Done()
	defer m.subWg.Done()
	// shouldClose = false, because it is not
	// the only one sending on m.out
	batch(subRec.ch, m.out, m.subParams, false)
}

func (m *grouped[T, K]) enqueueForCleanup(rec *sub[T, K]) {
	if m.debug != nil {
		fmt.Fprintf(m.debug, "evictq: key=%v lifetime=%d\n",
			rec.key, m.window.Lifetime())
	}

	if rec == nil {
		panic("rec nil but key still pending cleanup")
	}

	if !atomic.CompareAndSwapUint64(&rec.state, stateRunning, stateClosing) {
		panic("state != stateRunning")
	}

	close(rec.ch)
	m.evictq <- rec
	// What happens if we skip evictq and start a goroutine here
	// to wait for rec.wg and remove it from m.active directly?
}

func (m *grouped[T, K]) cleanup() {
	for evictee := range m.evictq {
		if m.debug != nil {
			fmt.Fprintf(m.debug,
				"cleanup: key=%v\n", evictee.key)
			// not safe to read window lifetime
		}

		evictee.wg.Wait()

		m.maplock.Lock()
		if evictee != m.active[evictee.key] {
			panic("map changed underneath us")
		}

		delete(m.active, evictee.key)
		m.maplock.Unlock()

		if !atomic.CompareAndSwapUint64(
			&evictee.state, stateClosing, stateDeleted) {
			panic("state != stateClosing")
		}
	}

	m.wg.Done()
}

// StartGrouped starts a grouping batcher. Every item received on the
// in channel has a group, which is determined by a custom keyer function.
// The grouping batcher works by creating sub-batchers for each unique
// item key observed. StartGrouped increments wg for you, and the batcher
// exits along with all started sub-batchers after in is closed.
// It will also close out for you.
//
// The keyer function determines the group of an item. It should be a
// pure function, i.e. it should always return the same K for any given T.
//
// GroupedParams embeds Params, which controls the batching behaviour
// for each group. GroupedParams also introduces new parameters specific
// to group control, e.g. for the cleanup of idle sub-batchers.
func StartGrouped[T any, K comparable](
	in <-chan T, out chan<- []T, keyer func(T) K, wg *sync.WaitGroup,
	params GroupedParams,
) {
	if params.SubChannelCap == 0 {
		if cap(in) < params.Threshold {
			params.SubChannelCap = cap(in)
		} else {
			params.SubChannelCap = params.Threshold
		}
	} else if params.SubChannelCap < 0 {
		params.SubChannelCap = 0
	}

	var mapsize int
	if params.Lifetime > 0 {
		mapsize = params.Lifetime
	} else if params.KeyCardinalityHint > 0 {
		mapsize = params.KeyCardinalityHint
	} else {
		mapsize = 100 // just a guess
	}

	m := &grouped[T, K]{
		active:    make(map[K]*sub[T, K], mapsize),
		wg:        wg,
		in:        in,
		out:       out,
		subParams: params.Params,
		subChCap:  params.SubChannelCap,
		keyer:     keyer,
		// debug:     os.Stderr,
	}

	if params.Lifetime > 0 {
		// TODO: This chan can be smaller, but by how much?
		m.evictq = make(chan *sub[T, K], params.Lifetime)
		// no need for the locked counter, since
		// Observe is only called serially from acceptOne
		m.window = slidingwindow.NewCounter(
			params.Lifetime, params.KeyCardinalityHint, m.enqueueForCleanup)
		wg.Add(1)
		go m.cleanup()
	} else {
		m.window = noopWindow[*sub[T, K]]{}
	}

	wg.Add(1)
	go m.accept()
}
