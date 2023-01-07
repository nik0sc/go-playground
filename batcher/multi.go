package batcher

import (
	"sync"
	"time"

	"go.lepak.sg/playground/slidingwindow"
)

// Keyer is the interface that items must implement
// to be aggregated into groups.
type Keyer[K comparable] interface {
	Key() K
}

type window[K comparable] interface {
	Observe(K)
	Lifetime() int
}

type multi[T Keyer[K], K comparable] struct {
	active map[K]chan T
	wg     *sync.WaitGroup
	subWg  sync.WaitGroup
	window window[K]

	in        <-chan T
	out       chan<- []T
	threshold int
	interval  time.Duration
	prealloc  bool
	subInCap  int
}

func (m *multi[T, K]) accept() {
	for item := range m.in {
		key := item.Key()

		dest, ok := m.active[key]
		if !ok {
			println("create: key:", key, "lifetime:", m.window.Lifetime())
			dest = make(chan T, m.subInCap)
			m.subWg.Add(1)
			go func() {
				defer m.subWg.Done()
				batch(dest, m.out, m.threshold, m.interval, m.prealloc, false)
			}()
			m.active[key] = dest
		}

		dest <- item
		m.window.Observe(key)
	}

	// shutdown
	// TODO: The order may be important here...
	for _, dest := range m.active {
		print("")
		close(dest)
	}

	m.subWg.Wait()
	close(m.out)
	m.wg.Done()
}

func (m *multi[T, K]) cleanup(key K) {
	println("evict: key:", key, "lifetime:", m.window.Lifetime())
	ch := m.active[key]
	if ch != nil {
		close(ch)
	}
	delete(m.active, key)
}

func StartMulti[T Keyer[K], K comparable](
	in <-chan T, out chan<- []T, threshold int, interval time.Duration,
	prealloc bool, subInCap, keepAliveFor, keyCardinalityHint int,
	wg *sync.WaitGroup,
) {
	m := &multi[T, K]{
		active:    make(map[K]chan T),
		wg:        wg,
		in:        in,
		out:       out,
		threshold: threshold,
		interval:  interval,
		prealloc:  prealloc,
		subInCap:  subInCap,
	}
	m.window = slidingwindow.NewCounter(keepAliveFor, keyCardinalityHint, m.cleanup)

	wg.Add(1)
	go m.accept()
}
