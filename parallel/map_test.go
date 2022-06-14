package parallel

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

type impl[T, R any] struct {
	name  string
	fname string
	f     func(context.Context, []T, func(int, T) R, int) ([]R, error)
}

func getimpls[T, R any]() []impl[T, R] {
	return []impl[T, R]{
		{
			name:  "Sema",
			fname: "MapBoundedSema",
			f:     MapBoundedSema[[]T, T, R],
		},
		{
			name:  "Pool",
			fname: "MapBoundedPool",
			f:     MapBoundedPool[[]T, T, R],
		},
		{
			name:  "PoolLockfree",
			fname: "MapBoundedPoolLockfree",
			f:     MapBoundedPoolLockfree[[]T, T, R],
		},
	}
}

func TestMapBounded(t *testing.T) {
	type args struct {
		ctx     context.Context
		list    []int
		f       func(int, int) int
		workers int
	}
	tests := []struct {
		name       string
		args       args
		wantResult []int
		wantErr    bool
	}{
		{
			name: "empty",
			args: args{
				ctx:     context.Background(),
				list:    nil,
				f:       func(_, v int) int { return v },
				workers: 0,
			},
			wantResult: []int{},
			wantErr:    false,
		},
		{
			name: "one",
			args: args{
				ctx:     context.Background(),
				list:    []int{1},
				f:       func(_, v int) int { return v * 2 },
				workers: 1,
			},
			wantResult: []int{2},
			wantErr:    false,
		},
		{
			name: "two",
			args: args{
				ctx:  context.Background(),
				list: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
				f: func(_, v int) int {
					return v * 2
				},
				workers: 2,
			},
			wantResult: []int{2, 4, 6, 8, 10, 12, 14, 16, 18, 20},
			wantErr:    false,
		},
		{
			name: "two but intense",
			args: args{
				ctx:  context.Background(),
				list: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
				f: func(_, v int) int {
					for i := 0; i < 1000000; i++ {
						v *= -1
					}
					return v
				},
				workers: 2,
			},
			wantResult: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			wantErr:    false,
		},
	}

	for _, impl := range getimpls[int, int]() {
		for _, tt := range tests {
			t.Run(impl.name+"/"+tt.name, func(t *testing.T) {
				gotResult, err := impl.f(tt.args.ctx, tt.args.list, tt.args.f, tt.args.workers)
				if (err != nil) != tt.wantErr {
					t.Errorf("%s() error = %v, wantErr %v", impl.fname, err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(gotResult, tt.wantResult) {
					t.Errorf("%s() = %v, want %v", impl.fname, gotResult, tt.wantResult)
				}

				goleak.VerifyNone(t)
			})
		}
	}
}

func TestMapBounded_Cancellation(t *testing.T) {
	for _, impl := range getimpls[int, int]() {
		t.Run(impl.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())

			in := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

			f := func(_, v int) int {
				if v == 1 {
					cancel()
				}
				return v * 2
			}

			workers := 2

			result, err := impl.f(ctx, in, f, workers)

			assert.ErrorIs(t, err, context.Canceled)

			t.Logf("result=%v", result)

			goleak.VerifyNone(t)
		})
	}
}
