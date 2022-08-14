package doneq

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.lepak.sg/playground/testutils"
	"go.uber.org/goleak"
)

func TestDone(t *testing.T) {
	acks := make([]int, 0, 10)
	goroutineExitOrder := make(chan int, 2)
	var wg sync.WaitGroup

	d := New(10, func(i int) {
		acks = append(acks, i)
	})

	one := d.Start(1)
	wg.Add(1)
	go func() {
		time.Sleep(time.Second)
		one.Done()
		goroutineExitOrder <- 1
		wg.Done()
	}()

	two := d.Start(2)
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

// TODO: test Start blocks when Done isn't called
