package iterator

import (
	"go.lepak.sg/playground/chops"
	"go.lepak.sg/playground/tree"
	"golang.org/x/exp/constraints"
)

var _ chops.Iterator[int] = (*InOrder[int])(nil)

// InOrder is an iterator object over a binary tree.
// The usage should be pretty familiar:
//	i := someBinaryTree.InOrderIterator()
//	for i.Next() {
//		k := i.Key()
//		... do stuff with k ...
//	}
// The iterator may be abandoned at any time.
// The result of mutating the tree while iterating over it is undefined.
type InOrder[T constraints.Ordered] struct {
	root, at *tree.Node[T]
}

// NewInOrder returns a new InOrderIterator over the tree rooted at root.
// Note: This is meant to be called by other tree implementations.
func NewInOrder[T constraints.Ordered](root *tree.Node[T]) *InOrder[T] {
	return &InOrder[T]{
		root: root,
	}
}

// Next returns true if there is a next node to yield with Key.
// Next must always be called before Key.
func (i *InOrder[T]) Next() bool {
	// https://www.cs.odu.edu/~zeil/cs361/latest/Public/treetraversal/index.html
	if i.at == nil {
		// So, if Next returned false, calling Key will cause a nil pointer dereference,
		// but, if you call Next again, Key will... return the first key in order, again!
		// Implementation detail
		i.at = i.root
		if i.at == nil {
			return false
		}

		for i.at.Left != nil {
			i.at = i.at.Left
		}
		return true
	}

	if i.at.Right != nil {
		i.at = i.at.Right

		for i.at.Left != nil {
			i.at = i.at.Left
		}

		return true
	} else {
		// may not succeed
		var child *tree.Node[T]

		for i.at != nil {
			i.at, child = i.at.Parent, i.at
			if i.at != nil && i.at.Left == child {
				return true
			}
		}

		return false
	}
}

// Item returns the current key of the iterator.
func (i *InOrder[T]) Item() T {
	return i.at.Key
}
