package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"go.lepak.sg/playground/doneq"
	"golang.org/x/exp/slices"
)

var (
	tasks = flag.Int("t", 100,
		"number of tasks to start")
	workers = flag.Int("w", 5,
		"number of workers to process tasks")
	doneqMax = flag.Int("c", 10,
		"max number of inflight tasks")
	delayMax = flag.Duration("d", time.Second,
		"max delay duration (actual task delay is between 0s and this)")

	dqImpl = flag.String("dq", "",
		"doneq implementation to use (\"last\" to use NewLast)")
	lastThreshold = flag.Int("lt", 3,
		"(last only) threshold")
	lastInterval = flag.Duration("ld", time.Second,
		"(last only) interval")
)

func main() {
	flag.Parse()

	var wg sync.WaitGroup
	var markOrder []int
	appender := func(i int) {
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
	}

	var dq interface {
		Start(context.Context, int) (*doneq.Task[int], error)
		ShutdownWait()
	}

	switch *dqImpl {
	case "last":
		fmt.Println("using NewLast")
		dq = doneq.NewLast(*doneqMax, appender, *lastThreshold, *lastInterval)
	default:
		fmt.Println("using New")
		dq = doneq.New(*doneqMax, appender)
	}

	fanOut := make(chan *doneq.Task[int])

	startTime := time.Now()
	delayMaxF := float64(*delayMax)

	// workers: this simulates tasks being processed in a random order
	wg.Add(*workers)
	for i := 0; i < *workers; i++ {
		i := i
		go func() {
			for t := range fanOut {
				sleepFor := time.Duration(rand.Float64() * delayMaxF)
				time.Sleep(sleepFor)

				delta := time.Since(startTime).String()
				fmt.Printf("%13s: worker %d marking %d\n", delta, i, t.T())
				t.Done()
			}
			wg.Done()
		}()
	}

	// source: this simulates data being read serially from
	// some data source like a database or csv file
	for i := 0; i < *tasks; i++ {
		// if progress were recorded here,
		// then if the process is killed
		// the tasks in flight would be counted as complete
		// even though they did not finish
		// in real use, you should use a cancelable context
		// and handle the error, which ensures forward progress
		// or at least prevents deadlock when quitting
		t, _ := dq.Start(context.Background(), i)
		fanOut <- t
	}
	close(fanOut)

	// after calling ShutdownWait, the done queue cannot create
	// new tasks, and all tasks will have completed
	dq.ShutdownWait()
	// this is unnecessary
	wg.Wait()

	fmt.Println(markOrder)
	fmt.Println("is sorted?", slices.IsSorted(markOrder))
}
