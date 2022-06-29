package iterator

import (
	"fmt"

	"go.lepak.sg/playground/tree"
	"golang.org/x/exp/constraints"
)

var _ Iterator[int] = (*InOrderStack[int, any])(nil)

// InOrderStack is an iterator object over a binary tree.
// It is functionally equivalent to InOrder, but this does
// not rely on the node parent pointer, instead keeping
// an internal stack of previous nodes.
type InOrderStack[T constraints.Ordered, X any] struct {
	root  *tree.Node[T, X]
	stack []*tree.Node[T, X]
}

// Recursive in order iteration looks like this:
//	func visit(n *Node, f func(*Node)) {
//		if n == nil {
//			return
//		}
//		visit(n.Left, f)	--(1)
//		f(n)
//		visit(n.Right, f)	--(2)
//	}
// When Next is called, everything up to (1) can be run,
// all the way down to the leftmost child node. This adds
// visit stack frames and we can replicate this in i.stack.
// The associated call to Item is equivalent to f(n).
// The next call to Next continues from (2).
// When we pop off an inOrderFrame from i.stack, we'll know
// we should be in the second half of visit, because we
// already did the first half before pushing on this frame.
// We can resume from (2), popping off the frame and
// pushing on all the right children of that node.

// NewInOrderStack creates a new in-order iterator.
// If the tree's height is known, pass it as heightHint.
// Otherwise it's safe to leave it as 0.
func NewInOrderStack[T constraints.Ordered, X any](
	root *tree.Node[T, X], heightHint int) *InOrderStack[T, X] {
	return &InOrderStack[T, X]{
		root:  root,
		stack: make([]*tree.Node[T, X], 0, heightHint+1),
	}
}

func (i *InOrderStack[T, X]) Next() bool {
	defer func() {
		fmt.Printf("the current stack (%d): ", len(i.stack))
		for _, v := range i.stack {
			fmt.Print(v.Key, "")
		}
		fmt.Println()
	}()

	if i.root == nil {
		return false
	}

	if len(i.stack) == 0 {
		n := i.root
		for n != nil {
			i.stack = append(i.stack, n)
			n = n.Left
		}
		return true
	}

	pop := i.stack[len(i.stack)-1]
	i.stack = i.stack[:len(i.stack)-1]

	if pop.Right != nil {
		n := pop.Right
		for n != nil {
			i.stack = append(i.stack, n)
			n = n.Left
		}
	} else if len(i.stack) == 0 {
		return false
	}

	return true
}

func (i *InOrderStack[T, X]) Item() T {
	return i.stack[len(i.stack)-1].Key
}
