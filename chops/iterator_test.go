package chops

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.lepak.sg/playground/testutils"
	"go.uber.org/goleak"
)

var _ Iterator[int] = (*sliter)(nil)

type sliter struct {
	s []int
	i int
}

func (sl *sliter) Next() bool {
	if sl == nil {
		return false
	}
	sl.i++
	return sl.i < len(sl.s)
}

func (sl *sliter) Item() int {
	return sl.s[sl.i]
}

func TestCoIterate_Nil(t *testing.T) {
	// This tests that untyped nil pointer can be handled
	co := CoIterate[int](nil)
	_, ok := <-co.Items()
	assert.False(t, ok)
}

func TestCoIterate(t *testing.T) {
	tests := []struct {
		name string
		sl   *sliter
		do   func(t *testing.T, co CoIterator[int])
	}{
		{
			name: "empty",
			do: func(t *testing.T, co CoIterator[int]) {
				testutils.DrainBlocking(t, nil, co.Items(), time.Second)
			},
		},
		{
			name: "one",
			sl: &sliter{
				s: []int{1},
				i: -1,
			},
			do: func(t *testing.T, co CoIterator[int]) {
				testutils.DrainBlocking(t, []int{1}, co.Items(), time.Second)
			},
		},
		{
			name: "stopping",
			sl: &sliter{
				s: []int{1, 2, 3},
				i: -1,
			},
			do: func(t *testing.T, co CoIterator[int]) {
				assert.Equal(t, 1, <-co.Items())
				co.Stop()
				testutils.DrainBlocking(t, nil, co.Items(), time.Second)
			},
		},
		{
			name: "usage",
			sl: &sliter{
				s: []int{1, 2, 3},
				i: -1,
			},
			do: func(t *testing.T, co CoIterator[int]) {
				var a []int
				for i := range co.Items() {
					a = append(a, i)
					if i == 1 {
						co.Stop()
					}
				}
				assert.Equal(t, []int{1}, a)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.do(t, CoIterate[int](tt.sl))
			goleak.VerifyNone(t)
		})
	}
}

func TestCoIterate_Concurrent(t *testing.T) {
	sl := &sliter{
		s: make([]int, 100),
		i: -1,
	}
	for i := range sl.s {
		sl.s[i] = i + 1
	}
	co := CoIterate[int](sl)

	barrier := make(chan struct{})
	var once sync.Once
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			<-barrier
			for j := range co.Items() {
				if j > 50 {
					once.Do(co.Stop)
				}
			}
		}()
	}

	close(barrier)
	wg.Wait()

	t.Logf("next index of iterator=%d", sl.i)

	goleak.VerifyNone(t)
}
