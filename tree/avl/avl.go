//go:build XXX

package avl

import (
	"go.lepak.sg/playground/tree"
	"go.lepak.sg/playground/tree/iterator"
	"golang.org/x/exp/constraints"
)

type AVL[T constraints.Ordered] struct {
	root *tree.Node[T, int]

	count  int
	height int
}

func (t *AVL[T]) Contains(k T) bool {
	parent, cmp, _ := t.insertWhere(k, false)

	return parent != nil && cmp == tree.Equal
}

func (t *AVL[T]) insertWhere(k T, unbalance bool) (parent *tree.Node[T, int], cmp tree.Order, depth int) {
	n := t.root
	for n != nil {
		depth++
		cmp = tree.Compare(k, n.Key)
		switch cmp {
		case tree.Less:
			if unbalance {
				n.Extra--
			}
			n, parent = n.Left, n
		case tree.Greater:
			if unbalance {
				n.Extra++
			}
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

func (t *AVL[T]) Insert(k T) bool {
	if t.root == nil {
		t.root = tree.NodeOf(k, 0)
		t.count = 1
		t.height = 1
		return true
	}

	parent, cmp, depth := t.insertWhere(k, true)
	if cmp == tree.Equal {
		return false
	}

	newnode := tree.NodeOf(k, 0)
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

	// TODO Rebalancing

	return true
}

func (t *AVL[T]) InOrderIterator() *iterator.InOrder[T, int] {
	return iterator.NewInOrder(t.root)
}
