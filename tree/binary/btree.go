package binary

import (
	"fmt"
	"strings"

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

// Less returns the largest key in the tree
// that is less than k.
// If there is no key in the tree less than k,
// p is the zero T and ok is false.
func (t *Tree[T]) Less(k T) (p T, ok bool) {
	// Find the node where k would be inserted to,
	// or the node whose key = k,
	// then find the previous node.

	// https://courses.csail.mit.edu/6.006/fall11/rec/rec05.pdf
	if t.root == nil {
		return
	}

	var c tree.Order
	n, parent := t.root, (*tree.Node[T])(nil)
	for n != nil {
		c = tree.Compare(k, n.Key)
		switch c {
		case tree.Less:
			n, parent = n.Left, n
		case tree.Greater:
			n, parent = n.Right, n
		case tree.Equal:
			n, parent = nil, n
			// setting n to nil breaks the for loop
		default:
			panic("unreachable")
		}
	}

	// parent is where we would be inserted...
	less := parent
	switch c {
	case tree.Less, tree.Equal:
		if less.Left != nil {
			// will be the max in the left subtree
			less = less.Left

			for less.Right != nil {
				less = less.Right
			}
		} else {
			// this may fail
			var child *tree.Node[T]
			for less != nil {
				less, child = less.Parent, less
				if less != nil && less.Right == child {
					return less.Key, true
				}
			}

			return
		}
	case tree.Greater:
		// do nothing
	default:
		panic("unreachable")
	}

	return less.Key, true
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

// String returns a string representation of the tree.
// A complete binary tree with height 2 would look like this:
//	4
//	├─L─2
//	│   ├─L─1
//	│   └─R─3
//	└─R─6
//	    ├─L─5
//	    └─R─7
func (t *Tree[T]) String() string {
	var sb strings.Builder

	if t.root == nil {
		return ""
	}

	printvisit(&sb, t.root, "", "", true, false)

	return sb.String()
}

const (
	treeMidBranch    = "├─"
	treeLastBranch   = "└─"
	treeLeftBranch   = "L─"
	treeRightBranch  = "R─"
	treeMidContinue  = "│   "
	treeLastContinue = "    "
)

func printvisit[T constraints.Ordered](
	sb *strings.Builder, n *tree.Node[T], prefix, branch string, initial, isMid bool) {
	if !initial {
		sb.WriteString(prefix)
		if isMid {
			prefix += treeMidContinue
			sb.WriteString(treeMidBranch)
		} else {
			prefix += treeLastContinue
			sb.WriteString(treeLastBranch)
		}
		sb.WriteString(branch)
	}
	sb.WriteString(fmt.Sprint(n.Key))
	sb.WriteRune('\n')

	if n.Left != nil {
		printvisit(sb, n.Left, prefix, treeLeftBranch, false, n.Right != nil)
	}

	if n.Right != nil {
		printvisit(sb, n.Right, prefix, treeRightBranch, false, false)
	}
}
