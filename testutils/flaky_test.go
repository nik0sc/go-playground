package testutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlaky(t *testing.T) {
	i := 0

	ok := t.Run("", Flaky(10, func(t FlakyT) {
		i++
		if i <= 3 {
			t.Error("error")
		}
	}))

	// should pass
	assert.True(t, ok)
	assert.Equal(t, i, 4)
}

func TestFlaky_NotFlaky(t *testing.T) {
	i := 0

	ok := t.Run("", Flaky(10, func(t FlakyT) {
		i++
		t.T().Log("run")
	}))

	assert.True(t, ok)
	assert.Equal(t, i, 1)
}

func TestFlaky_CallFrames(t *testing.T) {
	i := 0
	ok := t.Run("", Flaky(2, func(t FlakyT) {
		i++
		if i <= 1 {
			assert.True(t, false)
		}
	}))

	assert.True(t, ok)
}
