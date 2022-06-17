package chops

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestIsClosed(t *testing.T) {
	tests := []struct {
		name      string
		chFactory func() chan struct{}
		want      bool
	}{
		{
			"Open",
			func() chan struct{} {
				return make(chan struct{})
			},
			false,
		},
		{
			"Closed",
			func() chan struct{} {
				ch := make(chan struct{})
				close(ch)
				return ch
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsClosed(tt.chFactory()); got != tt.want {
				t.Errorf("IsClosed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTryRecv(t *testing.T) {
	tests := []struct {
		name      string
		chFactory func() chan string
		want      string
		want1     Status
	}{
		{
			"Ok",
			func() chan string {
				ch := make(chan string, 1)
				ch <- "Hello"
				return ch
			},
			"Hello",
			Ok,
		},
		{
			"Closed",
			func() chan string {
				ch := make(chan string)
				close(ch)
				return ch
			},
			"",
			Closed,
		},
		{
			"Blocked",
			func() chan string {
				return make(chan string)
			},
			"",
			Blocked,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := TryRecv(tt.chFactory()).Get()
			if got != tt.want {
				t.Errorf("TryRecv() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("TryRecv() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestTrySend(t *testing.T) {
	tests := []struct {
		name      string
		chFactory func() chan string
		x         string
		wantStat  Status
	}{
		{
			"Ok",
			func() chan string {
				return make(chan string, 1)
			},
			"Hello",
			Ok,
		},
		{
			"Closed",
			func() chan string {
				ch := make(chan string)
				close(ch)
				return ch
			},
			"yeet",
			Closed,
		},
		{
			"Blocked",
			func() chan string {
				return make(chan string)
			},
			"oof",
			Blocked,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotStat := TrySend(tt.chFactory(), tt.x); gotStat != tt.wantStat {
				t.Errorf("TrySend() = %v, want %v", gotStat, tt.wantStat)
			}
		})
	}
}

func TestWait_ZeroWg(t *testing.T) {
	var wg sync.WaitGroup

	_, ok := <-Wait(&wg)
	assert.False(t, ok)

	goleak.VerifyNone(t)
}

func TestWait_Block(t *testing.T) {
	// to be run with -race
	var wg sync.WaitGroup
	exited := false

	wg.Add(1)
	go func() {
		exited = true
		wg.Done()
	}()

	_, ok := <-Wait(&wg)
	assert.True(t, exited)
	assert.False(t, ok)

	goleak.VerifyNone(t)
}

func TestWait_Reuse(t *testing.T) {
	var wg sync.WaitGroup
	var exit1, exit2 bool

	wg.Add(1)
	ch1 := Wait(&wg)
	go func() {
		exit1 = true
		wg.Done()
	}()

	_, ok := <-ch1
	assert.True(t, exit1)
	assert.False(t, ok)

	wg.Add(1)
	ch2 := Wait(&wg)
	go func() {
		exit2 = true
		wg.Done()
	}()

	_, ok = <-ch2
	assert.True(t, exit2)
	assert.False(t, ok)

	goleak.VerifyNone(t)
}
