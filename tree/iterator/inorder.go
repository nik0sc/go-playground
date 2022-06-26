package iterator

import (
	"go.lepak.sg/playground/chops"
	"go.lepak.sg/playground/tree"
	"golang.org/x/exp/constraints"
)

var _ chops.Iterator[int] = (*InOrder[int, any])(nil)

// InOrder is an iterator object over a binary tree.
// The usage should be pretty familiar:
//	i := someBinaryTree.InOrderIterator()
//	for i.Next() {
//		k := i.Item()
//		... do stuff with k ...
//	}
// The iterator may be abandoned at any time.
// The result of mutating the tree while iterating over it is undefined.
type InOrder[T constraints.Ordered, X any] struct {
	root, at *tree.Node[T, X]
}

// InOrder has the type parameter X for the tree.Node.Extra field,
// this is an implementation detail and it really shouldn't be exposed to client code.
// Is there some way to hide this detail?

// NewInOrder returns a new InOrder iterator over the tree rooted at root.
// Note: This is meant to be called by other tree implementations.
func NewInOrder[T constraints.Ordered, X any](root *tree.Node[T, X]) *InOrder[T, X] {
	return &InOrder[T, X]{
		root: root,
	}
}

// Next returns true if there is a next node to yield with Key.
// Next must always be called before Key.
func (i *InOrder[T, X]) Next() bool {
	if i == nil {
		return false
	}

	// Adapted from the C++ iterator at
	// https://www.cs.odu.edu/~zeil/cs361/latest/Public/treetraversal/index.html

	// If this is the first iteration, find the smallest node
	if i.at == nil {
		// So, if Next returned false, calling Key will cause a nil pointer dereference,
		// but, if you call Next again, Key will... return the first key in order, again!
		// Implementation detail
		i.at = i.root
		if i.at == nil {
			return false
		}

		// The start will be all the way on the left, from the root.
		//   -> a
		//     / \
		//    b   c
		//   / \
		//  d   e
		// The tree is rooted at a, so "all the way left" leads to d.
		for i.at.Left != nil {
			i.at = i.at.Left
		}
		return true
	}

	// If there is a subtree on the right, the next node
	// is going to be in it, so check there first
	// Why? Suppose that the current node has both a parent and right subtree,
	// and that the current node is in its parent's left subtree:
	//      a
	//     / \
	// -> b   c
	//   / \
	//  d   e
	//
	// (b < e) as e is on b's right.
	// (b < a) and (e < a) as b and e are both on a's left.
	// Combined, (b < e < a). So e should go next.
	if i.at.Right != nil {
		i.at = i.at.Right

		// Then just head as far left as possible
		// after entering the right subtree.
		for i.at.Left != nil {
			i.at = i.at.Left
		}

		return true
	}

	// No right subtree, so check the parent instead. May not succeed
	var child *tree.Node[T, X]

	// Go up the chain of parents until we are at a node where we
	// passed through a left link.
	//      a
	//     / \
	//    b   c
	//   / \
	//  d   e <-
	//
	// e's parent is b, but e was on the right of b, so keep going.
	// b's parent is a, and b was on the left of a, so a is our next node.
	for i.at != nil {
		i.at, child = i.at.Parent, i.at
		if i.at != nil && i.at.Left == child {
			return true
		}
	}

	// If we run out of parents, and we never passed through a
	// left link, we were at the last node.
	//      a
	//     / \
	//    b   c <-
	//   / \
	//  d   e
	//
	// c's parent is a, but c was on the right of a, so keep going.
	// a has no parent, and we're out of nodes.
	return false

}

// Item returns the current key of the iterator.
func (i *InOrder[T, _]) Item() T {
	return i.at.Key
}
