package tree

import (
	"golang.org/x/exp/constraints"
)

// Node is a generic tree node. This should not be
// directly exposed to users of trees, but
// tree implementations may use it.
type Node[T comparable, X any] struct {
	Key                 T
	Left, Right, Parent *Node[T, X]
	// Extra is extra data meant for tree implementations
	// eg balance factor, node colour etc.
	Extra X
}

func NodeOf[T comparable, X any](k T, extra X) *Node[T, X] {
	return &Node[T, X]{
		Key:   k,
		Extra: extra,
	}
}

func BasicNodeOf[T comparable](k T) *Node[T, struct{}] {
	return NodeOf[T, struct{}](k, struct{}{})
}

// Should T be constraints.Ordered?

type Comparable[T any] interface {
	CompareTo(T) int
}

// This allows T to mutate, for example if we defined:
//	type IntPtr *int
// and then implemented:
//	func (ip IntPtr) CompareTo(ip2 IntPtr) int {
//		return (*ip2)-(*ip)
//	}
// client code could mutate *IntPtr at any time, ruining our tree invariants.
// This is probably not ideal.

type NodeComparable[T any] struct {
	Key                 Comparable[T]
	Left, Right, Parent *NodeComparable[T]
}

type Order int

const (
	Less Order = iota - 1
	Equal
	Greater
)

func Compare[T constraints.Ordered](l, r T) Order {
	if l < r {
		return Less
	} else if l > r {
		return Greater
	} else {
		return Equal
	}
}
