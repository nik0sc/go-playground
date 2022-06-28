package binary

import (
	"errors"
	"fmt"
	"math/rand"

	"go.lepak.sg/playground/tree"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

// BuildRandom builds a binary tree with num nodes.
// Node keys are in the range [0, num) and are inserted in a random order.
// The seed for the random insert order is a parameter,
// which ensures repeatable results.
func BuildRandom(num int, seed int64) *Tree[int] {
	rd := rand.New(rand.NewSource(seed))

	nodes := make([]int, num)
	for i := 0; i < num; i++ {
		nodes[i] = i
	}

	tr := &Tree[int]{}

	rd.Shuffle(num, func(i, j int) {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	})

	for _, n := range nodes {
		tr.Insert(n)
	}

	return tr
}

// BuildRandomBalanced builds a balanced binary tree with num nodes.
// Node keys are in the range [0, num) and are inserted in a random order.
// The seed for the random insert order is a parameter,
// which ensures repeatable results.
// Along the created binary tree, the number of attempts required
// to create the tree is also returned.
func BuildRandomBalanced(num int, seed int64) (*Tree[int], int) {
	// TODO: Accept context or attempt limit, and return error as well
	rd := rand.New(rand.NewSource(seed))

	nodes := make([]int, num)
	for i := 0; i < num; i++ {
		nodes[i] = i
	}

	var tr *Tree[int]
	attempts := 0

	for tr == nil || !tr.Balanced() {
		attempts++

		rd.Shuffle(num, func(i, j int) {
			nodes[i], nodes[j] = nodes[j], nodes[i]
		})

		tr = &Tree[int]{}
		for _, n := range nodes {
			tr.Insert(n)
		}
	}

	return tr, attempts
}

// BuildFromPreAndInOrderIter iteratively builds a binary tree
// from its pre- and in-order traversal.
func BuildFromPreAndInOrderIter[S ~[]T, T constraints.Ordered](
	pre, in S) (*Tree[T], error) {
	// Iterative method. Time O(N^2) Space O(N) (1x nodes, 1x the inOrderMap)
	if len(in) == 0 {
		return nil, errors.New("nothing to build")
	}

	if len(in) != len(pre) {
		return nil, errors.New("pre- and in-order traversals have different lengths")
	}

	inOrderMap := make(map[T]int)
	for i, v := range in {
		if _, ok := inOrderMap[v]; ok {
			return nil, errors.New("duplicated key in in-order traversal")
		}
		inOrderMap[v] = i
	}

	tr := &Tree[T]{root: tree.BasicNodeOf(pre[0])}

	for _, toInsert := range pre[1:] {
		// The idea: walk down the tree to find where toInsert should go
		current, parent := tr.root, (*tree.Node[T, struct{}])(nil)

		var result tree.Order
		for current != nil {
			toInsertIdx, ok := inOrderMap[toInsert]
			if !ok {
				return nil, errors.New("pre-order key not found in in-order traversal")
			}
			currentKeyIdx, ok := inOrderMap[current.Key]
			if !ok {
				// This is actually impossible as
				// previous keys in the pre-order traversal
				// would definitely exist in the tree at this point
				panic("current node key not found in in-order traversal")
			}
			// not actually tree-related, this Compare function is just handy
			result = tree.Compare(toInsertIdx, currentKeyIdx)
			switch result {
			case tree.Less:
				// toInsert is first - go left
				current, parent = current.Left, current
			case tree.Greater:
				// current node key is first - go right
				current, parent = current.Right, current
			default:
				// since we've already checked that the in-order traversal
				// doesn't contain any duplicate keys while building inOrderMap,
				// this can only be caused by:
				return nil, errors.New("duplicated key in pre-order traversal")
			}
		}

		newnode := tree.BasicNodeOf(toInsert)
		// vscode (using the official go lsp) doesn't complain
		// about this line, but goland will:
		// "Cannot use 'parent' (type *tree.Node[T, struct{}])
		// as the type *Node[T, X]"
		newnode.Parent = parent

		switch result {
		case tree.Less:
			parent.Left = newnode
		case tree.Greater:
			parent.Right = newnode
		default:
			panic("unreachable")
		}
	}

	return tr, nil
}

// BuildFromPreAndInOrderRec recursively builds a binary tree
// from its pre- and in-order traversal.
func BuildFromPreAndInOrderRec[S ~[]T, T constraints.Ordered](
	pre, in S) (tr *Tree[T], err error) {
	// A dog on the internet told me how to do this
	// Recursive method. Time O(N^2) Space O(N) (stack frames)
	if len(in) == 0 {
		return nil, errors.New("nothing to build")
	}

	if len(in) != len(pre) {
		return nil, errors.New("pre- and in-order traversals have different lengths")
	}

	//defer func() {
	//	r := recover()
	//	if r != nil {
	//		err = r.(error)
	//	}
	//}()

	tr = &Tree[T]{root: buildFromPreAndInOrderRecVisit(pre, in)}

	return tr, nil
}

func buildFromPreAndInOrderRecVisit[S ~[]T, T constraints.Ordered](
	pre, in S) *tree.Node[T, struct{}] {
	// N = len(pre) = len(in)
	// At least N calls to this function

	if len(pre) != len(in) {
		panic(fmt.Sprintf("invariant broken: "+
			"(len(pre) == %d) != (len(in) == %d)", len(pre), len(in)))
	}

	if len(pre) == 0 {
		return nil
	}

	if len(pre) == 1 {
		if in[0] != pre[0] {
			panic("key in pre-order traversal not found in in-order traversal (base case)")
		}

		return tree.BasicNodeOf(pre[0])
	}

	x := pre[0]
	// O(N) but this gets smaller
	xi := slices.Index(in, x)
	if xi < 0 {
		panic("key in pre-order traversal not found in in-order traversal")
	}

	inleft, inright := in[0:xi], in[xi+1:]

	preleft, preright := pre[1:xi+1], pre[xi+1:]

	n := tree.BasicNodeOf(x)
	n.Left = buildFromPreAndInOrderRecVisit(preleft, inleft)
	if n.Left != nil {
		n.Left.Parent = n.Left
	}
	n.Right = buildFromPreAndInOrderRecVisit(preright, inright)
	if n.Right != nil {
		n.Right.Parent = n.Right
	}

	return n
}
