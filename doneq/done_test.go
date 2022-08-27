package doneq

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.lepak.sg/playground/testutils"
	"go.uber.org/goleak"
)

func TestDone_EnsureOrder(t *testing.T) {
	acks := make([]int, 0, 10)
	goroutineExitOrder := make(chan int, 2)
	var wg sync.WaitGroup

	d := New(2, func(i int) {
		acks = append(acks, i)
	})

	// task 1 is in watch
	one, err := d.Start(context.Background(), 1)
	assert.NoError(t, err, "task 1 start returned error")
	wg.Add(1)
	go func() {
		time.Sleep(time.Second)
		one.Done()
		goroutineExitOrder <- 1
		wg.Done()
	}()

	// task 2 is in the channel
	two, err := d.Start(context.Background(), 2)
	assert.NoError(t, err, "task 2 start returned error")
	wg.Add(1)
	go func() {
		time.Sleep(100 * time.Millisecond)
		two.Done()
		goroutineExitOrder <- 2
		wg.Done()
	}()

	d.ShutdownWait()
	assert.EqualValues(t, []int{1, 2}, acks)

	wg.Wait()
	close(goroutineExitOrder)
	testutils.Drain(t, []int{2, 1}, goroutineExitOrder)

	goleak.VerifyNone(t)
}

func TestDone_Context(t *testing.T) {
	// This tests that StartContext blocks, but not forever

	d := New(1, func(i int) {
		assert.EqualValues(t, 1, i, "unexpected progress")
	})

	one, err := d.Start(context.Background(), 1)
	assert.NoError(t, err, "task 1 start returned error")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	two, err := d.Start(ctx, 2)
	assert.ErrorIs(t, err, context.DeadlineExceeded,
		"error was not DeadlineExceeded")
	assert.Nil(t, two, "task was not nil")

	one.Done()
	d.ShutdownWait()
	goleak.VerifyNone(t)
}
