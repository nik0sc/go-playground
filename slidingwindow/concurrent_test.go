package slidingwindow

import (
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type implInt interface {
	Get(int) int
	GetAll() map[int]int
	Lifetime() int
	Observe(int)
}

type testSpec struct {
	name       string
	threadsafe bool
	new        func(size, cardinalityHint int, onEvict func(int)) implInt
}

var (
	// Static type assertions
	_ implInt = (*Counter[int])(nil)
	_ implInt = (*LockedCounter[int])(nil)
	_ implInt = (*ConcurrentCounter[int])(nil)

	impls = []testSpec{
		{
			name: "simple",
			new: func(size, cardinalityHint int, onEvict func(int)) implInt {
				return NewCounter(size, cardinalityHint, onEvict)
			},
		},
		{
			name:       "locked",
			threadsafe: true,
			new: func(size, cardinalityHint int, onEvict func(int)) implInt {
				return NewLocked(NewCounter(size, cardinalityHint, onEvict))
			},
		},
		{
			name:       "concurrent",
			threadsafe: true,
			new: func(size, cardinalityHint int, onEvict func(int)) implInt {
				return NewConcurrentCounter(size, cardinalityHint, onEvict)
			},
		},
	}
)

func TestConcurrentCounter(t *testing.T) {
	evicted := -1
	c := NewConcurrentCounter(4, 0, func(value int) {
		evicted = value
	})

	c.Observe(1)
	c.Observe(2)
	c.Observe(3)
	c.Observe(3)

	t.Logf("%+v", c)
	assert.Equal(t, 1, c.Get(1))
	assert.Equal(t, 1, c.Get(2))
	assert.Equal(t, 2, c.Get(3))
	assert.Equal(t, 0, c.Get(4))
	assert.Equal(t, map[int]int{1: 1, 2: 1, 3: 2}, c.GetAll())
	assert.Equal(t, 4, c.Lifetime())
	assert.Equal(t, -1, evicted)

	c.Observe(4)
	t.Logf("%+v", c)
	assert.Equal(t, 0, c.Get(1))
	assert.Equal(t, 1, c.Get(2))
	assert.Equal(t, 2, c.Get(3))
	assert.Equal(t, 1, c.Get(4))
	assert.Equal(t, map[int]int{2: 1, 3: 2, 4: 1}, c.GetAll())
	assert.Equal(t, 5, c.Lifetime())
	assert.Equal(t, 1, evicted)
	evicted = -1

	c.Observe(5)
	t.Logf("%+v", c)
	assert.Equal(t, 0, c.Get(2))
	assert.Equal(t, 2, c.Get(3))
	assert.Equal(t, 1, c.Get(4))
	assert.Equal(t, 1, c.Get(5))
	assert.Equal(t, map[int]int{3: 2, 4: 1, 5: 1}, c.GetAll())
	assert.Equal(t, 6, c.Lifetime())
	assert.Equal(t, 2, evicted)
	evicted = -1

	c.Observe(5)
	t.Logf("%+v", c)
	assert.Equal(t, 1, c.Get(3))
	assert.Equal(t, 1, c.Get(4))
	assert.Equal(t, 2, c.Get(5))
	assert.Equal(t, map[int]int{3: 1, 4: 1, 5: 2}, c.GetAll())
	assert.Equal(t, 7, c.Lifetime())
	assert.Equal(t, -1, evicted)

	c.Observe(5)
	t.Logf("%+v", c)
	assert.Equal(t, 0, c.Get(3))
	assert.Equal(t, 1, c.Get(4))
	assert.Equal(t, 3, c.Get(5))
	assert.Equal(t, map[int]int{4: 1, 5: 3}, c.GetAll())
	assert.Equal(t, 8, c.Lifetime())
	assert.Equal(t, 3, evicted)
	evicted = -1

	c.Observe(5)
	t.Logf("%+v", c)
	assert.Equal(t, 0, c.Get(3))
	assert.Equal(t, 0, c.Get(4))
	assert.Equal(t, 4, c.Get(5))
	assert.Equal(t, map[int]int{5: 4}, c.GetAll())
	assert.Equal(t, 9, c.Lifetime())
	assert.Equal(t, 4, evicted)
	evicted = -1

	c.Observe(5)
	t.Logf("%+v", c)
	assert.Equal(t, 0, c.Get(3))
	assert.Equal(t, 0, c.Get(4))
	assert.Equal(t, 4, c.Get(5))
	assert.Equal(t, map[int]int{5: 4}, c.GetAll())
	assert.Equal(t, 10, c.Lifetime())
	assert.Equal(t, -1, evicted)
}

func TestConcurrentObserveInterleaved(t *testing.T) {
	for _, tt := range impls {
		if !tt.threadsafe {
			continue
		}

		implNew := tt.new

		t.Run(tt.name, func(t *testing.T) {
			random := rand.New(rand.NewSource(seed))
			const (
				writers = 4
				card    = 10
				times   = 1_000_000
			)

			fudgedTimes := int(float64(times) / float64(writers) * 1.2)

			values := make([][]int, writers)
			for i := range values {
				values[i] = make([]int, fudgedTimes)
			}

			for i := 0; i < fudgedTimes; i++ {
				for j := 0; j < writers; j++ {
					values[j][i] = random.Intn(card)
				}
			}

			c := implNew(10, 0, nil)
			barrier := make(chan struct{})
			stop := make(chan struct{})
			var wg sync.WaitGroup
			wg.Add(writers)

			for i := 0; i < writers; i++ {
				writerValues := values[i]
				go func() {
					defer wg.Done()
					<-barrier

					for _, value := range writerValues {
						select {
						case <-stop:
							return
						default:
						}

						c.Observe(value)
					}
				}()
			}

			close(barrier)

			for c.Lifetime() <= times {
			}

			close(stop)
			wg.Wait()

			t.Logf("%+v", c)
			t.Log(c.GetAll())
			t.Log(c.Lifetime())

		})
	}

}
