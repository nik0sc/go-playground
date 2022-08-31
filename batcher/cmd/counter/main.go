package main

import (
	"flag"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"go.lepak.sg/playground/batcher"
)

var (
	countTo = flag.Int("n", 100,
		"how high to count (also determines in channel capacity)")
	batchSize = flag.Int("b", 10,
		"max batch size")
	batchTime = flag.Duration("t", time.Second,
		"max time to wait in the batcher")
	randTime = flag.Duration("r", 300*time.Millisecond,
		"max of random time to wait in the producer")
	prealloc = flag.Bool("p", false,
		"whether to preallocate a full batch for the batch slice")
)

func main() {
	flag.Parse()

	in := make(chan int, *countTo)
	out := make(chan []int)

	var wg sync.WaitGroup

	batcher.Start(in, out, *batchSize, *batchTime, *prealloc, &wg)

	go func() {
		for i := 0; i < *countTo; i++ {
			in <- i

			waitFor := time.Duration(rand.Float64() * float64(*randTime))
			time.Sleep(waitFor)
		}
		close(in)
	}()

	for batch := range out {
		now := time.Now().Format("15:04:05.000")
		fmt.Println(now, batch)
	}

	wg.Wait() // unnecessary
	fmt.Println("bye")
}
