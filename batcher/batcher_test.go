package batcher

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"go.lepak.sg/playground/testutils"
)

func TestBatch(t *testing.T) {
	tests := []struct {
		name               string
		inCap, outCap      int
		before, concurrent func(chan int)
		threshold          int
		interval           time.Duration
		prealloc           bool
		drain              [][]int
	}{
		{
			name:   "none",
			inCap:  0,
			outCap: 1,
			concurrent: func(ch chan int) {
				close(ch)
			},
			threshold: 10,
			interval:  time.Second,
		},
		{
			name:   "none with delay",
			inCap:  0,
			outCap: 1,
			concurrent: func(ch chan int) {
				time.Sleep(time.Second)
				close(ch)
			},
			threshold: 10,
			interval:  time.Millisecond,
		},
		{
			name:   "one no delay",
			inCap:  0,
			outCap: 1,
			concurrent: func(ch chan int) {
				ch <- 1
				close(ch)
			},
			threshold: 10,
			interval:  time.Second,
			drain:     [][]int{{1}},
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
			threshold: 10,
			interval:  time.Millisecond,
			drain:     [][]int{{1}, {2}},
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
			threshold: 3,
			interval:  time.Second,
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
			threshold: 3,
			interval:  time.Second,
			drain: [][]int{
				{0, 1, 2},
				{3, 4},
				{5, 6, 7},
				{8, 9},
			},
		},
		{
			name:   "threshold 1",
			inCap:  3,
			outCap: 3,
			before: func(ch chan int) {
				for i := 0; i < 3; i++ {
					ch <- i
				}
				close(ch)
			},
			threshold: 1,
			interval:  time.Second,
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
				Batch(in, out, test.threshold, test.interval, test.prealloc)
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

func TestBatch_Panic(t *testing.T) {
	tests := []struct {
		name      string
		in        <-chan int
		out       chan<- []int
		threshold int
		interval  time.Duration
	}{
		{
			name:      "nil in",
			in:        nil,
			out:       make(chan<- []int),
			threshold: 1,
			interval:  time.Second,
		},
		{
			name:      "nil out",
			in:        make(<-chan int),
			out:       nil,
			threshold: 1,
			interval:  time.Second,
		},
		{
			name:      "0 threshold",
			in:        make(<-chan int),
			out:       make(chan<- []int),
			threshold: 0,
			interval:  time.Second,
		},
		{
			name:      "0 interval",
			in:        make(<-chan int),
			out:       make(chan<- []int),
			threshold: 1,
			interval:  0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Panics(t, func() {
				Batch(test.in, test.out, test.threshold, test.interval, false)
			})
		})
	}
}
