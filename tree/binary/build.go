package binary

import (
	"go.lepak.sg/playground/tree"
	"golang.org/x/exp/constraints"
)

func BuildFromPreAndInOrder[S ~[]T, T constraints.Ordered](
	pre, in S) *Tree[T] {
	if len(in) == 0 {
		panic("nothing to build")
	}

	if len(in) != len(pre) {
		panic("pre- and in-order traversals have different lengths")
	}

	tr := &Tree[T]{root: tree.BasicNodeOf(pre[0])}

	for _, toInsert := range pre[1:] {
		n, parent := tr.root, (*tree.Node[T, struct{}])(nil)

		var result whichFirstResult
		for n != nil {
			result = whichFirst(in, toInsert, n.Key)
			switch result {
			case whichFirstA:
				// toInsert is first - go left
				n, parent = n.Left, n
			case whichFirstB:
				n, parent = n.Right, n
			default:
				panic("in-order traversal is invalid")
			}
		}

		newnode := tree.BasicNodeOf(toInsert)
		newnode.Parent = parent

		switch result {
		case whichFirstA:
			parent.Left = newnode
		case whichFirstB:
			parent.Right = newnode
		default:
			panic("unreachable")
		}
	}

	return tr
}

type whichFirstResult int

const (
	whichFirstNone whichFirstResult = iota
	whichFirstA
	whichFirstB
)

func whichFirst[S ~[]T, T comparable](sl S, a, b T) whichFirstResult {
	for _, v := range sl {
		if v == a {
			return whichFirstA
		} else if v == b {
			return whichFirstB
		}
	}
	return whichFirstNone
}
