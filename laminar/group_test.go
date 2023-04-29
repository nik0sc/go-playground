package laminar

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.lepak.sg/playground/graph"
	"go.lepak.sg/playground/testutils"
	"go.uber.org/goleak"
)

func TestGroupCycleDetection(t *testing.T) {
	g := NewGroup(context.Background(), NoLimit)

	one := g.NewTask("one", func(ctx context.Context) error {
		t.Error("one ran")
		return nil
	})

	two := g.NewTask("two", func(ctx context.Context) error {
		t.Error("two ran")
		return nil
	})

	one.After(two)
	two.After(one)

	assert.ErrorIs(t, g.Start(), graph.ErrCycleDetected)

	goleak.VerifyNone(t)
}

// flaky
func TestGroupParentCancellation(t *testing.T) {
	testutils.Flaky(10, func(t testutils.FlakyT) {
		ctx, cancel := context.WithCancel(context.Background())

		g := NewGroup(ctx, 1)

		one := g.NewTask("one", func(ctx context.Context) error {
			cancel() // cancel only after one has started
			// two may already be waiting for the errgroup,
			// and once in the errgroup, one's doneCh may be selected
			// instead of ctx.Done, so it may run
			// three should never run
			return nil
		})

		two := g.NewTask("two", func(ctx context.Context) error {
			t.Log("two ran")
			return nil
		}).After(one)

		g.NewTask("three", func(ctx context.Context) error {
			t.Error("three ran")
			return nil
		}).After(two)

		assert.NoError(t, g.Start())

		assert.ErrorIs(t, g.Wait(), context.Canceled)

		t.Log(g)

		goleak.VerifyNone(t)
	})
}

// flaky
func TestGroupTaskError(t *testing.T) {
	testutils.Flaky(10, func(ft testutils.FlakyT) {
		g := NewGroup(context.Background(), 2)

		one := g.NewTask("one", func(ctx context.Context) error {
			return nil
		})

		two := g.NewTask("two", func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)
			return errors.New("oops")
		})

		three := g.NewTask("three", func(ctx context.Context) error {
			t.Error("three ran") // dequeued but exits in errgroup waiting for dependents
			return nil
		}).After(one, two)

		four := g.NewTask("four", func(ctx context.Context) error {
			t.Error("four ran") // dequeued but may be waiting for errgroup
			return nil
		}).After(three)

		g.NewTask("five", func(ctx context.Context) error {
			t.Error("five ran") // never dequeued
			return nil
		}).After(four)

		assert.NoError(t, g.Start())

		assert.ErrorContains(t, g.Wait(), "two: oops")

		t.Log(g)

		goleak.VerifyNone(t)
	})
}

func TestGroupEmpty(t *testing.T) {
	g := NewGroup(context.Background(), NoLimit)
	assert.NoError(t, g.Start())
	assert.NoError(t, g.Wait())
	goleak.VerifyNone(t)
}

func TestGroupStartOnce(t *testing.T) {
	panics := []func(*Group){
		func(g *Group) {
			g.NewTask("bad", func(ctx context.Context) error {
				t.Error("how did this run?")
				return nil
			})
		},
		func(g *Group) {
			g.Start()
		},
	}

	noPanics := []func(*Group){
		func(g *Group) {
			assert.NoError(t, g.Wait())
		},
	}

	setup := func() *Group {
		g := NewGroup(context.Background(), NoLimit)
		assert.NoError(t, g.Start())
		assert.NoError(t, g.Wait())
		return g
	}

	for i, f := range panics {
		g := setup()
		assert.Panicsf(t, func() {
			f(g)
		}, "#%d did not panic", i)
	}

	for _, f := range noPanics {
		g := setup()
		f(g)
	}
}

func TestTaskDependTwice(t *testing.T) {
	g := NewGroup(context.Background(), NoLimit)

	var oneTimes, twoTimes uint64

	one := g.NewTask("one", func(ctx context.Context) error {
		atomic.AddUint64(&oneTimes, 1)
		return nil
	})

	two := g.NewTask("two", func(ctx context.Context) error {
		atomic.AddUint64(&twoTimes, 1)
		return nil
	}).After(one, one)

	assert.NoError(t, g.Start())

	assert.NoError(t, g.Wait())

	assert.EqualValues(t, 1, oneTimes)
	assert.EqualValues(t, 1, twoTimes)

	t.Log(g)
	t.Logf("two.waitFor=%v", two.waitFor)

	goleak.VerifyNone(t)
}

func TestTaskHighOutdegree(t *testing.T) {
	g := NewGroup(context.Background(), 2)

	one := g.NewTask("1", func(ctx context.Context) error {
		return nil
	})

	two := g.NewTask("2", func(ctx context.Context) error {
		return nil
	}).After(one)

	three := g.NewTask("3", func(ctx context.Context) error {
		return nil
	}).After(one)

	four := g.NewTask("4", func(ctx context.Context) error {
		return nil
	}).After(one)

	five := g.NewTask("5", func(ctx context.Context) error {
		return nil
	}).After(two, three, four)

	assert.NoError(t, g.Start())

	assert.NoError(t, g.Wait())

	// test chan is copied correctly
	assert.Len(t, two.waitFor, 1)
	assert.Len(t, three.waitFor, 1)
	assert.Len(t, four.waitFor, 1)
	assert.Equal(t, one.wgChan, two.waitFor[0])
	assert.Equal(t, two.waitFor[0], three.waitFor[0])
	assert.Equal(t, three.waitFor[0], four.waitFor[0])
	assert.ElementsMatch(t, []<-chan struct{}{two.wgChan, three.wgChan, four.wgChan}, five.waitFor)
	// test chan is created on demand
	assert.Nil(t, five.wgChan)

	t.Log(g)

	goleak.VerifyNone(t)
}
