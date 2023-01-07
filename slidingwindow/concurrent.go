package slidingwindow

import (
	"fmt"
	"math"
	"reflect"
	"runtime"
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

func (c *ConcurrentCounter[T]) String() string {
	c.lock.Lock()
	defer c.lock.Unlock()

	funcname := runtime.FuncForPC(reflect.ValueOf(c.evict).Pointer()).Name()

	head, lifetime := unpack(c.fusedHeadLifetime)

	current := make(map[T]entry)
	for value, e := range c.current {
		current[value] = *e
	}

	return fmt.Sprintf("&{window:%v evictfuncname:%s head:%d lifetime:%d "+
		"current:%+v codemap:%+v}",
		c.window, funcname, head, lifetime, current, c.codemap)
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
	var oldHead, lifetime, evicteeCode, code uint32
	var evictee T
	var evicteeEntryPtr *entry
	var evicteeReadOk bool

	for {
		packed := atomic.LoadUint64(&c.fusedHeadLifetime)
		oldHead, lifetime = unpack(packed)
		lifetime += 1
		newHead := oldHead + 1
		if newHead >= size {
			newHead = 0
		}

		if atomic.CompareAndSwapUint64(&c.fusedHeadLifetime,
			packed, pack(newHead, lifetime)) {
			break
		}
	}

	evicteeCode = atomic.LoadUint32(&c.window[oldHead])

	c.lock.RLock()
	entryPtr, ok := c.current[value]
	evictee = c.codemap[evicteeCode]
	evicteeEntryPtr, evicteeReadOk = c.current[evictee]
	c.lock.RUnlock()

	if !ok || entryPtr == nil {
		// need to insert
		c.lock.Lock()
		// check again
		entryPtr, ok = c.current[value]
		if !ok || entryPtr == nil {
			code = lifetime
			entryPtr = &entry{
				code:  lifetime, //code,
				count: 1,
			}
			c.current[value] = entryPtr
			c.codemap[lifetime] = value
		} else {
			atomic.AddInt32(&entryPtr.count, 1)
			code = entryPtr.code
		}
		c.lock.Unlock()
	} else {
		atomic.AddInt32(&entryPtr.count, 1)
		code = entryPtr.code
	}

	atomic.StoreUint32(&c.window[oldHead], code)

	needEvict := lifetime > size
	if needEvict {

		if !evicteeReadOk || evicteeEntryPtr == nil {
			// Already evicted 1->0 by another goroutine
			// Do nothing
			return
			// panic("evictee is not in current")
		}
		var oldCount, updatedCount int32

		for {
			oldCount = atomic.LoadInt32(&evicteeEntryPtr.count)
			updatedCount = oldCount - 1
			if atomic.CompareAndSwapInt32(&evicteeEntryPtr.count, oldCount, updatedCount) {
				break
			}
		}

		if updatedCount == 0 {
			c.lock.Lock()
			// do we need to check count condition on entry again?

			delete(c.current, evictee)
			delete(c.codemap, evicteeCode)

			c.lock.Unlock()

			if c.evict != nil {
				c.evict(evictee)
			}
		} else if updatedCount < 0 {
			// XXX could this happen normally?
			// panic("evictee count was 0")
			return
		}
	}
}
