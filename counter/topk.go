package counter

import "container/heap"

// Entry represents an element-count pair.
type Entry[E comparable] struct {
	Element E
	Count   int
}

// entries is used to implement a max-heap.
type entries[E comparable] []*Entry[E]

var _ heap.Interface = (*entries[int])(nil)

func (e entries[_]) Len() int {
	return len(e)
}

func (e entries[E]) Less(i, j int) bool {
	// yes, the sign is correct
	// see container/heap PriorityQueue example
	return e[i].Count > e[j].Count
}

func (e entries[_]) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e *entries[E]) Push(x any) {
	in := x.(*Entry[E]) //...

	*e = append(*e, in)
}

func (e *entries[E]) Pop() any {
	x := (*e)[len(*e)-1]
	*e = (*e)[:len(*e)-1]
	return x
}

// entriesMin is used to implement a min-heap.
type entriesMin[E comparable] struct {
	entries[E]
}

var _ heap.Interface = (*entriesMin[int])(nil)

func (e entriesMin[_]) Less(i, j int) bool {
	return e.entries[i].Count < e.entries[j].Count
}

// heapk creates either a min- or max-heap from the element-count pairs
// in the counter, then pops off k elements and returns them.
func heapk[E comparable](ctr map[E]int, k int, max bool) []Entry[E] {
	// Is it possible to have a type parameter H
	// that *entries[E] and *entriesMin[E] would conform to?
	// The H must be a superset of heap.Interface
	// and the type must be ~*[]Entry[E]

	if k == 0 {
		return []Entry[E]{}
	} else if k > len(ctr) {
		panic("k is larger than number of elements in ctr")
	} else if k < 0 {
		panic("k is negative")
	}

	heapslice := make([]*Entry[E], len(ctr))
	i := 0
	for el, cnt := range ctr {
		heapslice[i] = &Entry[E]{
			Element: el,
			Count:   cnt,
		}
		i++
	}

	var hptr heap.Interface

	if max {
		h := entries[E](heapslice)
		hptr = &h
	} else {
		h := entriesMin[E]{entries: heapslice}
		hptr = &h
	}

	heap.Init(hptr)

	out := make([]Entry[E], k)
	for i := 0; i < k; i++ {
		entry := heap.Pop(hptr).(*Entry[E])
		out[i] = *entry
	}

	return out
}

// TopK returns the k most-frequent elements from the counter.
// The returned entries are in descending order of frequency.
// If two elements have the same count, their relative order in
// the returned slice is undefined, however they will be after
// all elements that occur more frequently.
func TopK[E comparable](ctr map[E]int, k int) []Entry[E] {
	return heapk(ctr, k, true)
}

// BottomK returns the k least-frequent elements from the counter.
// The returned entries are in ascending order of frequency.
// If two elements have the same count, their relative order in
// the returned slice is undefined, however they will be after
// all elements that occur less frequently.
func BottomK[E comparable](ctr map[E]int, k int) []Entry[E] {
	return heapk(ctr, k, false)
}
