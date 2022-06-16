package counter

import (
	"go.lepak.sg/playground/heap"
)

// entries2 is used to implement a max-heap.
// Push and Pop use type parameters in their signatures.
type entries2[E comparable] []*Entry[E]

func (e entries2[_]) Len() int {
	return len(e)
}

func (e entries2[E]) Less(i, j int) bool {
	// yes, the sign is correct
	// see container/heap PriorityQueue example
	return e[i].Count > e[j].Count
}

func (e entries2[_]) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e *entries2[E]) Push(x *Entry[E]) {
	*e = append(*e, x)
}

func (e *entries2[E]) Pop() *Entry[E] {
	x := (*e)[len(*e)-1]
	*e = (*e)[:len(*e)-1]
	return x
}

// TopKAlt is like TopK, but fully generic, as it does not use
// the standard library heap. (The standard library heap.Pop()
// returns any instead of a concrete type.)
func TopKAlt[E comparable](ctr map[E]int, k int) []Entry[E] {
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
	h := entries2[E](heapslice)

	heap.Init[*Entry[E]](&h)

	out := make([]Entry[E], k)
	for i := 0; i < k; i++ {
		entry := heap.Pop[*Entry[E]](&h)
		out[i] = *entry
	}

	return out

}
