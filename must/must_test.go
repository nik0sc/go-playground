package must

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test2(t *testing.T) {
	f := func() (int, error) {
		return 1, errors.New("oops")
	}

	var r1 int

	assert.PanicsWithError(t, "oops", func() {
		r1 = Must2(f())
	})

	assert.Equal(t, 0, r1)

	f2 := func() (int, error) {
		return 1, nil
	}

	r1 = Must2(f2())

	assert.Equal(t, 1, r1)
}

func Test3(t *testing.T) {
	f := func() (int, string, error) {
		return 1, "str", errors.New("oops")
	}

	var r1 int
	var r2 string

	assert.PanicsWithError(t, "oops", func() {
		r1, r2 = Must3(f())
	})

	assert.Equal(t, 0, r1)
	assert.Equal(t, "", r2)

	f2 := func() (int, string, error) {
		return 1, "str", nil
	}

	r1, r2 = Must3(f2())

	assert.Equal(t, 1, r1)
	assert.Equal(t, "str", r2)
}
