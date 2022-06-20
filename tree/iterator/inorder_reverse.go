package iterator

import (
	"go.lepak.sg/playground/chops"
	"go.lepak.sg/playground/tree"
	"golang.org/x/exp/constraints"
)

var _ chops.Iterator[int] = (*InOrderReverse[int])(nil)

// InOrderReverse is an iterator object over a binary tree.
// Iteration starts from the *largest* element and runs to
// the *smallest* element.
// The usage should be pretty familiar:
//	i := someBinaryTree.InOrderReverseIterator()
//	for i.Next() {
//		k := i.Item()
//		... do stuff with k ...
//	}
// The iterator may be abandoned at any time.
// The result of mutating the tree while iterating over it is undefined.
type InOrderReverse[T constraints.Ordered] struct {
	root, at *tree.Node[T]
}

// NewInOrderReverse returns a new InOrderReverse iterator over the tree
// rooted at root.
// Note: This is meant to be called by other tree implementations.
func NewInOrderReverse[T constraints.Ordered](
	root *tree.Node[T]) *InOrderReverse[T] {
	return &InOrderReverse[T]{
		root: root,
	}
}

// Next returns true if there is a next node to yield with Key.
// Next must always be called before Key.
func (i *InOrderReverse[T]) Next() bool {
	// Basically InOrder.Next but left and right are flipped.
	if i == nil {
		return false
	}

	if i.at == nil {
		i.at = i.root
		if i.at == nil {
			return false
		}

		for i.at.Right != nil {
			i.at = i.at.Right
		}
		return true
	}

	if i.at.Left != nil {
		i.at = i.at.Left

		for i.at.Right != nil {
			i.at = i.at.Right
		}

		return true
	}

	var child *tree.Node[T]

	for i.at != nil {
		i.at, child = i.at.Parent, i.at
		if i.at != nil && i.at.Right == child {
			return true
		}
	}

	return false
}

// Item returns the current key of the iterator.
func (i *InOrderReverse[T]) Item() T {
	return i.at.Key
}
