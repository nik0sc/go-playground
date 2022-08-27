package doneq

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestLast(t *testing.T) {
	acks := make([]int, 0, 10)

	d := NewLast(10, func(i int) {
		acks = append(acks, i)
	}, 3, time.Second)

	testdone(t)(d.Start(context.Background(), 1))
	testdone(t)(d.Start(context.Background(), 2))
	time.Sleep(100 * time.Millisecond)
	testdone(t)(d.Start(context.Background(), 3))

	d.ShutdownWait()
	assert.EqualValues(t, []int{3}, acks)
	goleak.VerifyNone(t)
}

func testdone(t *testing.T) func(task *Task[int], err error) {
	return func(task *Task[int], err error) {
		assert.NoError(t, err)
		task.Done()
	}
}
