package parallel

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/semaphore"
)

type mbsig[S ~[]T, T, R any] func(context.Context, S, func(int, T) R, int) ([]R, error)

func getmbimpl[S ~[]T, T, R any](name string) mbsig[S, T, R] {
	switch name {
	case "sema":
		return MapBoundedSema[S, T, R]
	case "pool":
		return MapBoundedPool[S, T, R]
	case "lf":
		return MapBoundedPoolLockfree[S, T, R]
	default:
		return nil
	}
}

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

	// possible that the context is canceled after we started the last worker
	// but before we acquired the entire semaphore
	// we shouldn't exit before the workers
	for sema.Acquire(ctx, int64(inflight)) != nil {
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

	for i := 0; i < workers; i++ {
		wg.Add(1)
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
				fmt.Printf("w%d i%d\n", i, curr)
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
		wg.Wait()
	case <-notifyDone:
	}

	return
}
