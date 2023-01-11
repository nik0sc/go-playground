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
	}
)

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

			getAll := c.GetAll()
			sum := 0
			for k, v := range getAll {
				assert.NotEqualf(t, 0, v, "k=%d v=0", k)
				sum += v
			}

			assert.Equal(t, 10, sum)
		})
	}

}
