package slidingwindow

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCounter(t *testing.T) {
	evicted := -1
	c := NewCounter(4, 0, func(value int) {
		evicted = value
	})

	c.Observe(1)
	c.Observe(2)
	c.Observe(3)
	c.Observe(3)

	assert.Equal(t, 1, c.Get(1))
	assert.Equal(t, 1, c.Get(2))
	assert.Equal(t, 2, c.Get(3))
	assert.Equal(t, 0, c.Get(4))
	assert.Equal(t, map[int]int{1: 1, 2: 1, 3: 2}, c.GetAll())
	assert.Equal(t, 4, c.Lifetime())
	assert.Equal(t, -1, evicted)

	c.Observe(4)
	assert.Equal(t, 0, c.Get(1))
	assert.Equal(t, 1, c.Get(2))
	assert.Equal(t, 2, c.Get(3))
	assert.Equal(t, 1, c.Get(4))
	assert.Equal(t, map[int]int{2: 1, 3: 2, 4: 1}, c.GetAll())
	assert.Equal(t, 5, c.Lifetime())
	assert.Equal(t, 1, evicted)
	evicted = -1

	c.Observe(5)
	assert.Equal(t, 0, c.Get(2))
	assert.Equal(t, 2, c.Get(3))
	assert.Equal(t, 1, c.Get(4))
	assert.Equal(t, 1, c.Get(5))
	assert.Equal(t, map[int]int{3: 2, 4: 1, 5: 1}, c.GetAll())
	assert.Equal(t, 6, c.Lifetime())
	assert.Equal(t, 2, evicted)
	evicted = -1

	c.Observe(5)
	assert.Equal(t, 1, c.Get(3))
	assert.Equal(t, 1, c.Get(4))
	assert.Equal(t, 2, c.Get(5))
	assert.Equal(t, map[int]int{3: 1, 4: 1, 5: 2}, c.GetAll())
	assert.Equal(t, 7, c.Lifetime())
	assert.Equal(t, -1, evicted)

	c.Observe(5)
	assert.Equal(t, 0, c.Get(3))
	assert.Equal(t, 1, c.Get(4))
	assert.Equal(t, 3, c.Get(5))
	assert.Equal(t, map[int]int{4: 1, 5: 3}, c.GetAll())
	assert.Equal(t, 8, c.Lifetime())
	assert.Equal(t, 3, evicted)
	evicted = -1

	c.Observe(5)
	assert.Equal(t, 0, c.Get(3))
	assert.Equal(t, 0, c.Get(4))
	assert.Equal(t, 4, c.Get(5))
	assert.Equal(t, map[int]int{5: 4}, c.GetAll())
	assert.Equal(t, 9, c.Lifetime())
	assert.Equal(t, 4, evicted)
	evicted = -1

	c.Observe(5)
	assert.Equal(t, 0, c.Get(3))
	assert.Equal(t, 0, c.Get(4))
	assert.Equal(t, 4, c.Get(5))
	assert.Equal(t, map[int]int{5: 4}, c.GetAll())
	assert.Equal(t, 10, c.Lifetime())
	assert.Equal(t, -1, evicted)
}

func TestCounter_NoSpuriousEviction(t *testing.T) {
	c := NewCounter(2, 0, func(value int) {
		t.Fatalf("unexpected eviction of %d", value)
	})

	c.Observe(1)
	c.Observe(2)
	c.Observe(1)
	c.Observe(2)
}

const (
	seed = 58913
)

func BenchmarkCounter_SmallWindow(b *testing.B) {
	const (
		reps int = 1e8
		size int = 10
	)

	cardinalities := []int{0, 10, 100, 1e3, 1e4}
	intervals := []int64{math.MaxInt64, 1e3, 10}

	for _, cardinality := range cardinalities {
		for _, interval := range intervals {
			b.Run(makeCounterBenchmark(reps, size, cardinality, interval))
		}
	}
}

func makeCounterBenchmark(
	reps, size, cardinalityHint int, randInterval int64,
) (string, func(b *testing.B)) {
	name := fmt.Sprintf("reps=%d/size=%d/card=%d/intv=%d",
		reps, size, cardinalityHint, randInterval)
	bench := func(b *testing.B) {
		b.StopTimer()
		b.ReportAllocs()

		random := rand.New(rand.NewSource(seed))

		b.StartTimer()
		c := NewCounter[int64](size, cardinalityHint, nil)
		for i := 0; i < reps; i++ {
			c.Observe(random.Int63n(randInterval))
			// too noisy
			// if i != 0 && i%1e6 == 0 {
			// 	b.Log("REP:", i)
			// }
		}
		b.StopTimer()

		// very unsafe magic: see runtime/map.go
		type hmap struct {
			count     int // # live cells == size of map.  Must be first (used by len() builtin)
			flags     uint8
			B         uint8  // log_2 of # of buckets (can hold up to loadFactor * 2^B items)
			noverflow uint16 // approximate number of overflow buckets; see incrnoverflow for details
			hash0     uint32 // hash seed
		}
		b.Log("len of map:", len(c.current))
		header := (*hmap)(reflect.ValueOf(c.current).UnsafePointer())
		b.ReportMetric(float64(b.N), "log2(buckets)/op")
		b.Logf("%+v\n", header)
	}

	return name, bench
}
