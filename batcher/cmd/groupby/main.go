package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"go.lepak.sg/playground/batcher"
)

const (
	cardinality = 256
)

var (
	threshold = flag.Int("t", 3, "threshold")
	interval  = flag.Duration("d", time.Second, "duration")
	subChCap  = flag.Int("s", 0, "sub channel capacity")
	lifetime  = flag.Int("i", 10, "lifetime")
	// cardinality = flag.Int("c", 256, "key cardinality hint")
)

func keyer(s string) byte {
	if len(s) == 0 {
		return 0
	}
	return s[0]
}

func main() {
	flag.Parse()

	in := make(chan string)
	out := make(chan []string)
	var wg sync.WaitGroup

	batcher.StartGrouped(in, out, keyer, &wg, batcher.GroupedParams{
		Params: batcher.Params{
			Threshold: *threshold,
			Interval:  *interval,
		},
		SubChannelCap:      *subChCap,
		Lifetime:           *lifetime,
		KeyCardinalityHint: cardinality,
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			in <- sc.Text()
		}
		fmt.Println("scanner ended:", sc.Err())
		close(in)
	}()

	for b := range out {
		fmt.Println(b)
	}

	wg.Wait()
}
