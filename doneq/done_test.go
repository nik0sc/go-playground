package doneq

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.lepak.sg/playground/testutils"
	"go.uber.org/goleak"
)

func TestDone(t *testing.T) {
	acks := make([]int, 0, 10)
	goroutineExitOrder := make(chan int, 2)

	d := New(10, func(i int) {
		acks = append(acks, i)
	})

	one := d.Start(1)
	go func() {
		time.Sleep(time.Second)
		one.Done()
		goroutineExitOrder <- 1
	}()

	two := d.Start(2)
	go func() {
		time.Sleep(100 * time.Millisecond)
		two.Done()
		goroutineExitOrder <- 2
	}()

	d.ShutdownWait()
	assert.EqualValues(t, []int{1, 2}, acks)
	close(goroutineExitOrder)
	testutils.Drain(t, []int{2, 1}, goroutineExitOrder)
	goleak.VerifyNone(t)
}

// TODO: test Start blocks when Done isn't called
