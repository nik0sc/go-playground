package tree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func newCompleteTree_2Tall() *Node[int, struct{}] {
	t := &Node[int, struct{}]{
		Left: &Node[int, struct{}]{
			Left: &Node[int, struct{}]{
				Key: 1,
			},
			Key: 2,
			Right: &Node[int, struct{}]{
				Key: 3,
			},
		},
		Key: 4,
		Right: &Node[int, struct{}]{
			Left: &Node[int, struct{}]{
				Key: 5,
			},
			Key: 6,
			Right: &Node[int, struct{}]{
				Key: 7,
			},
		},
	}

	t.Left.Left.Parent = t.Left
	t.Left.Right.Parent = t.Left
	t.Left.Parent = t

	t.Right.Left.Parent = t.Right
	t.Right.Right.Parent = t.Right
	t.Right.Parent = t

	return t
}

func TestNode_RotateLeft(t *testing.T) {
	tr := newCompleteTree_2Tall()

	should6 := tr.RotateLeft()

	assert.Equal(t, 6, should6.Key)
}

func TestNode_RotateRight(t *testing.T) {
	tr := newCompleteTree_2Tall()

	should2 := tr.RotateRight()

	assert.Equal(t, 2, should2.Key)
}

// TODO: more tests especially for correct pointers
