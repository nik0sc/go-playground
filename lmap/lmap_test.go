package lmap

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	type setargs struct {
		k    int
		v    string
		bump bool
	}

	tests := []struct {
		name string
		set  []setargs
		do   func(t *testing.T, lm *LinkedMap[int, string])
	}{
		{
			name: "empty",
			set:  nil,
			do: func(t *testing.T, lm *LinkedMap[int, string]) {
				assert.Equal(t, 0, lm.Len())

				k, v, ok := lm.Head(false)
				assert.Falsef(t, ok, "head: %d->%s", k, v)

				k, v, ok = lm.Tail(false)
				assert.Falsef(t, ok, "tail: %d->%s", k, v)

				lm.ForEach(func(k int, v string) bool {
					t.Errorf("iter callback was called: %d->%s", k, v)
					return false
				})

				i := lm.Iterator()
				if i.Next() {
					k, v = i.Entry()
					t.Errorf("iterator.Next: %d->%s", k, v)
				}
			},
		},
		{
			name: "one",
			set: []setargs{
				{1, "one", false},
			},
			do: func(t *testing.T, lm *LinkedMap[int, string]) {
				assert.Equal(t, 1, lm.Len())

				var k int
				var v string
				var ok bool

				v, ok = lm.Get(1, false)
				assert.True(t, ok)
				assert.Equal(t, "one", v)

				v, ok = lm.Get(2, false)
				assert.Falsef(t, ok, "2->%s", v)

				k, v, ok = lm.Head(false)
				assert.True(t, ok)
				assert.Equal(t, 1, k)
				assert.Equal(t, "one", v)

				k, v, ok = lm.Tail(false)
				assert.True(t, ok)
				assert.Equal(t, 1, k)
				assert.Equal(t, "one", v)

				k, v, ok = lm.Prev(1)
				assert.Falsef(t, ok, "Prev: %d->%s", k, v)

				k, v, ok = lm.Next(1)
				assert.Falsef(t, ok, "Next: %d->%s", k, v)

				times := 0
				lm.ForEach(func(k int, v string) bool {
					assert.Equalf(t, 0, times, "called too many times: %d", times)
					assert.Equal(t, 1, k)
					assert.Equal(t, "one", v)
					times++
					return true
				})

				i := lm.Iterator()
				assert.True(t, i.Next(), "first item")
				k, v = i.Entry()
				assert.Equal(t, 1, k)
				assert.Equal(t, "one", v)
				assert.False(t, i.Next(), "end of iteration")
			},
		},
		{
			name: "two",
			set: []setargs{
				{1, "one", false},
				{2, "two", false},
			},
			do: func(t *testing.T, lm *LinkedMap[int, string]) {
				assert.Equal(t, 2, lm.Len())

				var k int
				var v string
				var ok bool

				v, ok = lm.Get(1, false)
				assert.True(t, ok)
				assert.Equal(t, "one", v)

				v, ok = lm.Get(2, false)
				assert.True(t, ok)
				assert.Equal(t, "two", v)

				k, v, ok = lm.Head(false)
				assert.True(t, ok)
				assert.Equal(t, 1, k)
				assert.Equal(t, "one", v)

				k, v, ok = lm.Tail(false)
				assert.True(t, ok)
				assert.Equal(t, 2, k)
				assert.Equal(t, "two", v)

				k, v, ok = lm.Prev(2)
				assert.True(t, ok)
				assert.Equal(t, 1, k)
				assert.Equal(t, "one", v)

				k, v, ok = lm.Next(1)
				assert.True(t, ok)
				assert.Equal(t, 2, k)
				assert.Equal(t, "two", v)

				times := 0
				lm.ForEach(func(k int, v string) bool {
					switch times {
					case 0:
						assert.Equal(t, 1, k)
						assert.Equal(t, "one", v)
					case 1:
						assert.Equal(t, 2, k)
						assert.Equal(t, "two", v)
					default:
						t.Errorf("called too many times: %d", times)
						return false
					}

					times++
					return true
				})

				i := lm.Iterator()
				assert.True(t, i.Next(), "first item")
				k, v = i.Entry()
				assert.Equal(t, 1, k)
				assert.Equal(t, "one", v)
				assert.True(t, i.Next(), "second item")
				k, v = i.Entry()
				assert.Equal(t, 2, k)
				assert.Equal(t, "two", v)
				assert.False(t, i.Next(), "end of iteration")
			},
		},
		{
			name: "bump",
			set: []setargs{
				{1, "one", false},
				{2, "two", false},
				{3, "three", false},
				{1, "one!", true},
				{2, "two!", false},
				{4, "four", true},
			},
			do: func(t *testing.T, lm *LinkedMap[int, string]) {
				assert.Equal(t, 4, lm.Len())

				var k int
				var v string

				i := lm.Iterator()
				assert.True(t, i.Next(), "first item")
				k, v = i.Entry()
				assert.Equal(t, 2, k)
				assert.Equal(t, "two!", v)
				assert.True(t, i.Next(), "second item")
				k, v = i.Entry()
				assert.Equal(t, 3, k)
				assert.Equal(t, "three", v)
				assert.True(t, i.Next(), "third item")
				k, v = i.Entry()
				assert.Equal(t, 1, k)
				assert.Equal(t, "one!", v)
				assert.True(t, i.Next(), "fourth item")
				k, v = i.Entry()
				assert.Equal(t, 4, k)
				assert.Equal(t, "four", v)
				assert.False(t, i.Next(), "end of iteration")
			},
		},
		{
			name: "delete",
			set: []setargs{
				{1, "", false},
				{2, "", false},
				{3, "", false},
				{4, "", false},
				{5, "", false},
			},
			do: func(t *testing.T, lm *LinkedMap[int, string]) {
				var k int

				assert.True(t, lm.Delete(1))
				assert.True(t, lm.Delete(3))
				assert.True(t, lm.Delete(5))
				assert.False(t, lm.Delete(6))

				assert.Equal(t, 2, lm.Len())

				i := lm.Iterator()
				assert.True(t, i.Next(), "first item")
				k, _ = i.Entry()
				assert.Equal(t, 2, k)
				assert.True(t, i.Next(), "second item")
				k, _ = i.Entry()
				assert.Equal(t, 4, k)
				assert.False(t, i.Next(), "end of iteration")
			},
		},
		{
			name: "pop head",
			set: []setargs{
				{1, "", false},
				{2, "", false},
				{3, "", false},
			},
			do: func(t *testing.T, lm *LinkedMap[int, string]) {
				var k int
				var ok bool

				k, _, ok = lm.Head(true)
				assert.True(t, ok)
				assert.Equal(t, 1, k)
				k, _, ok = lm.Head(true)
				assert.True(t, ok)
				assert.Equal(t, 2, k)
				k, _, ok = lm.Head(true)
				assert.True(t, ok)
				assert.Equal(t, 3, k)
				k, _, ok = lm.Head(true)
				assert.Falsef(t, ok, "had element: %d", k)

				assert.Equal(t, 0, lm.Len())
			},
		},
		{
			name: "pop tail",
			set: []setargs{
				{1, "", false},
				{2, "", false},
				{3, "", false},
			},
			do: func(t *testing.T, lm *LinkedMap[int, string]) {
				var k int
				var ok bool

				k, _, ok = lm.Tail(true)
				assert.True(t, ok)
				assert.Equal(t, 3, k)
				k, _, ok = lm.Tail(true)
				assert.True(t, ok)
				assert.Equal(t, 2, k)
				k, _, ok = lm.Tail(true)
				assert.True(t, ok)
				assert.Equal(t, 1, k)
				k, _, ok = lm.Tail(true)
				assert.Falsef(t, ok, "had element: %d", k)

				assert.Equal(t, 0, lm.Len())
			},
		},
		{
			name: "get bump",
			set: []setargs{
				{1, "", false},
				{2, "", false},
				{3, "", false},
			},
			do: func(t *testing.T, lm *LinkedMap[int, string]) {
				var k int
				var ok bool

				_, ok = lm.Get(1, true)
				assert.True(t, ok)

				i := lm.Iterator()
				assert.True(t, i.Next(), "first item")
				k, _ = i.Entry()
				assert.Equal(t, 2, k)
				assert.True(t, i.Next(), "second item")
				k, _ = i.Entry()
				assert.Equal(t, 3, k)
				assert.True(t, i.Next(), "third item")
				k, _ = i.Entry()
				assert.Equal(t, 1, k)
				assert.False(t, i.Next(), "end of iteration")
			},
		},
		{
			name: "foreach stop",
			set: []setargs{
				{1, "", false},
				{2, "", false},
			},
			do: func(t *testing.T, lm *LinkedMap[int, string]) {
				times := 0
				lm.ForEach(func(k int, _ string) bool {
					times++
					assert.Equal(t, 1, k)
					return false
				})

				assert.Equal(t, 1, times)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := New[int, string]()
			for _, e := range tt.set {
				lm.Set(e.k, e.v, e.bump)
			}

			tt.do(t, lm)
		})
	}
}

