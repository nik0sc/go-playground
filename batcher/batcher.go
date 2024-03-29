// Package batcher provides a timed batch middleware.
// It will build batches until either they reach a maximum size
// or a maximum interval elapses.
// The batcher interacts with other code through two channels:
// an input channel on which items are received, and an output
// channel on which slices of items are sent.
//
// Inspired by https://old.reddit.com/r/golang/comments/v9m37a
// "Looking for examples of a "batch release threshold" pattern"
package batcher

import (
	"sync"
	"time"
)

type Params struct {
	Threshold int
	Interval  time.Duration
	Prealloc  bool
}

// Start starts a batcher. Start increments wg for you, and the batcher
// exits after in is closed. Otherwise the remaining parameters are the
// same as in batcher.Batch.
func Start[T any](
	in <-chan T, out chan<- []T, wg *sync.WaitGroup, params Params,
) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		Batch(in, out, params)
	}()
}

// Batch batches up items from the in channel and sends the batches
// on the out channel. It will build batches until either they reach
// the threshold size or the interval has elapsed. Batch exits after
// in is closed, and it will close out as well (i.e. channel closing
// propagates through Batch).
//
// If params.Prealloc is true, the batch slice is allocated with the
// threshold as capacity. If you expect that the timeout won't be
// reached frequently, this could reduce unnecessary reallocations.
// On the other hand, if the timeout is actually reached frequently,
// setting this to false would reduce unnecessary memory usage.
func Batch[T any](
	in <-chan T, out chan<- []T, params Params,
) {
	batch(in, out, params, true)
}

func batch[T any](
	in <-chan T, out chan<- []T, params Params,
	shouldClose bool,
) {
	var t *time.Timer

	if shouldClose {
		defer close(out)
	}

	for {
		// only proceed once there is at least one item
		item, ok := <-in
		if !ok {
			return
		}

		var slice []T
		if params.Prealloc {
			slice = make([]T, 1, params.Threshold)
		} else {
			slice = make([]T, 1)
		}
		slice[0] = item

		if params.Threshold <= 1 {
			out <- slice
			continue
		}

		// this is a one-shot timer, not a Ticker; it's easier to reason
		// about, since we control its lifetime explicitly
		if t == nil {
			t = time.NewTimer(params.Interval)
		} else {
			t.Reset(params.Interval)
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
					return
				}

				slice = append(slice, item)
				if len(slice) >= params.Threshold {
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
