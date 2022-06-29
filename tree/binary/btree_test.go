package binary

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestString(t *testing.T) {
	tests := []struct {
		name    string
		inserts []insertspec[int]
		want    string
	}{
		{
			name: "empty",
			want: "",
		},
		{
			name: "one",
			inserts: []insertspec[int]{
				{
					key:     1,
					success: true,
				},
			},
			want: "1\n",
		},
		{
			name: "complete height 2",
			inserts: []insertspec[int]{
				{
					key:     4,
					success: true,
				},
				{
					key:     2,
					success: true,
				},
				{
					key:     1,
					success: true,
				},
				{
					key:     3,
					success: true,
				},
				{
					key:     6,
					success: true,
				},
				{
					key:     5,
					success: true,
				},
				{
					key:     7,
					success: true,
				},
			},
			want: `4
├─L─2
│   ├─L─1
│   └─R─3
└─R─6
    ├─L─5
    └─R─7
`,
		},
		{
			name: "zigzag",
			inserts: []insertspec[int]{
				{
					key:     1,
					success: true,
				},
				{
					key:     5,
					success: true,
				},
				{
					key:     2,
					success: true,
				},
				{
					key:     4,
					success: true,
				},
				{
					key:     3,
					success: true,
				},
			},
			want: `1
└─R─5
    └─L─2
        └─R─4
            └─L─3
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := Tree[int]{}

			for _, k := range tt.inserts {
				assert.Equal(t, k.success, tr.Insert(k.key))
			}

			s := tr.String()
			require.Equal(t, tt.want, s)
			t.Log(s)
		})
	}
}

func TestLess(t *testing.T) {
	tests := []struct {
		name    string
		inserts []insertspec[int]
		k       int
		less    int
		ok      bool
	}{
		{
			name: "empty",
			k:    0,
			less: 0,
			ok:   false,
		},
		{
			name: "less than smallest",
			inserts: []insertspec[int]{
				{
					key:     10,
					success: true,
				},
			},
			k:    1,
			less: 0,
			ok:   false,
		},
		{
			name: "greater than smallest",
			inserts: []insertspec[int]{
				{
					key:     10,
					success: true,
				},
			},
			k:    11,
			less: 10,
			ok:   true,
		},
		{
			name: "equal to smallest",
			inserts: []insertspec[int]{
				{
					key:     10,
					success: true,
				},
			},
			k:    10,
			less: 0,
			ok:   false,
		},
		{
			name: "part 1",
			inserts: []insertspec[int]{
				{
					key:     10,
					success: true,
				},
				{
					key:     5,
					success: true,
				},
				{
					key:     1,
					success: true,
				},
				{
					key:     7,
					success: true,
				},
				{
					key:     12,
					success: true,
				},
			},
			k:    6,
			less: 5,
			ok:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := Tree[int]{}

			for _, k := range tt.inserts {
				assert.Equal(t, k.success, tr.Insert(k.key))
			}

			less, ok := tr.Less(tt.k)

			assert.Equal(t, tt.less, less)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestLess_VsInOrder(t *testing.T) {
	tests := []struct {
		name    string
		inserts []insertspec[int]
		start   int
		end     int
	}{
		{
			name: "part 1",
			inserts: []insertspec[int]{
				{
					key:     10,
					success: true,
				},
				{
					key:     5,
					success: true,
				},
				{
					key:     1,
					success: true,
				},
				{
					key:     7,
					success: true,
				},
				{
					key:     12,
					success: true,
				},
			},
			start: 0,
			end:   13,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := Tree[int]{}

			for _, k := range tt.inserts {
				assert.Equal(t, k.success, tr.Insert(k.key))
			}

			var sorted []int
			it := tr.InOrderIterator()
			for it.Next() {
				sorted = append(sorted, it.Item())
			}

			t.Logf("sorted=%v", sorted)

			for i := tt.start; i <= tt.end; i++ {
				less, ok := tr.Less(i)

				var lessSlow int
				var okSlow bool

				idx := sort.SearchInts(sorted, i)
				if idx > 0 {
					lessSlow, okSlow = sorted[idx-1], true
				}

				assert.Equalf(t, lessSlow, less, "i=%d", i)
				assert.Equalf(t, okSlow, ok, "i=%d", i)
			}
		})
	}
}
