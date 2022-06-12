package parallel

import (
	"context"
	"reflect"
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestMapBoundedPoolLockfree(t *testing.T) {
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
					time.Sleep(time.Millisecond)
					return v * 2
				},
				workers: 2,
			},
			wantResult: []int{2, 4, 6, 8, 10, 12, 14, 16, 18, 20},
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := MapBoundedPoolLockfree(tt.args.ctx, tt.args.list, tt.args.f, tt.args.workers)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapBoundedPoolLockfree() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("MapBoundedPoolLockfree() = %v, want %v", gotResult, tt.wantResult)
			}

			goleak.VerifyNone(t)
		})
	}
}
