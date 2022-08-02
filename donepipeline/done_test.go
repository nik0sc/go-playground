package donepipeline

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestDone(t *testing.T) {
	acks := make([]int, 0, 10)

	d := New(10, func(i int) {
		acks = append(acks, i)
	})

	one := d.Start(1)
	go func() {
		time.Sleep(time.Second)
		one.Done()
	}()

	two := d.Start(2)
	go func() {
		time.Sleep(100 * time.Millisecond)
		two.Done()
	}()

	d.ShutdownWait()
	assert.EqualValues(t, []int{1, 2}, acks)
	goleak.VerifyNone(t)
}

// TODO: test Start blocks when Done isn't called
