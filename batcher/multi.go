package batcher

import (
	"sync"
	"time"
)

type Keyer interface {
	Key() string
}

type Multi[T Keyer] struct {
	active map[string]chan T
	subWg  sync.WaitGroup

	in        <-chan T
	out       chan<- []T
	threshold int
	interval  time.Duration
	prealloc  bool
	subInCap  int
}

func (m *Multi[T]) accept(wg *sync.WaitGroup) {
	for item := range m.in {
		key := item.Key()

		dest, ok := m.active[key]
		if !ok {
			dest = make(chan T, m.subInCap)
			m.subWg.Add(1)
			go func() {
				defer m.subWg.Done()
				batch(dest, m.out, m.threshold, m.interval, m.prealloc, false)
			}()
			m.active[key] = dest
		}

		dest <- item
	}

	// shutdown
	for _, dest := range m.active {
		close(dest)
	}

	m.subWg.Wait()
	wg.Done()
}

func StartMulti[T Keyer](
	in <-chan T, out chan<- []T, threshold int, interval time.Duration,
	prealloc bool, subInCap int, wg *sync.WaitGroup,
) {
	m := &Multi[T]{
		active:    make(map[string]chan T),
		in:        in,
		out:       out,
		threshold: threshold,
		interval:  interval,
		prealloc:  prealloc,
		subInCap:  subInCap,
	}

	wg.Add(1)
	go m.accept(wg)
}
