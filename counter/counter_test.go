package counter

import (
	"reflect"
	"testing"
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
