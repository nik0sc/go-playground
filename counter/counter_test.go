package counter

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		list string
		want map[byte]int
	}{
		{
			"empty",
			"",
			map[byte]int{},
		},
		{
			"one",
			"a",
			map[byte]int{
				'a': 1,
			},
		},
		{
			"multi",
			"abracadabra",
			map[byte]int{
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
			if got := New([]byte(tt.list)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTopK(t *testing.T) {
	ctr := New([]byte("abracadabra"))
	res := TopK(ctr, 1)
	assert.Len(t, res, 1)
	assert.Equal(t, 5, res[0].Count)
	assert.Equal(t, byte('a'), res[0].Value)
}
