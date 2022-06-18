package binary

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/constraints"
)

type insertspec[T constraints.Ordered] struct {
	key     T
	success bool
}

func TestInsert(t *testing.T) {
	tests := []struct {
		name    string
		inserts []insertspec[int]
		post    func(t *testing.T, tr *Tree[int])
	}{
		{
			name: "empty",
			post: func(t *testing.T, tr *Tree[int]) {
				assert.Nil(t, tr.root)
			},
		},
		{
			name: "one",
			inserts: []insertspec[int]{
				{
					key:     1,
					success: true,
				},
			},
			post: func(t *testing.T, tr *Tree[int]) {
				assert.NotNil(t, tr.root)
				assert.Equal(t, 1, tr.root.Key)
				assert.Nil(t, tr.root.Left)
				assert.Nil(t, tr.root.Right)
				assert.Nil(t, tr.root.Parent)
			},
		},
		{
			name: "one duplicate",
			inserts: []insertspec[int]{
				{
					key:     1,
					success: true,
				},
				{
					key:     1,
					success: false,
				},
			},
			post: func(t *testing.T, tr *Tree[int]) {
				assert.NotNil(t, tr.root)
				assert.Equal(t, 1, tr.root.Key)
				assert.Nil(t, tr.root.Left)
				assert.Nil(t, tr.root.Right)
				assert.Nil(t, tr.root.Parent)
			},
		},
		{
			name: "left",
			inserts: []insertspec[int]{
				{
					key:     2,
					success: true,
				},
				{
					key:     1,
					success: true,
				},
			},
			post: func(t *testing.T, tr *Tree[int]) {
				assert.NotNil(t, tr.root)
				assert.Equal(t, 2, tr.root.Key)
				assert.NotNil(t, tr.root.Left)
				assert.Nil(t, tr.root.Right)
				assert.Nil(t, tr.root.Parent)
				assert.Equal(t, 1, tr.root.Left.Key)
				assert.Nil(t, tr.root.Left.Left)
				assert.Nil(t, tr.root.Left.Right)
				assert.Equal(t, tr.root, tr.root.Left.Parent)
			},
		},
		{
			name: "right",
			inserts: []insertspec[int]{
				{
					key:     1,
					success: true,
				},
				{
					key:     2,
					success: true,
				},
			},
			post: func(t *testing.T, tr *Tree[int]) {
				assert.NotNil(t, tr.root)
				assert.Equal(t, 1, tr.root.Key)
				assert.Nil(t, tr.root.Left)
				assert.NotNil(t, tr.root.Right)
				assert.Nil(t, tr.root.Parent)
				assert.Equal(t, 2, tr.root.Right.Key)
				assert.Nil(t, tr.root.Right.Left)
				assert.Nil(t, tr.root.Right.Right)
				assert.Equal(t, tr.root, tr.root.Right.Parent)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := Tree[int]{}

			for _, k := range tt.inserts {
				assert.Equal(t, k.success, tr.Insert(k.key))
			}

			tt.post(t, &tr)
		})
	}
}
