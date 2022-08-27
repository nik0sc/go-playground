package doneq

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"golang.org/x/exp/slices"
)

func ExampleDone() {
	var wg, wgWorker sync.WaitGroup
	var markOrder []int
	const tasks = 100
	const workers = 5
	// doneqMax has a rate-limiting effect exerted through
	// backpressure on the data source
	const doneqMax = 10
	const delayMax = time.Millisecond

	last := New(doneqMax, func(i int) {
		// this function could be used to record task progress
		// and flush it to a checkpoint file
		// this way, you will only record progress for tasks
		// that are definitely finished, and
		// you may end up losing progress if there are tasks
		// that are done but still waiting for another task
		// started earlier to finish

		// it's safe to do this without synchronization
		// as this function will only run from one goroutine
		markOrder = append(markOrder, i)
	})

	fanOut := make(chan *Task[int])

	// source: this simulates data being read serially from
	// some data source like a database or file
	wg.Add(1)
	go func() {
		for i := 0; i < tasks; i++ {
			// in real use, you should pass a real context and
			// check the error, which ensures forward progress
			t, _ := last.Start(context.Background(), i)
			fanOut <- t
		}
		close(fanOut)
		wg.Done()
	}()

	// workers: this simulates tasks being processed in a random order
	wgWorker.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			for t := range fanOut {
				sleepFor := time.Duration(rand.Float64() * float64(delayMax))
				time.Sleep(sleepFor)
				t.Done()
			}
			wgWorker.Done()
		}()
	}

	wg.Wait()
	last.ShutdownWait()
	wgWorker.Wait()

	fmt.Println("last item:", markOrder[len(markOrder)-1])
	fmt.Println("is sorted?:", slices.IsSorted(markOrder))
	// Output:
	// last item: 99
	// is sorted?: true
}
