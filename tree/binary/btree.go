package binary

import (
	"fmt"
	"math"
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

	count  int
	height int
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
	parent, cmp, _ := t.insertWhere(k)

	return parent != nil && cmp == tree.Equal
}

// insertWhere takes a key and returns where to insert it.
// If cmp is tree.Less, k should be inserted on parent's left.
// If cmp is tree.Greater, k should be inserted on parent's right.
// In both these cases, the left/right pointer of parent will be nil.
// If cmp is tree.Equal, parent's key is k.
// If t.root is nil, parent is nil.
// depth is the node distance between the root and the location of k,
// including the root.
func (t *Tree[T]) insertWhere(k T) (parent *tree.Node[T], cmp tree.Order, depth int) {
	n := t.root
	for n != nil {
		depth++
		cmp = tree.Compare(k, n.Key)
		switch cmp {
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

	return
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

	// currently, less is where we would be inserted
	// now go back up the tree to find the previous node,
	// updating less along the way
	less, cmp, _ := t.insertWhere(k)

	switch cmp {
	case tree.Less, tree.Equal:
		if less.Left != nil {
			// will be the max in the left subtree
			less = less.Left

			for less.Right != nil {
				less = less.Right
			}
		} else {
			// this may fail
			// go up the parent and the first right link
			// we pass over, leads to the previous node
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
		// if we would be inserted to the right of some node,
		// that node is the previous node
	default:
		panic("unreachable")
	}

	return less.Key, true
}

// Greater returns the smallest key in the tree
// that is greater than k.
// If there is no key in the tree greater than k,
// p is the zero T and ok is false.
func (t *Tree[T]) Greater(k T) (p T, ok bool) {
	// Find the node where k would be inserted to,
	// or the node whose key = k,
	// then find the next node.

	// https://courses.csail.mit.edu/6.006/fall11/rec/rec05.pdf
	if t.root == nil {
		return
	}

	// currently, greater is where we would be inserted
	// now go back up the tree to find the next node,
	// updating greater along the way
	greater, cmp, _ := t.insertWhere(k)

	switch cmp {
	case tree.Greater, tree.Equal:
		if greater.Right != nil {
			// will be the min in the right subtree
			greater = greater.Right

			for greater.Left != nil {
				greater = greater.Left
			}
		} else {
			// this may fail
			// go up the parent and the first left link
			// we pass over, leads to the next node
			var child *tree.Node[T]
			for greater != nil {
				greater, child = greater.Parent, greater
				if greater != nil && greater.Left == child {
					return greater.Key, true
				}
			}

			return
		}
	case tree.Less:
		// do nothing
		// if we would be inserted to the left of some node,
		// that node is the next node
	default:
		panic("unreachable")
	}

	return greater.Key, true
}

// Insert inserts k into the binary tree.
// If k is already in the tree, Insert returns false.
func (t *Tree[T]) Insert(k T) bool {
	if t.root == nil {
		t.root = tree.NodeOf(k)
		t.count = 1
		t.height = 1
		return true
	}

	parent, cmp, depth := t.insertWhere(k)
	if cmp == tree.Equal {
		return false
	}

	newnode := tree.NodeOf(k)
	newnode.Parent = parent

	switch cmp {
	case tree.Less:
		if parent.Left != nil {
			panic("impossible")
		}
		parent.Left = newnode
	case tree.Greater:
		if parent.Right != nil {
			panic("impossible")
		}
		parent.Right = newnode
	default:
		panic("unreachable")
	}

	t.count++
	if depth > t.height {
		t.height = depth
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

func (t *Tree[T]) PreOrder(f func(k T) bool) {
	t.visitPreOrder(t.root, f)
}

func (t *Tree[T]) visitPreOrder(n *tree.Node[T], f func(k T) bool) bool {
	if !f(n.Key) {
		return false
	}

	if n.Left != nil {
		if !t.visitPreOrder(n.Left, f) {
			return false
		}
	}

	if n.Right != nil {
		if !t.visitPreOrder(n.Right, f) {
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

// printvisit builds a string representation of the tree.
// This is done by a recursive pre-order traversal.
// At each call to printvisit, the line for node n is written.
// The format of the line is like this:
//
//	      branch
//	      ↓↓
//	│   ├─L─1
//	↑↑↑↑↑↑  ↑
//  prefix  fmt.Sprint(n.Key)
//
// prefix is built up by previous visits of the nodes between the root
// and the current node n. Each visit writes either treeMidContinue or
// treeLastContinue depending on whether that node is the last one
// at that visit's level; this information is passed from parent to child
// by isMid.
// branch is decided by whether the current node n is the left or right
// child of its parent.
// initial is true for the root node but false for all others. This
// suppresses growing of the prefix at the root, as the root node has
// no parent.
func printvisit[T constraints.Ordered](
	sb *strings.Builder, n *tree.Node[T], prefix, branch string, initial, isMid bool) {
	if !initial {
		sb.WriteString(prefix)
		if isMid {
			// child nodes of prefix will require
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

	// To generalise this to n-ary trees:
	// - One call for each child
	// - Only the last child printvisit gets isMid=false
	// - branch would need to be expanded beyond treeLeft/RightBranch
	//   (or, just get rid of it)
}

// Height returns the actual height of the tree as well as
// its ideal height if it were perfectly balanced.
func (t *Tree[T]) Height() (actual, ideal int) {
	actual = t.height
	ideal = int(math.Floor(math.Log2(float64(t.count))))
	return
}

// Count returns the number of nodes in the tree.
func (t *Tree[T]) Count() int {
	return t.count
}

// Balanced returns whether the tree is balanced.
func (t *Tree[T]) Balanced() bool {
	actual, ideal := t.Height()
	return actual == ideal
}
