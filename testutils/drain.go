package testutils

import (
	"time"

	"github.com/stretchr/testify/assert"
)

type TestT interface {
	Log(...any)
	Logf(string, ...any)
	Error(...any)
	Errorf(string, ...any) // also used by testify/assert
}

// Drain expects to receive data in order from ch, then expects
// ch to be closed.
// The channel must already be filled with the expected data.
// This will not work if the producer is still sending
// when this is called.
func Drain[T any](t TestT, data []T, ch <-chan T) {
	t.Logf("draining: expecting %v", data)
	for i, datum := range data {
		select {
		case el, ok := <-ch:
			assert.Truef(t, ok, "channel closed early, expecting %v", datum)
			assert.Equal(t, datum, el)
		default:
			t.Errorf("channel was empty, expecting i=%d %v", i, datum)
			// future iterations will fail, return now
			return
		}
	}

	select {
	case b, ok := <-ch:
		assert.Falsef(t, ok, "channel should be closed, but received: %v", b)
	default:
		t.Error("at the end of draining, channel was empty but unclosed")
	}
}

// DrainBlocking expects to receive data in order from ch, then expects
// ch to be closed.
// DrainBlocking may run concurrently with the producer. Each channel receive
// (including the final one to determine closedness) will fail if it takes
// longer than timeout duration to complete.
func DrainBlocking[T any](t TestT, data []T, ch <-chan T, timeout time.Duration) {
	t.Logf("draining: expecting %v", data)

	var timer *time.Timer

	for i, datum := range data {
		if timer == nil {
			timer = time.NewTimer(timeout)
		} else {
			timer.Reset(timeout)
		}

		select {
		case el, ok := <-ch:
			assert.Truef(t, ok, "channel closed early, expecting %v", datum)
			assert.Equal(t, datum, el)
			if !timer.Stop() {
				<-timer.C
			}
		case <-timer.C:
			t.Errorf("timed out waiting for producer for %s, expecting i=%d %v",
				timeout.String(), i, datum)
			return
		}
	}
	// if this loop ran at least once, then timer is not nil,
	// is drained, and needs resetting

	if timer == nil {
		timer = time.NewTimer(timeout)
	} else {
		timer.Reset(timeout)
	}

	select {
	case b, ok := <-ch:
		assert.Falsef(t, ok, "channel should be closed, but received: %v", b)
	case <-timer.C:
		t.Errorf("timed out waiting for producer to close the channel for %s",
			timeout.String())
	}
}
