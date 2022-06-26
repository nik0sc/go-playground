package iterator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lepak.sg/playground/tree"
)

func newCompleteTree_2Tall() *tree.Node[int, struct{}] {
	t := &tree.Node[int, struct{}]{
		Left: &tree.Node[int, struct{}]{
			Left: &tree.Node[int, struct{}]{
				Key: 1,
			},
			Key: 2,
			Right: &tree.Node[int, struct{}]{
				Key: 3,
			},
		},
		Key: 4,
		Right: &tree.Node[int, struct{}]{
			Left: &tree.Node[int, struct{}]{
				Key: 5,
			},
			Key: 6,
			Right: &tree.Node[int, struct{}]{
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

func TestInOrder(t *testing.T) {
	tests := []struct {
		name   string
		create func() *tree.Node[int, struct{}]
		post   func(t *testing.T, i *InOrder[int, struct{}])
	}{
		{
			name: "empty",
			create: func() *tree.Node[int, struct{}] {
				return nil
			},
			post: func(t *testing.T, i *InOrder[int, struct{}]) {
				assert.False(t, i.Next(), "first")
			},
		},
		{
			name: "one",
			create: func() *tree.Node[int, struct{}] {
				return &tree.Node[int, struct{}]{
					Key: 1,
				}
			},
			post: func(t *testing.T, i *InOrder[int, struct{}]) {
				assert.True(t, i.Next(), "first")
				assert.Equal(t, 1, i.Item())
				assert.False(t, i.Next(), "second")
			},
		},
		{
			name:   "height=2",
			create: newCompleteTree_2Tall,
			post: func(t *testing.T, i *InOrder[int, struct{}]) {
				assert.True(t, i.Next(), "first")
				assert.Equal(t, 1, i.Item())
				assert.True(t, i.Next(), "second")
				assert.Equal(t, 2, i.Item())
				assert.True(t, i.Next(), "third")
				assert.Equal(t, 3, i.Item())
				assert.True(t, i.Next(), "fourth")
				assert.Equal(t, 4, i.Item())
				assert.True(t, i.Next(), "fifth")
				assert.Equal(t, 5, i.Item())
				assert.True(t, i.Next(), "sixth")
				assert.Equal(t, 6, i.Item())
				assert.True(t, i.Next(), "seventh")
				assert.Equal(t, 7, i.Item())
				assert.False(t, i.Next(), "eighth")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.post(t, NewInOrder(tt.create()))
		})
	}
}
