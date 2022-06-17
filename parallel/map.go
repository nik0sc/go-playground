// Package parallel provides higher-order functions that run in parallel.
// Maximum concurrency may be restricted for all the functions.
//
// Context cancellation: If the input context is canceled, MapBounded* will
// immediately stop mapping any new items, wait for workers running to exit,
// then return the context error.
package parallel

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"go.lepak.sg/playground/chops"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

const debug = true

// MapBoundedSema maps a list of ~[]T to []R using a provided map function f.
// It does this in parallel with a maximum of inflight workers.
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
		// possible that the context is canceled after we started
		// the last workerbut before we acquired the entire semaphore
		err = sema.Acquire(ctx, int64(inflight))
		if err == nil {
			// context is not canceled, this is the happy path
			return
		}
	}

	// context is already canceled, this will eventually acquire
	for sema.Acquire(ctx, int64(inflight)) != nil {
	}

	return
}

// MapBoundedPool maps a list of ~[]T to []R using a provided map function f.
// It does this in parallel with a fixed-size pool of workers.
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
				currI64 := atomic.LoadInt64(&next)
				if currI64 == int64(len(list)) {
					return
				}

				if !atomic.CompareAndSwapInt64(&next, currI64, currI64+1) {
					continue
				}

				curr := int(currI64)
				if debug {
					wstat[curr] = i
				}
				result[curr] = f(curr, list[curr])
			}
		}()
	}

	select {
	case <-ctx.Done():
		atomic.StoreInt64(&next, int64(len(list)))
		err = ctx.Err()
		// still need to wait for everyone to exit
		wg.Wait()
	case <-chops.Wait(&wg):
	}

	if debug {
		fmt.Println(wstat)
	}

	return
}

// MapBoundedErrgroup maps a list of ~[]T to []R
// using a provided map function f.
// It does this in parallel with a maximum of inflight workers.
// An errgroup.Group is used to coordinate them.
func MapBoundedErrgroup[S ~[]T, T, R any](
	ctx context.Context, list S, f func(int, T) R, workers int,
) (result []R, err error) {
	result = make([]R, len(list))

	g, ctx := errgroup.WithContext(ctx)

	g.SetLimit(workers)

	for i := range list {
		i := i
		if ctx.Err() != nil {
			break
		}
		g.Go(func() error {
			result[i] = f(i, list[i])
			return ctx.Err()
		})
	}

	// err = g.Wait()
	// if err == nil {
	// 	return result, nil
	// } else {
	// 	return nil, err
	// }
	return result, g.Wait()
}

// MapBoundedPoolErrgroup maps a list of ~[]T to []R
// using a provided map function f.
// It does this in parallel with a fixed-size pool of workers.
// An errgroup.Group is used to coordinate them.
func MapBoundedPoolErrgroup[S ~[]T, T, R any](
	ctx context.Context, list S, f func(int, T) R, workers int,
) (result []R, err error) {
	result = make([]R, len(list))
	indices := make(chan int)

	var wstat []int
	if debug {
		wstat = make([]int, len(list))
	}

	g, ctx := errgroup.WithContext(ctx)

	for i := 0; i < workers; i++ {
		i := i
		g.Go(func() error {
			for {
				select {
				case j, ok := <-indices:
					if !ok {
						return nil
					}
					if debug {
						wstat[j] = i
					}
					result[j] = f(j, list[j])
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		})
	}

producer:
	for i := range list {
		select {
		case <-ctx.Done():
			break producer
		case indices <- i:
		}
	}
	close(indices)

	err = g.Wait()

	if debug {
		fmt.Println(wstat)
	}

	return
}
