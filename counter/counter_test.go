package counter

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCounter(t *testing.T) {
	tests := []struct {
		name string
		list string
		want map[byte]int
	}{
		{
			name: "empty",
			want: map[byte]int{},
		},
		{
			name: "one",
			list: "a",
			want: map[byte]int{
				'a': 1,
			},
		},
		{
			name: "multi",
			list: "abracadabra",
			want: map[byte]int{
				'a': 5,
				'b': 2,
				'r': 2,
				'c': 1,
				'd': 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Counter([]byte(tt.list)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Counter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name string
		a    map[byte]int
		b    map[byte]int
		want map[byte]int
	}{
		{
			name: "empty",
			a:    nil,
			b:    nil,
			want: map[byte]int{},
		},
		{
			name: "identity",
			a: map[byte]int{
				'a': 1,
				'b': 2,
			},
			b: nil,
			want: map[byte]int{
				'a': 1,
				'b': 2,
			},
		},
		{
			name: "identity 2",
			a:    nil,
			b: map[byte]int{
				'a': 1,
				'b': 2,
			},
			want: map[byte]int{
				'a': 1,
				'b': 2,
			},
		},
		{
			name: "sum",
			a: map[byte]int{
				'a': 1,
				'b': 2,
			},
			b: map[byte]int{
				'a': 3,
				'b': 4,
			},
			want: map[byte]int{
				'a': 4,
				'b': 6,
			},
		},
		{
			name: "disjoint",
			a: map[byte]int{
				'a': 1,
				'b': 2,
			},
			b: map[byte]int{
				'c': 3,
				'd': 4,
			},
			want: map[byte]int{
				'a': 1,
				'b': 2,
				'c': 3,
				'd': 4,
			},
		},
		{
			name: "negative",
			a: map[byte]int{
				'a': 1,
				'b': -2,
				'c': 3,
			},
			b: map[byte]int{
				'a': -2,
				'b': 1,
				'c': -3,
			},
			want: map[byte]int{
				'a': -1,
				'b': -1,
				'c': 0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acopy, bcopy := mapcopy(tt.a), mapcopy(tt.b)

			assert.Equal(t, tt.want, Add(tt.a, tt.b))
			assert.Equal(t, acopy, tt.a)
			assert.Equal(t, bcopy, tt.b)
		})
	}
}

func TestSubtract(t *testing.T) {
	tests := []struct {
		name string
		a    map[byte]int
		b    map[byte]int
		want map[byte]int
	}{
		{
			name: "empty",
			a:    nil,
			b:    nil,
			want: map[byte]int{},
		},
		{
			name: "identity",
			a: map[byte]int{
				'a': 1,
				'b': 2,
			},
			b: nil,
			want: map[byte]int{
				'a': 1,
				'b': 2,
			},
		},
		{
			name: "invert sign",
			a:    nil,
			b: map[byte]int{
				'a': 1,
				'b': 2,
			},
			want: map[byte]int{
				'a': -1,
				'b': -2,
			},
		},
		{
			name: "difference",
			a: map[byte]int{
				'a': 1,
				'b': 4,
			},
			b: map[byte]int{
				'a': 3,
				'b': 2,
			},
			want: map[byte]int{
				'a': -2,
				'b': 2,
			},
		},
		{
			name: "disjoint",
			a: map[byte]int{
				'a': 1,
				'b': 2,
			},
			b: map[byte]int{
				'c': 3,
				'd': 4,
			},
			want: map[byte]int{
				'a': 1,
				'b': 2,
				'c': -3,
				'd': -4,
			},
		},
		{
			name: "negative",
			a: map[byte]int{
				'a': 1,
				'b': -2,
			},
			b: map[byte]int{
				'a': -2,
				'b': 1,
			},
			want: map[byte]int{
				'a': 3,
				'b': -3,
			},
		},
		{
			name: "self",
			a: map[byte]int{
				'a': 1,
				'b': -2,
			},
			b: map[byte]int{
				'a': 1,
				'b': -2,
			},
			want: map[byte]int{
				'a': 0,
				'b': 0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acopy, bcopy := mapcopy(tt.a), mapcopy(tt.b)

			assert.Equal(t, tt.want, Subtract(tt.a, tt.b))
			assert.Equal(t, acopy, tt.a)
			assert.Equal(t, bcopy, tt.b)
		})
	}
}

func mapcopy[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return nil
	}

	cp := make(map[K]V, len(m))

	for k, v := range m {
		cp[k] = v
	}

	return cp
}

func TestTotal(t *testing.T) {
	tests := []struct {
		name string
		ctr  map[byte]int
		want int
	}{
		{
			name: "empty",
			ctr:  nil,
			want: 0,
		},
		{
			name: "sum",
			ctr: map[byte]int{
				'a': 1,
				'b': 2,
				'c': 3,
			},
			want: 6,
		},
		{
			name: "sum to zero",
			ctr: map[byte]int{
				'a': 1,
				'b': 2,
				'c': -3,
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Total(tt.ctr))
		})
	}
}
