package batcher

import (
	"sync"
	"time"
)

// Start starts a batcher. Start increments wg for you, and the batcher
// exits after in is closed. Otherwise the remaining parameters are the
// same as in batcher.Batch.
func Start[T any](
	in <-chan T, out chan<- []T, threshold int, interval time.Duration,
	prealloc bool, wg *sync.WaitGroup,
) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		Batch(in, out, threshold, interval, prealloc)
	}()
}

// Batch batches up items from the in channel and sends the batches
// on the out channel. It will build batches until either they reach
// the threshold size or the interval has elapsed.
//
// If prealloc is true, the batch slice is allocated with the threshold
// as capacity. If you expect that the timeout won't be reached frequently,
// this could reduce unnecessary reallocations, at the expense of memory use.
func Batch[T any](
	in <-chan T, out chan<- []T, threshold int, interval time.Duration,
	prealloc bool,
) {
	var t *time.Timer

	for {
		// only proceed once there is at least one item
		item, ok := <-in
		if !ok {
			close(out)
			return
		}

		var slice []T
		if prealloc {
			slice = make([]T, 1, threshold)
		} else {
			slice = make([]T, 1)
		}
		slice[0] = item

		// this is a one-shot timer, not a Ticker; it's easier to reason
		// about, since we control its lifetime explicitly
		if t == nil {
			t = time.NewTimer(interval)
		} else {
			t.Reset(interval)
		}

		running := true
		for running {
			select {
			case <-t.C:
				running = false

			case item, ok := <-in:
				if !ok {
					out <- slice
					t.Stop()
					// never using t again, don't care about draining t.C
					close(out)
					return
				}

				slice = append(slice, item)
				if len(slice) >= threshold {
					if !t.Stop() {
						<-t.C
					}
					running = false
				}
			}
		}

		out <- slice
		// on the next iteration, slice will fall out of scope,
		// which is exactly what we want
	}
}
