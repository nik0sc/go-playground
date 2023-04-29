package testutils

import (
	"testing"
)

type FlakyT interface {
	Error(args ...any)
	Errorf(format string, args ...any)
	// Fail()
	// FailNow()
	// Failed() bool
	// Fatal(args ...any)
	// Fatalf(format string, args ...any)

	Log(args ...any)
	Logf(format string, args ...any)

	originalHelper
	T() *testing.T
}

type originalHelper interface {
	Helper()
}

type flakyT struct {
	t *testing.T
	// flakyT cannot define its own Helper method
	// as testing.T.Helper uses a call to runtime.Callers
	// with hardcoded number of frames to skip, so
	// it would mark func (ft *flakyT) Helper() as the
	// helper function and not its caller.
	// by embedding the original testing.T.Helper() in here
	// we keep the number of frames the same
	originalHelper
	allowed    int
	lastFailed bool
}

func (ft *flakyT) decr() bool {
	ft.lastFailed = true
	ft.allowed--
	if ft.allowed > 0 {
		ft.t.Helper()
		ft.t.Logf("test flaked, %d more times allowed", ft.allowed)
		return false
	} else {
		return true
	}
}

func (ft *flakyT) T() *testing.T {
	return ft.t
}

func (ft *flakyT) Errorf(format string, args ...any) {
	ft.t.Helper()
	if ft.decr() {
		ft.t.Errorf(format, args...)
	}
}

func (ft *flakyT) Error(args ...any) {
	ft.t.Helper()
	if ft.decr() {
		ft.t.Error(args...)
	}
}

func (ft *flakyT) Log(args ...any) {
	ft.t.Log(args...)
}

func (ft *flakyT) Logf(format string, args ...any) {
	ft.t.Logf(format, args...)
}

// Flaky allows a test to fail for maxTimes before reporting a failure.
// Flaky wraps the test functions passed to testing.T.Run like so:
//
//	t.Run("name", testutils.Flaky(maxTimes, func(testutils.FlakyT){
//	  ...
//	}))
//
// Flaky can also be used directly in a test:
//
//	func TestXxx(t *testing.T) {
//	  testutils.Flaky(maxTimes, func(t testutils.FlakyT){
//	    ...
//	  })
//	}
func Flaky(maxTimes int, testFunc func(FlakyT)) func(*testing.T) {
	ft := &flakyT{
		allowed: maxTimes,
	}

	return func(t *testing.T) {
		if ft.t == nil {
			ft.t = t
			ft.originalHelper = t
		}
		firstRun := true
		t.Helper()

		for ft.lastFailed || firstRun {
			firstRun = false
			ft.lastFailed = false // reset
			testFunc(ft)
		}
	}
}
