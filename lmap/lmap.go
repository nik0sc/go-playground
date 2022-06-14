package lmap

const (
	flagIterating = 1 << iota
)

// LinkedMap is a map combined with a linked list. It preserves insertion order and therefore
// iteration order as well.
// LinkedMap is not safe for concurrent use.
type LinkedMap[K comparable, V any] struct {
	m          map[K]*entryb[K, V]
	head, tail *entryb[K, V]
	flags      int
}

type entryb[K comparable, V any] struct {
	k          K
	v          V
	prev, next *entryb[K, V]
}

// Not used, but may be called for debug purposes
// todo: return cycle start and size too
func (e *entryb[_, _]) cycle() bool {
	tortoise, hare := e, e

	for hare != nil && hare.next != nil {
		tortoise = tortoise.next
		hare = hare.next.next
		if tortoise == hare {
			return true
		}
	}

	return false
}

// New returns a pointer to a new LinkedMap.
func New[K comparable, V any]() *LinkedMap[K, V] {
	return &LinkedMap[K, V]{
		m: make(map[K]*entryb[K, V]),
	}
}

func (l *LinkedMap[_, _]) assertNotIterating() {
	if l.flags&flagIterating > 0 {
		panic("about to mutate or iterate over linked list " +
			"while already iterating over it")
	}
}

func (l *LinkedMap[K, V]) remove(e *entryb[K, V]) {
	if e == nil {
		panic("nil entry")
	}

	if l.head == nil || l.tail == nil {
		panic("nil head or tail")
	}

	l.assertNotIterating()

	if e.prev != nil {
		e.prev.next = e.next
	} else {
		if l.head != e {
			panic("entry has no previous node but it is not the head")
		}
		l.head = e.next
	}

	if e.next != nil {
		e.next.prev = e.prev
	} else {
		if l.tail != e {
			panic("entry has no next node but it is not the tail")
		}
		l.tail = e.prev
	}
}

func (l *LinkedMap[K, V]) push(e *entryb[K, V]) {
	if e == nil {
		panic("nil entry")
	}

	l.assertNotIterating()

	if l.head == nil && l.tail == nil {
		l.head, l.tail = e, e
		return
	}

	e.prev = l.tail
	l.tail.next = e

	e.next = nil
	l.tail = e
}

// Get behaves like the map access `v, ok := l[k]`. If bump is true and k is in the map,
// k is moved to the tail of the list, as if it were removed and added back to the map.
func (l *LinkedMap[K, V]) Get(k K, bump bool) (v V, ok bool) {
	e, ok := l.m[k]
	if !ok {
		return
	}

	if bump {
		l.remove(e)
		l.push(e)
	}

	return e.v, true
}

// Set behaves like the map set `l[k] = v`. If bumpOnExist is true and k is in the map,
// k is moved to the tail of the list, as if it were removed and added back into the map.
// Otherwise, if k is not in the map, it is appended to the tail of the list.
func (l *LinkedMap[K, V]) Set(k K, v V, bumpOnExist bool) {
	l.assertNotIterating()

	e, exist := l.m[k]
	if exist {
		if e.k != k {
			panic("entry key does not match map key")
		}

		e.v = v
		if bumpOnExist {
			l.remove(e)
			l.push(e)
		}
	} else {
		e = &entryb[K, V]{
			k: k,
			v: v,
		}

		l.m[k] = e

		l.push(e)
	}
}

// Delete behaves like `delete(l, k)`. If the key was not found, ok will be false.
func (l *LinkedMap[K, _]) Delete(k K) (ok bool) {
	e, ok := l.m[k]
	if !ok {
		return
	}

	l.remove(e)
	delete(l.m, k)

	return
}

// Iter allows ordered iteration over the map in the same vein as `for k, v := range l {}`.
// The function f is called for every key-value pair in order. If f returns false at any
// iteration, the iteration process is stopped early.
//
// The result of modifying the map while iterating over the map is undefined.
func (l *LinkedMap[K, V]) Iter(f func(k K, v V) bool) {
	if l.head == nil {
		return
	}

	l.assertNotIterating()
	l.flags |= flagIterating

	hare := l.head.next

	for e := l.head; e != nil; e = e.next {
		if e == hare {
			// bug in the map, not in the caller
			panic("cycle detected, iteration will not end")
		}

		if !f(e.k, e.v) {
			break
		}

		if hare != nil && hare.next != nil {
			hare = hare.next.next
		} else {
			// hare has reached the end, iteration will too
			// e will never be nil
			hare = nil
		}
	}

	l.flags &^= flagIterating
}

// Len behaves like `len(l)`. This is a constant-time operation.
func (l *LinkedMap[_, _]) Len() int {
	return len(l.m)
}

// Head returns the head element of the linked list. If pop is true, the head element
// is also removed from the map and list. If ok is false, no element was found.
func (l *LinkedMap[K, V]) Head(pop bool) (k K, v V, ok bool) {
	if l.head == nil {
		return
	}

	k, v, ok = l.head.k, l.head.v, true

	if pop {
		l.remove(l.head)
		delete(l.m, k)
	}

	return
}

// Tail returns the tail element of the linked list. If pop is true, the tail element
// is also removed from the map and list. If ok is false, no element was found.
func (l *LinkedMap[K, V]) Tail(pop bool) (k K, v V, ok bool) {
	if l.tail == nil {
		return
	}

	k, v, ok = l.tail.k, l.tail.v, true

	if pop {
		l.remove(l.tail)
		delete(l.m, k)
	}

	return
}

// Next returns the key and value of element after k in the linked map.
// If k is not in the linked map, or k is already the last element,
// ok is false, and kn and vn are their zero values.
// This may also be used to iterate over the map.
func (l *LinkedMap[K, V]) Next(k K) (kn K, vn V, ok bool) {
	e, ok := l.m[k]
	if !ok || e.next == nil {
		ok = false
		return
	}

	return e.next.k, e.next.v, true
}

// Next returns the key and value of element before k in the linked map.
// If k is not in the linked map, or k is already the first element,
// ok is false, and kn and vn are their zero values.
func (l *LinkedMap[K, V]) Prev(k K) (kp K, vp V, ok bool) {
	e, ok := l.m[k]
	if !ok || e.prev == nil {
		ok = false
		return
	}

	return e.prev.k, e.prev.v, true
}

func (l *LinkedMap[K, V]) Iterator() Iterator[K, V] {
	return Iterator[K, V]{
		head: l.head,
	}
}

type Iterator[K comparable, V any] struct {
	head, cur, hare *entryb[K, V]
}

// Next advances the iterator and returns whether there is anything
// to be read with Item.
// Next must be called before Item.
func (i *Iterator[K, V]) Next() bool {
	if i.cur == nil {
		if i.head == nil {
			return false
		}
		i.cur = i.head
		i.hare = i.head
	} else if i.cur.next == nil {
		return false
	} else {
		i.cur = i.cur.next
	}

	if i.hare != nil && i.hare.next != nil {
		i.hare = i.hare.next.next
	} else {
		// hare has reached the end, iteration will too
		i.hare = nil
	}

	if i.cur == i.hare {
		panic("cycle detected, iteration will not end")
	}

	return true
}

// Item returns the current key and value of the iterator.
func (i *Iterator[K, V]) Item() (k K, v V) {
	return i.cur.k, i.cur.v
}
