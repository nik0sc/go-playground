package slidingwindow

import (
	"reflect"

	"golang.org/x/exp/maps"
)

// Counter is a sliding window-based counter.
// The main interaction with Counter is through its Observe method,
// which records one observation of a value.
// Counter has a size, which limits how many observations are kept:
// if the size is 10, then the 11th observation will replace
// the 1st observation from the Counter.
//
// NewCounter is not safe for concurrent use. If you want to share it
// across multiple goroutines, it must be protected by a mutex. In this case,
// the cardinalityHint should be set to a sufficiently large value, to minimize
// time spent in Observe holding the mutex.
// See NewCounter for more details on cardinalityHint.
//
// T may be any comparable type, but consider keeping its size small:
// ints, constant strings, and small structs at or under 16 bytes are fine.
// Dynamically allocated strings may be kept around for longer than desired,
// since the window will keep the string alive.
// Floats have strange equality properties (namely with NaN and epsilon),
// so consider using a fixed-point representation of the number instead.
// See https://github.com/golang/go/issues/56351.
//
// Counter is not intended for use in calculating averages,
// although that can be implemented with the GetAll method.
type Counter[T comparable] struct {
	window   []T
	head     int
	lifetime int
	current  map[T]int
	evict    func(T)
}

// NewCounter creates a new sliding window-based counter with the given size.
// NewCounter is not safe for concurrent use.
//
// If onEvict is not nil, then when the last occurrence of a previously
// observed value is evicted from the window, onEvict will be called with the
// evicted value. onEvict will run in the same goroutine that called Observe,
// so consider doing any long-running work in another goroutine.
//
// The cardinalityHint is your guess of how many distinct values will ever be
// seen by Counter. If you are not sure, pass 0 and NewCounter will guess for
// you. This hint is used to size the internal map of values to counts.
// If cardinalityHint is too small, more time will be spent resizing the map
// and resolving hash collisions. If cardinalityHint is too large, more memory
// will be used for no performance gain.
func NewCounter[T comparable](
	size int, cardinalityHint int, onEvict func(T),
) *Counter[T] {
	if size < 1 {
		panic("invalid size")
	}

	if cardinalityHint == 0 {
		var zeroT T
		cardinalityHint = guessCardinalityHint(zeroT)
	}

	c := &Counter[T]{
		window:  make([]T, size),
		current: make(map[T]int, cardinalityHint),
		evict:   onEvict,
	}

	return c
}

func guessCardinalityHint(T any) int {
	var sz uintptr
	const maxSize = ^uintptr(0)

	rt := reflect.TypeOf(T)
	switch rt.Kind() {
	// T is comparable, which excludes some possibilities
	case reflect.Struct:
		// try and exclude padding
		for i := 0; i < rt.NumField(); i++ {
			ft := rt.Field(i).Type
			if ft.Kind() == reflect.String {
				sz = maxSize
				break
			}
			sz += ft.Size()
		}
	case reflect.String:
		// basically infinite cardinality
		sz = maxSize
	default:
		sz = rt.Size()
	}

	switch {
	case sz <= 4:
		return 256
	case sz <= 8:
		return 1024
	case sz <= 16:
		return 2048
	default:
		return 4096
	}
}

// Get returns the value's count, which may be 0, but never larger than the
// window size.
func (c *Counter[T]) Get(value T) int {
	return c.current[value]
}

// GetAll returns a map of all observed values in the window to their counts.
func (c *Counter[T]) GetAll() map[T]int {
	return maps.Clone(c.current)
}

// Lifetime returns the lifetime count of observations.
func (c *Counter[T]) Lifetime() int {
	return c.lifetime
}

// Observe makes an observation of a value.
func (c *Counter[T]) Observe(value T) {
	size := len(c.window)

	needEvict := c.lifetime >= size
	if needEvict {
		evictee := c.window[c.head]
		updatedCount := c.current[evictee] - 1

		if updatedCount > 0 {
			c.current[evictee] = updatedCount
		} else if updatedCount == 0 {
			delete(c.current, evictee)
			if c.evict != nil {
				c.evict(evictee)
			}
		} else {
			// this implies that either evictee wasn't in current
			// or evictee was not removed from current when it hit 0 previously
			// which are both bad
			panic("evictee count was 0")
		}
	}

	c.window[c.head] = value
	c.lifetime += 1
	c.head += 1
	if c.head >= size {
		c.head = 0
	}
	c.current[value] += 1
}