// makeCyclic inserts n elements to an empty map
// and links the head and tail together:
//
//	  head─┐                    tail─┐
//	lmap [ 1 <-> 2 <-> 3 <-> ... <-> n ]
//	       ↑                         ↑
//	       └─────────────────────────┘
//
// Any test iterating over a cyclic LinkedMap
// must have an iteration limit, otherwise if
// the test fails, it will run until it times out!
func makeCyclic(t *testing.T, n int) *LinkedMap[int, struct{}] {
	lm := New[int, struct{}]()

	for i := 1; i <= n; i++ {
		lm.Set(i, struct{}{}, false)
	}

	require.Equal(t, 1, lm.head.k)
	require.Equal(t, n, lm.tail.k)
	require.Nil(t, lm.tail.next)
	require.Nil(t, lm.head.prev)

	lm.tail.next = lm.head
	lm.head.prev = lm.tail

	return lm
}

func TestCyclicPanic(t *testing.T) {
	tests := []struct {
		n   int
		max int
	}{
		{
			n:   1,
			max: 1,
		},
		{
			n:   3,
			max: 10,
		},
		{
			n:   100,
			max: 100,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("ForEach/n=%d,max=%d", tt.n, tt.max), func(t *testing.T) {
			lm := makeCyclic(t, tt.n)

			visit := 0
			assert.Panics(t, func() {
				lm.ForEach(func(k int, v struct{}) bool {
					visit++
					require.LessOrEqual(t, visit, tt.max, "too many visits")
					t.Logf("at %d", k)
					return true
				})
			})
		})

		t.Run(fmt.Sprintf("Iterator/n=%d,max=%d", tt.n, tt.max), func(t *testing.T) {
			i := makeCyclic(t, tt.n).Iterator()

			visit := 0
			assert.Panics(t, func() {
				for i.Next() {
					visit++
					require.LessOrEqual(t, visit, tt.max, "too many visits")
					k, _ := i.Entry()
					t.Logf("at %d", k)
				}
			})
		})
	}
}
