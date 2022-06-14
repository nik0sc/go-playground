package parallel

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/semaphore"
)

const debug = true

// MapBoundedSema maps a list of ~[]T to []R using a provided map function f.
// It does this in parallel with a maximum of inflight workers.
//
// Context cancellation: If the context is canceled, MapBounded* will
// immediately stop mapping any new items, wait for workers running to exit,
// then return the context error.
func MapBoundedSema[S ~[]T, T, R any](
	ctx context.Context, list S, f func(int, T) R, inflight int,
) (result []R, err error) {
	result = make([]R, len(list))

	sema := semaphore.NewWeighted(int64(inflight))

	for i, v := range list {
		err = sema.Acquire(ctx, 1)
		if err != nil {
			// ctx was canceled
			break
		}

		go func(i int, v T) {
			defer sema.Release(1)
			result[i] = f(i, v)
		}(i, v)
	}

	if err == nil {
		// possible that the context is canceled after we started the last worker
		// but before we acquired the entire semaphore
		err = sema.Acquire(ctx, int64(inflight))
		if err != nil {
			for sema.Acquire(ctx, int64(inflight)) != nil {
			}
		}
	} else {
		// context is already canceled, this will eventually acquire
		for sema.Acquire(ctx, int64(inflight)) != nil {
		}
	}

	return
}

// MapBoundedPool maps a list of ~[]T to []R using a provided map function f.
// It does this in parallel with a fixed-size pool of workers.
//
// Context cancellation: If the context is canceled, MapBounded* will
// immediately stop mapping any new items, wait for workers running to exit,
// then return the context error.
func MapBoundedPool[S ~[]T, T, R any](
	ctx context.Context, list S, f func(int, T) R, workers int,
) (result []R, err error) {
	result = make([]R, len(list))
	indices := make(chan int, workers)
	var wg sync.WaitGroup

	var wstat []int
	if debug {
		wstat = make([]int, len(list))
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case j, ok := <-indices:
					if !ok {
						return
					}
					if debug {
						wstat[j] = i
					}
					result[j] = f(j, list[j])
				}
			}
		}()
	}

producer:
	for i := range list {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break producer
		case indices <- i:
		}
	}
	close(indices)

	wg.Wait()

	if debug {
		fmt.Println(wstat)
	}

	return
}

// MapBoundedPoolLockfree maps a list of ~[]T to []R using a
// provided map function f.
// It does this in parallel with a fixed-size pool of workers.
// No channels or locks are involved in the dispatch process.
//
// Context cancellation: If the context is canceled, MapBounded* will
// immediately stop mapping any new items, wait for workers running to exit,
// then return the context error.
func MapBoundedPoolLockfree[S ~[]T, T, R any](
	ctx context.Context, list S, f func(int, T) R, workers int,
) (result []R, err error) {
	result = make([]R, len(list))
	var next int64
	var wg sync.WaitGroup

	var wstat []int
	if debug {
		wstat = make([]int, len(list))
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()

			for {
				curr := atomic.LoadInt64(&next)
				if curr == int64(len(list)) {
					return
				}

				if !atomic.CompareAndSwapInt64(&next, curr, curr+1) {
					continue
				}
				if debug {
					wstat[int(curr)] = i
				}
				result[int(curr)] = f(int(curr), list[int(curr)])
			}
		}()
	}

	// a little clunky, but still better than spinning on
	// atomic.LoadInt64(&next) == int64(len(list))
	notifyDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(notifyDone)
	}()

	select {
	case <-ctx.Done():
		atomic.StoreInt64(&next, int64(len(list)))
		err = ctx.Err()
		// still need to wait for everyone to exit
		<-notifyDone
	case <-notifyDone:
	}

	if debug {
		fmt.Println(wstat)
	}

	return
}
