package iterator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lepak.sg/playground/tree"
)

func TestInOrderStack(t *testing.T) {
	tests := []struct {
		name       string
		create     func() *tree.Node[int, struct{}]
		heightHint int
		post       func(t *testing.T, i *InOrderStack[int, struct{}])
	}{
		{
			name: "empty",
			create: func() *tree.Node[int, struct{}] {
				return nil
			},
			post: func(t *testing.T, i *InOrderStack[int, struct{}]) {
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
			post: func(t *testing.T, i *InOrderStack[int, struct{}]) {
				assert.True(t, i.Next(), "first")
				assert.Equal(t, 1, i.Item())
				assert.False(t, i.Next(), "second")
			},
		},
		{
			name:   "height=2",
			create: newCompleteTree_2Tall,
			post: func(t *testing.T, i *InOrderStack[int, struct{}]) {
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
		{
			name: "dogleg no parent",
			create: func() *tree.Node[int, struct{}] {
				return &tree.Node[int, struct{}]{
					Left: &tree.Node[int, struct{}]{
						Left: &tree.Node[int, struct{}]{
							Key: 1,
						},
						Key: 5,
						Right: &tree.Node[int, struct{}]{
							Left: &tree.Node[int, struct{}]{
								Key: 6,
							},
							Key: 7,
						},
					},
					Key: 8,
					Right: &tree.Node[int, struct{}]{
						Key: 9,
					},
				}
			},
			heightHint: 3,
			post: func(t *testing.T, i *InOrderStack[int, struct{}]) {
				assert.True(t, i.Next(), "first")
				assert.Equal(t, 1, i.Item())
				assert.True(t, i.Next(), "second")
				assert.Equal(t, 5, i.Item())
				assert.True(t, i.Next(), "third")
				assert.Equal(t, 6, i.Item())
				assert.True(t, i.Next(), "fourth")
				assert.Equal(t, 7, i.Item())
				assert.True(t, i.Next(), "fifth")
				assert.Equal(t, 8, i.Item())
				assert.True(t, i.Next(), "sixth")
				assert.Equal(t, 9, i.Item())
				assert.False(t, i.Next(), "seventh")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.post(t, NewInOrderStack(tt.create(), tt.heightHint))
		})
	}
}
