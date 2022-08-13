package doneq

import (
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

	d.Start(1).Done()
	d.Start(2).Done()
	time.Sleep(100 * time.Millisecond)
	d.Start(3).Done()

	d.ShutdownWait()
	assert.EqualValues(t, []int{3}, acks)
	goleak.VerifyNone(t)
}
