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

type record[T any] struct {
	ch    chan T         // to sub-batcher
	wg    sync.WaitGroup // sub-batcher decrements this
	state uint64
}

type evictRec[T any, K comparable] struct {
	rec *record[T] // avoids another map lock and access in cleanup
	key K
}

type grouped[T any, K comparable] struct {
	active  map[K]*record[T]
	maplock sync.Mutex
	wg      *sync.WaitGroup
	subWg   sync.WaitGroup

	window window[K]

	in        <-chan T
	out       chan<- []T
	subParams Params
	subChCap  int

	keyer func(T) K

	evictq chan evictRec[T, K]

	debug io.Writer
}

func (m *grouped[T, K]) accept() {
	for item := range m.in {
		m.acceptOne(item)
	}

	// shutdown
	for _, dest := range m.active {
		close(dest.ch)
	}

	if m.evictq != nil {
		close(m.evictq)
	}

	m.subWg.Wait()
	close(m.out)
	m.wg.Done()
}

func (m *grouped[T, K]) acceptOne(item T) {
	key := m.keyer(item)
	var oldRec *record[T]

	m.maplock.Lock()
	rec, ok := m.active[key]
	m.maplock.Unlock()

	// this would be set in cleanup2 - which runs in the same goroutine
	// as accept - so there is no race
	if ok && atomic.LoadUint64(&rec.state) >= stateClosing {
		ok = false
		oldRec = rec // tells the if !ok {} block to wait for state to change
	}

	if !ok {
		if m.debug != nil {
			fmt.Fprintf(m.debug, "create: key=%v lifetime=%d\n",
				key, m.window.Lifetime())
		}

		rec = &record[T]{
			ch: make(chan T, m.subChCap),
		}
		rec.wg.Add(1)
		m.subWg.Add(1)
		go func() {
			defer rec.wg.Done()
			defer m.subWg.Done()
			batch(rec.ch, m.out, m.subParams, false)
		}()

		// spinwait if needed
		if oldRec != nil {
			oldRec.wg.Wait() // but don't spin for too long
			for atomic.LoadUint64(&oldRec.state) != stateDeleted {
				// odds are we won't spend too long in here
				// accept isn't holding the lock
				// so cleanup can grab it immediately
				runtime.Gosched()
			}
		}

		m.maplock.Lock()
		m.active[key] = rec
		m.maplock.Unlock()
	}

	rec.ch <- item
	m.window.Observe(key)
}

func (m *grouped[T, K]) enqueueForCleanup(key K) {
	if m.debug != nil {
		fmt.Fprintf(m.debug, "evictq: key=%v lifetime=%d\n",
			key, m.window.Lifetime())
	}

	m.maplock.Lock()
	rec := m.active[key]
	m.maplock.Unlock()
	if rec != nil {
		if !atomic.CompareAndSwapUint64(&rec.state, stateRunning, stateClosing) {
			panic("rec.state != stateRunning")
		}
		m.evictq <- evictRec[T, K]{
			rec: rec,
			key: key,
		}
	}
}

func (m *grouped[T, K]) cleanup() {
	for evictee := range m.evictq {
		if m.debug != nil {
			fmt.Fprintf(m.debug,
				"cleanup: key=%v\n", evictee.key)
			// not safe to read window lifetime
		}

		close(evictee.rec.ch)
		evictee.rec.wg.Wait()

		m.maplock.Lock()
		if evictee.rec != m.active[evictee.key] {
			panic("map changed underneath us")
		}

		delete(m.active, evictee.key)
		m.maplock.Unlock()

		if !atomic.CompareAndSwapUint64(
			&evictee.rec.state, stateClosing, stateDeleted) {
			panic("rec.state != stateClosing")
		}
	}

	m.wg.Done()
}

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

	m := &grouped[T, K]{
		active:    make(map[K]*record[T]),
		wg:        wg,
		in:        in,
		out:       out,
		subParams: params.Params,
		subChCap:  params.SubChannelCap,
		keyer:     keyer,
		// debug:     os.Stderr,
	}

	if params.Lifetime > 0 {
		m.evictq = make(chan evictRec[T, K], params.Lifetime)
		m.window = slidingwindow.NewCounter(
			params.Lifetime, params.KeyCardinalityHint, m.enqueueForCleanup)
		wg.Add(1)
		go m.cleanup()
	} else {
		m.window = noopWindow[K]{}
	}

	wg.Add(1)
	go m.accept()
}
