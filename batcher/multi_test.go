package batcher

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.lepak.sg/playground/testutils"
	"go.uber.org/goleak"
	"golang.org/x/exp/slices"
)

type char byte

func (c char) String() string {
	return string(c)
}

// static type assertion
var _ *grouped[string, char] = nil

func stringKeyer(str string) char {
	if len(str) == 0 {
		return 0
	}
	return char(str[0])
}

func TestStartMulti(t *testing.T) {
	const (
		cardinality = 0    // 100
		times       = 1000 // detect flakiness
	)

	tests := []struct {
		name               string
		repeat             bool
		inCap, outCap      int
		before, concurrent func(chan string)
		params             GroupedParams
		drain              [][]string
		drainFunc          func(*testing.T, [][]string)
	}{
		{
			name:   "none",
			inCap:  0,
			outCap: 1,
			concurrent: func(ch chan string) {
				close(ch)
			},
			params: GroupedParams{
				Params: Params{
					Threshold: 10,
					Interval:  time.Second,
				},
				SubChannelCap: 1,
				Lifetime:      1,
			},
		},
		{
			name:   "degenerate",
			repeat: true,
			inCap:  4,
			outCap: 4,
			concurrent: func(ch chan string) {
				ch <- "apple"
				ch <- "banana"
				ch <- "blueberry"
				ch <- "apricot"
				close(ch)
			},
			drainFunc: func(t *testing.T, d [][]string) {
				assert.Less(t,
					slices.IndexFunc(d, func(dd []string) bool {
						return dd[0] == "apple"
					}),
					slices.IndexFunc(d, func(dd []string) bool {
						return dd[0] == "apricot"
					}),
				)

				assert.Less(t,
					slices.IndexFunc(d, func(dd []string) bool {
						return dd[0] == "banana"
					}),
					slices.IndexFunc(d, func(dd []string) bool {
						return dd[0] == "blueberry"
					}),
				)
			},
		},
		{
			name:   "grouping order",
			repeat: true,
			inCap:  10,
			outCap: 4,
			before: func(ch chan string) {
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
			params: GroupedParams{
				Params: Params{
					Threshold: 3,
					Interval:  time.Second,
				},
				// subchcap should be 3
				Lifetime: 6,
			},
			drainFunc: func(t *testing.T, d [][]string) {
				assert.ElementsMatch(t, d, [][]string{
					{"apple"},
					{"apricot", "avocado"},
					{"banana", "blueberry", "blackcurrant"},
					{"cherry", "coconut", "cantaloupe"},
				})

				a1Index := slices.IndexFunc(d, func(dd []string) bool {
					return dd[0] == "apple"
				})

				a2Index := slices.IndexFunc(d, func(dd []string) bool {
					return dd[0] == "apricot"
				})

				assert.GreaterOrEqual(t, a1Index, 0)
				assert.GreaterOrEqual(t, a2Index, 0)
				assert.Less(t, a1Index, a2Index, "same-key order broken")
			},
			drain: [][]string{ // ignored - drainFunc
				{"apple"},
				{"apricot", "avocado"},
				{"banana", "blueberry", "blackcurrant"},
				{"cherry", "coconut", "cantaloupe"},
			},
		},
	}
	for _, tt := range tests {
		n := 1
		if tt.repeat {
			n = times
		}

		for i := 0; i < n; i++ {
			t.Run(tt.name, func(t *testing.T) {
				in := make(chan string, tt.inCap)
				out := make(chan []string, tt.outCap)

				if tt.before != nil {
					tt.before(in)
				}

				var wg sync.WaitGroup
				StartGrouped(in, out, stringKeyer, &wg, tt.params)

				if tt.concurrent != nil {
					tt.concurrent(in)
				}

				wg.Wait()
				if tt.drainFunc != nil {
					var drain [][]string
					ok := true
					for ok {
						var d []string
						select {
						case d, ok = <-out:
							if ok {
								drain = append(drain, d)
							}
						default:
							t.Fatal("(drainFunc) channel not closed or " +
								"multi.accept still running after wg.Done")
						}
					}
					t.Logf("drain: %v", drain)
					tt.drainFunc(t, drain)
				} else {
					testutils.Drain(t, tt.drain, out)
				}
				goleak.VerifyNone(t)
			})

		}
	}
}
