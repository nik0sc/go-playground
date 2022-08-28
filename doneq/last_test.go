package doneq

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestLast(t *testing.T) {
	acks := make([]int, 0, 10)

	dq := NewLast(10, func(i int) {
		acks = append(acks, i)
	}, 3, time.Second)

	testdone(t)(dq.Start(context.Background(), 1))
	testdone(t)(dq.Start(context.Background(), 2))
	time.Sleep(100 * time.Millisecond)
	testdone(t)(dq.Start(context.Background(), 3))

	dq.ShutdownWait()
	assert.EqualValues(t, []int{3}, acks)
	goleak.VerifyNone(t)
}

func TestLast_DoneThenBatch(t *testing.T) {
	// Old implementation of Last applied the batch before
	// the wait queue, but the expected behaviour is the
	// other way around
	acks := make([]int, 0, 10)

	dq := NewLast(10, func(i int) {
		acks = append(acks, i)
	}, 2, 100*time.Millisecond)

	barrier := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(3)

	one, _ := dq.Start(context.Background(), 1)
	go func() {
		<-barrier
		one.Done()
		wg.Done()
	}()

	time.Sleep(200 * time.Millisecond)
	two, _ := dq.Start(context.Background(), 2)
	go func() {
		<-barrier
		two.Done()
		wg.Done()
	}()

	three, _ := dq.Start(context.Background(), 3)
	go func() {
		<-barrier
		three.Done()
		wg.Done()
	}()

	time.Sleep(300 * time.Millisecond)
	close(barrier)

	dq.ShutdownWait()
	wg.Wait()
	// old: {1 3}
	assert.EqualValues(t, []int{2, 3}, acks)
	goleak.VerifyNone(t)
}

func testdone(t *testing.T) func(task *Task[int], err error) {
	return func(task *Task[int], err error) {
		assert.NoError(t, err)
		task.Done()
	}
}
