package batcher

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.lepak.sg/playground/testutils"
	"go.uber.org/goleak"
)

type strData string

func (s strData) Key() byte {
	if len(s) == 0 {
		return 0
	}
	return s[0]
}

// static type assertion
var _ *multi[strData, byte] = nil

func TestStartMulti(t *testing.T) {
	const (
		cardinality = 0 // 100
	)

	tests := []struct {
		name               string
		inCap, outCap      int
		before, concurrent func(chan strData)
		threshold          int
		interval           time.Duration
		prealloc           bool
		subInCap           int
		keepAliveFor       int
		drain              [][]strData
		drainFunc          func(*testing.T, [][]strData)
	}{
		{
			name:   "none",
			inCap:  0,
			outCap: 1,
			concurrent: func(ch chan strData) {
				close(ch)
			},
			threshold:    10,
			interval:     time.Second,
			subInCap:     1,
			keepAliveFor: 1,
		},
		{
			name:   "grouping order",
			inCap:  10,
			outCap: 4,
			before: func(ch chan strData) {
				ch <- "apple"
				ch <- "banana"
				ch <- "cherry"
				ch <- "blueberry"
				ch <- "coconut"
				ch <- "blackcurrant"
				ch <- "cantaloupe"
				ch <- "apricot"
				ch <- "avocado"
				close(ch)
			},
			threshold:    3,
			interval:     time.Second,
			subInCap:     3,
			keepAliveFor: 6,
			drainFunc: func(t *testing.T, d [][]strData) {
				assert.ElementsMatch(t, d, [][]strData{
					{"apple"},
					{"apricot", "avocado"},
					{"banana", "blueberry", "blackcurrant"},
					{"cherry", "coconut", "cantaloupe"},
				})
				// TODO: Order should be assured somehow
			},
			drain: [][]strData{ // ignored - drainFunc
				{"apple"},
				{"apricot", "avocado"},
				{"banana", "blueberry", "blackcurrant"},
				{"cherry", "coconut", "cantaloupe"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := make(chan strData, tt.inCap)
			out := make(chan []strData, tt.outCap)

			if tt.before != nil {
				tt.before(in)
			}

			var wg sync.WaitGroup
			StartMulti[strData, byte](
				in,
				out,
				tt.threshold,
				tt.interval,
				tt.prealloc,
				tt.subInCap,
				tt.keepAliveFor,
				cardinality,
				&wg)

			if tt.concurrent != nil {
				tt.concurrent(in)
			}

			wg.Wait()
			if tt.drainFunc != nil {
				var drain [][]strData
				ok := true
				for ok {
					var d []strData
					select {
					case d, ok = <-out:
						if ok {
							drain = append(drain, d)
						}
					default:
						t.Fatal("channel not closed or multi.accept still running after wg.Done (drainFunc)")
					}
				}
				tt.drainFunc(t, drain)
			} else {
				testutils.Drain(t, tt.drain, out)
			}
			goleak.VerifyNone(t)
		})
	}
}
