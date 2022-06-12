package testutils

import (
	"github.com/stretchr/testify/assert"
	"go.lepak.sg/playground/chops"
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
		chops.TryRecv(ch).Match(
			func(el T) {
				assert.Equal(t, datum, el)
			},
			func() {
				t.Errorf("channel closed early, expecting %v", datum)
			},
			func() {
				t.Errorf("channel was empty, expecting i=%d %v", i, datum)
			},
		)
	}

	chops.TryRecv(ch).Match(
		func(el T) {
			t.Errorf("channel chould be closed, but received: %v", el)
		},
		func() {},
		func() {
			t.Error("at the end of draining, channel was empty but unclosed")
		},
	)
}
