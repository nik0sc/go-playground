package slidingwindow

import (
	"math"
	"sync"
	"sync/atomic"

	"golang.org/x/sys/cpu"
)

type entry struct {
	code  uint32
	count int32
}

type ConcurrentCounter[T comparable] struct {
	window []uint32
	evict  func(T)
	_      cpu.CacheLinePad

	fusedHeadLifetime uint64 // (63) head (32) || (31) lifetime (0)

	lock    sync.RWMutex // only protects the maps!
	current map[T]*entry
	codemap map[uint32]T
}

func unpack(fused uint64) (head uint32, lifetime uint32) {
	head = uint32(fused >> 32)
	lifetime = uint32(fused & uint64(math.MaxUint32))
	return
}

func pack(head uint32, lifetime uint32) uint64 {
	return (uint64(head) << 32) | uint64(lifetime)
}

func NewConcurrentCounter[T comparable](
	size int, cardinalityHint int, onEvict func(T),
) *ConcurrentCounter[T] {
	if size < 1 || size > math.MaxUint32 {
		panic("invalid size")
	}

	if cardinalityHint == 0 {
		var zeroT T
		cardinalityHint = guessCardinalityHint(zeroT)
	}

	c := &ConcurrentCounter[T]{
		window:  make([]uint32, size),
		current: make(map[T]*entry, cardinalityHint),
		codemap: make(map[uint32]T, cardinalityHint),
		evict:   onEvict,
	}

	return c
}

func (c *ConcurrentCounter[T]) Get(value T) int {
	c.lock.RLock()
	entryPtr := c.current[value]
	c.lock.RUnlock()

	if entryPtr != nil {
		return int(atomic.LoadInt32(&entryPtr.count))
	}

	return 0
}

func (c *ConcurrentCounter[T]) GetAll() map[T]int {
	var zeroT T
	out := make(map[T]int, guessCardinalityHint(zeroT)) // could be smaller

	c.lock.RLock()

	for value, entryPtr := range c.current {
		if entryPtr == nil {
			continue
		}

		count := atomic.LoadInt32(&entryPtr.count)

		if count == 0 {
			continue
		}

		out[value] = int(count)
	}

	c.lock.RUnlock()

	return out
}

func (c *ConcurrentCounter[T]) Lifetime() int {
	_, lifetime := unpack(atomic.LoadUint64(&c.fusedHeadLifetime))
	return int(lifetime)
}

func (c *ConcurrentCounter[T]) Observe(value T) {
	size := uint32(len(c.window))
	var oldHead, newHead, lifetime uint32

	for {
		packed := atomic.LoadUint64(&c.fusedHeadLifetime)
		oldHead, lifetime = unpack(packed)
		lifetime += 1
		newHead = oldHead + 1
		if newHead >= size {
			newHead = 0
		}

		if atomic.CompareAndSwapUint64(&c.fusedHeadLifetime,
			packed, pack(newHead, lifetime)) {
			break
		}
	}

	var code uint32

	c.lock.RLock()
	entryPtr, ok := c.current[value]
	c.lock.RUnlock()

	if !ok || entryPtr == nil {
		// need to insert
		c.lock.Lock()
		// check again
		entryPtr, ok = c.current[value]
		if !ok || entryPtr == nil {
			code = lifetime
			entryPtr = &entry{
				code:  code,
				count: 1,
			}
			c.current[value] = entryPtr
			c.codemap[code] = value
		} else {
			atomic.AddInt32(&entryPtr.count, 1)
			code = entryPtr.code
		}
		c.lock.Unlock()
	} else {
		atomic.AddInt32(&entryPtr.count, 1)
		code = entryPtr.code
	}

	needEvict := lifetime > size
	if needEvict {
	again:
		evicteeCode := atomic.LoadUint32(&c.window[oldHead])
		c.lock.RLock()
		evictee := c.codemap[evicteeCode]
		evicteeEntryPtr, ok := c.current[evictee]
		c.lock.RUnlock()

		if !ok || evicteeEntryPtr == nil {
			panic("evictee is not in current")
		}
		oldCount := atomic.LoadInt32(&evicteeEntryPtr.count)
		updatedCount := oldCount - 1

		if updatedCount > 0 {
			for {
				// XXX this is wrong
				if !atomic.CompareAndSwapInt32(&evicteeEntryPtr.count, oldCount, updatedCount) {
					goto again
				}
			}
		} else if updatedCount == 0 {
			c.lock.Lock()
			// TODO check count condition on entry again!!

			delete(c.current, evictee)
			delete(c.codemap, evicteeCode)

			c.lock.Unlock()

			c.evict(evictee)
		} else {
			// could this happen normally?
			panic("evictee count was 0")
		}

	}

}
