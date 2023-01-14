package testutils

import (
	"github.com/stretchr/testify/assert"
)

type TestT interface {
	Log(...any)
	Logf(string, ...any)
	Error(...any)
	Errorf(string, ...any) // also used by testify/assert
	Helper()
}

// Drain expects to receive data in order from ch, then expects
// ch to be closed.
// The channel must already be filled with the expected data.
// This will not work if the producer is still sending
// when this is called.
func Drain[T any](t TestT, data []T, ch <-chan T) {
	t.Helper()
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
