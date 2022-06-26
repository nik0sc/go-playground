package tree

// RotateLeft rotates a Node to the left and returns
// the Node that now occupies its old position.
// For example, this is the result of calling n.RotateLeft:
//	  -> n            p
//      / \          / \
//	   m   p   ->   n   q
//	      / \      / \
//	     o   q    m   o
// The right child p is returned from n.RotateLeft.
// The ordering invariant m < n < o < p < q is always preserved.
func (n *Node[T, X]) RotateLeft() *Node[T, X] {
	if n == nil {
		panic("cannot RotateLeft on nil")
	}

	if n.Right == nil {
		panic("cannot RotateLeft with nil right")
	}

	if n.Right.Left == nil {
		panic("cannot RotateLeft with nil right-left")
	}

	p, o := n.Right, n.Right.Left
	p.Parent = n.Parent

	n.Right = o
	o.Parent = n.Right

	p.Left = n
	n.Parent = p

	return p
}

// RotateRight rotates a Node to the right and returns
// the Node that now occupies its old position.
// For example, this is the result of calling n.RotateRight:
//	  -> n            l
//      / \          / \
//	   l   o   ->   k   n
//	  / \              / \
//	 k   m            m   o
// The left child l is returned from n.RotateRight.
// The ordering invariant k < l < m < n < o is always preserved.
func (n *Node[T, X]) RotateRight() *Node[T, X] {
	if n == nil {
		panic("cannot RotateRight on nil")
	}

	if n.Left == nil {
		panic("cannot RotateRight with nil left")
	}

	if n.Left.Right == nil {
		panic("cannot RotateRight with nil left-right")
	}

	l, m := n.Left, n.Left.Right
	l.Parent = n.Parent

	n.Left = m
	m.Parent = n.Left

	l.Right = n
	n.Parent = l

	return l
}
