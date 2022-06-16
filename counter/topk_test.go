package counter

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTopK(t *testing.T) {
	type args struct {
		ctr map[byte]int
		k   int
	}
	tests := []struct {
		name string
		args args
		want []Entry[byte]
	}{
		{
			name: "empty",
			want: []Entry[byte]{},
		},
		{
			name: "one",
			args: args{
				ctr: map[byte]int{'a': 1},
				k:   1,
			},
			want: []Entry[byte]{
				{
					Element: 'a',
					Count:   1,
				},
			},
		},
		{
			name: "two",
			args: args{
				ctr: Counter([]byte("aardvark")),
				k:   2,
			},
			want: []Entry[byte]{
				{
					Element: 'a',
					Count:   3,
				},
				{
					Element: 'r',
					Count:   2,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TopK(tt.args.ctr, tt.args.k); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TopK() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTopK_Panic(t *testing.T) {
	type args struct {
		ctr map[byte]int
		k   int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty",
			args: args{
				ctr: map[byte]int{'a': 1},
				k:   2,
			},
			want: "k is larger than number of elements in ctr",
		},
		{
			name: "negative k",
			args: args{
				k: -1,
			},
			want: "k is negative",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PanicsWithValue(t, tt.want, func() {
				_ = TopK(tt.args.ctr, tt.args.k)
			})
		})
	}
}

func TestBottomK(t *testing.T) {
	type args struct {
		ctr map[byte]int
		k   int
	}
	tests := []struct {
		name string
		args args
		want []Entry[byte]
	}{
		{
			name: "empty",
			want: []Entry[byte]{},
		},
		{
			name: "one",
			args: args{
				ctr: map[byte]int{'a': 1},
				k:   1,
			},
			want: []Entry[byte]{
				{
					Element: 'a',
					Count:   1,
				},
			},
		},
		{
			name: "two",
			args: args{
				ctr: Counter([]byte("rraacecaarr")),
				k:   2,
			},
			want: []Entry[byte]{
				{
					Element: 'e',
					Count:   1,
				},
				{
					Element: 'c',
					Count:   2,
				},
			},
		},
		{
			name: "negative",
			args: args{
				ctr: map[byte]int{
					'a': 1,
					'b': -1,
					'c': 0,
				},
				k: 2,
			},
			want: []Entry[byte]{
				{
					Element: 'b',
					Count:   -1,
				},
				{
					Element: 'c',
					Count:   0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BottomK(tt.args.ctr, tt.args.k); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BottomK() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTopK2(t *testing.T) {
	type args struct {
		ctr map[byte]int
		k   int
	}
	tests := []struct {
		name string
		args args
		want []Entry[byte]
	}{
		{
			name: "empty",
			want: []Entry[byte]{},
		},
		{
			name: "one",
			args: args{
				ctr: map[byte]int{'a': 1},
				k:   1,
			},
			want: []Entry[byte]{
				{
					Element: 'a',
					Count:   1,
				},
			},
		},
		{
			name: "two",
			args: args{
				ctr: Counter([]byte("aardvark")),
				k:   2,
			},
			want: []Entry[byte]{
				{
					Element: 'a',
					Count:   3,
				},
				{
					Element: 'r',
					Count:   2,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TopKAlt(tt.args.ctr, tt.args.k); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TopK() = %v, want %v", got, tt.want)
			}
		})
	}
}
