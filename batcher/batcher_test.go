package batcher

import (
	"sync"
	"testing"
	"time"

	"go.uber.org/goleak"

	"go.lepak.sg/playground/testutils"
)

func TestBatch(t *testing.T) {
	tests := []struct {
		name               string
		inCap, outCap      int
		before, concurrent func(chan int)
		params             Params
		drain              [][]int
	}{
		{
			name:   "none",
			inCap:  0,
			outCap: 1,
			concurrent: func(ch chan int) {
				close(ch)
			},
			params: Params{
				Threshold: 10,
				Interval:  time.Second,
			},
		},
		{
			name:   "none with delay",
			inCap:  0,
			outCap: 1,
			concurrent: func(ch chan int) {
				time.Sleep(time.Second)
				close(ch)
			},
			params: Params{
				Threshold: 10,
				Interval:  time.Millisecond,
			},
		},
		{
			name:   "one no delay",
			inCap:  0,
			outCap: 1,
			concurrent: func(ch chan int) {
				ch <- 1
				close(ch)
			},
			params: Params{
				Threshold: 10,
				Interval:  time.Second,
			},
			drain: [][]int{{1}},
		},
		{
			name:   "two with delay between",
			inCap:  2,
			outCap: 2,
			concurrent: func(ch chan int) {
				ch <- 1
				time.Sleep(time.Second)
				ch <- 2
				close(ch)
			},
			params: Params{
				Threshold: 10,
				Interval:  time.Millisecond,
			},
			drain: [][]int{{1}, {2}},
		},
		{
			name:   "no delay",
			inCap:  10,
			outCap: 4,
			before: func(ch chan int) {
				for i := 0; i < 10; i++ {
					ch <- i
				}
				close(ch)
			},
			params: Params{
				Threshold: 3,
				Interval:  time.Second,
			},
			drain: [][]int{
				{0, 1, 2},
				{3, 4, 5},
				{6, 7, 8},
				{9},
			},
		},
		{
			name:   "with delay",
			inCap:  10,
			outCap: 4,
			concurrent: func(ch chan int) {
				for i := 0; i < 10; i++ {
					if i == 5 {
						time.Sleep(2 * time.Second)
					}
					ch <- i
				}
				close(ch)
			},
			params: Params{
				Threshold: 3,
				Interval:  time.Second,
			},
			drain: [][]int{
				{0, 1, 2},
				{3, 4},
				{5, 6, 7},
				{8, 9},
			},
		},
		{
			name:   "degenerate",
			inCap:  3,
			outCap: 3,
			concurrent: func(ch chan int) {
				for i := 0; i < 3; i++ {
					ch <- i
				}
				close(ch)
			},
			drain: [][]int{
				{0},
				{1},
				{2},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			in := make(chan int, test.inCap)
			out := make(chan []int, test.outCap)

			if test.before != nil {
				test.before(in)
			}

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				Batch(in, out, test.params)
			}()

			if test.concurrent != nil {
				test.concurrent(in)
			}

			wg.Wait()
			testutils.Drain(t, test.drain, out)
			goleak.VerifyNone(t)
		})
	}
}
