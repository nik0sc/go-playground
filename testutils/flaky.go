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
	// Log(args ...any)
	// Logf(format string, args ...any)

	T() *testing.T
}

type flakyT struct {
	t          *testing.T
	allowed    int
	lastFailed bool
}

func (ft *flakyT) decr() bool {
	ft.lastFailed = true
	ft.t.Log("test flaked")
	ft.allowed--
	return ft.allowed <= 0
}

func (ft *flakyT) T() *testing.T {
	return ft.t
}

func (ft *flakyT) Errorf(format string, args ...any) {
	if ft.decr() {
		ft.t.Errorf(format, args...)
	}
}

func (ft *flakyT) Error(args ...any) {
	if ft.decr() {
		ft.t.Error(args...)
	}
}

// Flaky allows a test to fail for maxTimes before reporting a failure.
// Flaky wraps the test functions passed to testing.T.Run like so:
//
//	t.Run("name", testutils.Flaky(maxTimes, func(testutils.FlakyT){
//	  ...
//	}))
func Flaky(maxTimes int, testFunc func(FlakyT)) func(*testing.T) {
	ft := &flakyT{
		allowed: maxTimes,
	}

	return func(t *testing.T) {
		if ft.t == nil {
			ft.t = t
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
