package tree

import (
	"golang.org/x/exp/constraints"
)

type Node[T comparable] struct {
	Key                 T
	Left, Right, Parent *Node[T]
}

func NodeOf[T comparable](k T) *Node[T] {
	return &Node[T]{
		Key: k,
	}
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
