// Package iterator provides tree iterators for use
// by tree implementations.
package iterator

import (
	"go.lepak.sg/playground/chops"
	"golang.org/x/exp/constraints"
)

// Iterator describes the common interface for all
// iterators in this package.
// Next must always be called before Item, even for
// the first round of iteration.
// If Next returns false, Item must not be called.
// Next may be called any number of times.
// Item may be called any number of times if the
// last call to Next returned true.
// The iterator may be abandoned at any time.
//
// The usual usage of an Iterator is like this:
//	i := someTree.Iterator()
//	for i.Next() {
//		k := i.Item()
//		... do stuff with k, or break ...
//	}
type Iterator[T constraints.Ordered] interface {
	Next() bool
	Item() T
}

// Make sure this Iterator is kept in sync with the one in chops.
var _ chops.Iterator[int] = (Iterator[int])(nil)
