package laminar

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.lepak.sg/playground/graph"
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

func TestGroupParentCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// g := NewGroup(ctx, NoLimit)
	g := NewGroup(ctx, 1)

	oneRunning := make(chan struct{})

	one := g.NewTask("one", func(ctx context.Context) error {
		close(oneRunning)
		<-ctx.Done()
		return ctx.Err()
	})

	two := g.NewTask("two", func(ctx context.Context) error {
		t.Error("two ran")
		return nil
	}).After(one)

	g.NewTask("three", func(ctx context.Context) error {
		t.Error("three ran")
		return nil
	}).After(two)

	assert.NoError(t, g.Start())

	<-oneRunning
	cancel() // cancel only after one has started, but before two can

	assert.ErrorIs(t, g.Wait(), context.Canceled)

	t.Log(g)

	goleak.VerifyNone(t)
}

func TestGroupTaskError(t *testing.T) {
	g := NewGroup(context.Background(), 2)

	one := g.NewTask("one", func(ctx context.Context) error {
		<-ctx.Done()
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

	assert.ErrorContains(t, g.Wait(), "oops")

	t.Log(g)

	goleak.VerifyNone(t)
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

func TestGroupDependTwice(t *testing.T) {
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
