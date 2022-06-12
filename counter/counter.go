package counter

import "container/heap"

func New[S ~[]E, E comparable](list S) map[E]int {
	c := make(map[E]int)

	for _, v := range list {
		c[v]++
	}

	return c
}

type Entry[E comparable] struct {
	Value E
	Count int
}

type Entries[E comparable] []*Entry[E]

var _ heap.Interface = (*Entries[int])(nil)

func (e Entries[_]) Len() int {
	return len(e)
}

func (e Entries[_]) Less(i, j int) bool {
	// yes, the sign is correct
	// see container/heap PriorityQueue example
	return e[i].Count > e[j].Count
}

func (e Entries[_]) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e *Entries[E]) Push(x any) {
	in := x.(*Entry[E]) //...

	*e = append(*e, in)
}

func (e *Entries[E]) Pop() any {
	x := (*e)[len(*e)-1]
	*e = (*e)[:len(*e)-1]
	return x
}

func TopK[E comparable](ctr map[E]int, k int) []Entry[E] {
	var entries Entries[E]

	for el, cnt := range ctr {
		entries = append(entries, &Entry[E]{
			Value: el,
			Count: cnt,
		})
	}

	heap.Init(&entries)

	var out []Entry[E]
	for i := 0; i < k; i++ {
		out = append(out, *heap.Pop(&entries).(*Entry[E]))
	}

	return out
}
