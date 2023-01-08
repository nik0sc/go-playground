package slices

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlatten(t *testing.T) {
	assert.Equal(t,
		[]int{},
		Flatten([][]int{}, nil),
	)

	assert.Equal(t,
		[]int{},
		Flatten[int, []int, [][]int](nil, nil),
	)

	assert.Equal(t,
		[]int{1, 2, 3, 4, 5, 6},
		Flatten([][]int{{1, 2, 3}, {4, 5}, {}, {6}}, nil),
	)

	sl := make([]int, 0, 6)
	assert.Equal(t,
		[]int{1, 2, 3, 4, 5, 6},
		Flatten([][]int{{1, 2, 3}, {4, 5}, {}, {6}}, sl),
	)
	sl = sl[:6]
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, sl)
}

func TestFlattenDeep(t *testing.T) {
	assert.Equal(t,
		[]int(nil), // because append() is never called
		FlattenDeep[int]([][]int{}),
	)

	assert.Equal(t,
		[]int(nil), // because append() is never called
		FlattenDeep[int](nil),
	)

	assert.Equal(t,
		[]int{1, 2, 3, 4, 5, 6},
		FlattenDeep[int]([][]int{{1, 2, 3}, {4, 5}, {}, {6}}),
	)

	var obj any
	data := []byte("[[1], [[2,3], [4,[5,[6]]]], [7],[],[[[[8]]]]]")
	require.NoError(t, json.Unmarshal(data, &obj))

	assert.Equal(t,
		[]float64{1, 2, 3, 4, 5, 6, 7, 8},
		FlattenDeep[float64](obj))
}
