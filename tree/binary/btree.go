package binary

import (
	"go.lepak.sg/playground/chops"
	"go.lepak.sg/playground/tree"
	"go.lepak.sg/playground/tree/iterator"
	"golang.org/x/exp/constraints"
)

// Tree is a binary search tree. It is safe for concurrent reads
// (searching, iterating, etc) but not for concurrent reads and writes
// (inserting).
//
// The zero Tree may be used immediately. Tree should not be passed
// around as a value (ie. just use &Tree{} when creating one).
//
// This tree implementation does not support removal. It is also not
// self-balancing.
//
// Invariants:
//  - At any node N in the tree, all node keys in the subtree rooted at N.Left
//    will be less than N.Key
//  - At any node N in the tree, all node keys in the subtree rooted at N.Right
//    will be greater than N.Key
//  - For every possible key, there will be at most one node with that key
//    in the tree (No duplicates allowed)
type Tree[T constraints.Ordered] struct {
	// the tree is rooted here.
	// don't return nodes directly - client could mutate data or children!
	root *tree.Node[T]
	// I lied - after the first insertion, Tree may be passed around by value
}

// Instead of using constraints.Ordered, I also considered using
// interface[T any] { CompareTo(T) int } (forgive the syntax).
// This allows T to mutate, for example if we defined:
//	type IntPtr *int
// and then implemented:
//	func (ip IntPtr) CompareTo(ip2 IntPtr) int {
//		return (*ip2)-(*ip)
//	}
// client code could mutate *IntPtr at any time, ruining our tree invariants.
// This is probably not ideal.

// Contains searches for k in the tree and returns true if it was found.
func (t *Tree[T]) Contains(k T) bool {
	n := t.root

	for n != nil {
		switch tree.Compare(k, n.Key) {
		case tree.Less:
			n = n.Left
		case tree.Greater:
			n = n.Right
		case tree.Equal:
			return true
		default:
			panic("unreachable")
		}
	}

	return false
}

// Less ...
func (t *Tree[T]) Less(k T) T {
	// n := t.root

	// for n != nil {

	// }

	panic("todo")
}

// Insert inserts k into the binary tree.
// If k is already in the tree, Insert returns false.
func (t *Tree[T]) Insert(k T) bool {
	if t.root == nil {
		t.root = tree.NodeOf(k)
		return true
	}

	n, p := t.root, (*tree.Node[T])(nil)
	var cmp tree.Order

	for n != nil {
		cmp = tree.Compare(k, n.Key)
		switch cmp {
		case tree.Less:
			n, p = n.Left, n
		case tree.Greater:
			n, p = n.Right, n
		case tree.Equal:
			return false
		default:
			panic("unreachable")
		}
	}

	newnode := tree.NodeOf(k)
	newnode.Parent = p

	switch cmp {
	case tree.Less:
		if p.Left != nil {
			panic("impossible")
		}
		p.Left = newnode
	case tree.Greater:
		if p.Right != nil {
			panic("impossible")
		}
		p.Right = newnode
	default:
		panic("unreachable")
	}

	return true
}

// InOrder applies f to each key in the tree in-order.
// If f returns false, the iteration is stopped early.
func (t *Tree[T]) InOrder(f func(k T) bool) {
	t.visitInOrder(t.root, f)
}

func (t *Tree[T]) visitInOrder(n *tree.Node[T], f func(k T) bool) bool {
	// Classic recursive in-order iteration.
	// Compare this to iterator.InOrder which is not recursive
	if n.Left != nil {
		if !t.visitInOrder(n.Left, f) {
			return false
		}
	}

	if !f(n.Key) {
		return false
	}

	if n.Right != nil {
		if !t.visitInOrder(n.Right, f) {
			return false
		}
	}

	return true
}

// InOrderCoroutine starts coroutine-style in-order iteration.
// The usage is as follows:
//
//	co := t.InOrderCoroutine()
//	for k := range co.Items() {
//		... do stuff with k ...
//		if k meets some stopping condition {
//			co.Stop()
//		}
//	}
//
// Note: InOrderCoroutine starts a goroutine, which exits when either
// Stop() is called or the iteration is finished.
// If you follow the usage above, the goroutine will not live beyond
// the end of the for-range loop.
func (t *Tree[T]) InOrderCoroutine() chops.CoIterator[T] {
	// ?? Why can't T be inferred for CoIterate ??
	return chops.CoIterate[T](t.InOrderIterator())
}

// InOrderIterator returns an iterator object that yields
// keys from the tree in-order.
func (t *Tree[T]) InOrderIterator() *iterator.InOrder[T] {
	return iterator.NewInOrder(t.root)
}
