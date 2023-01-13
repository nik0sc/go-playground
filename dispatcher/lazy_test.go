package dispatcher

import (
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	ctr "go.lepak.sg/playground/counter"
	"go.lepak.sg/playground/testutils"
	"golang.org/x/exp/maps"
)

type stringKeyer struct {
	value string
}

func (s *stringKeyer) Key() string {
	return s.value[:1]
}

type stringEvent struct {
	id    uint64
	value string
}

type stringAcceptor struct {
	t      *testing.T
	id     uint64
	key    string
	ev     chan stringEvent
	closed uint64
}

func (a *stringAcceptor) Accept(item Keyer) error {
	if atomic.LoadUint64(&a.closed) == 1 {
		return errors.New("closed")
	}

	k, ok := item.(*stringKeyer)
	if !ok {
		return errors.New("wrong type")
	}

	if k.Key() != a.key {
		return errors.New("wrong key")
	}

	a.ev <- stringEvent{a.id, k.value}

	return nil
}

func (a *stringAcceptor) Close() {
	a.t.Logf("closing: key=%q id=%d", a.key, a.id)
	if !atomic.CompareAndSwapUint64(&a.closed, 0, 1) {
		a.t.Logf("already closed! key=%q id=%d", a.key, a.id)
	}
}

type errorAcceptor struct {
	t *testing.T
}

func (a *errorAcceptor) Accept(item Keyer) error {
	return errors.New("oops")
}

func (a *errorAcceptor) Close() {
	a.t.Fatal("unexpected close")
}

const (
	seed = 58913
)

func TestLazy(t *testing.T) {
	tests := []struct {
		name           string
		windowSize     int
		keyCardinality int
		chanSize       int
		do             func(t *testing.T, l *Lazy)
		after          func(t *testing.T, ch chan stringEvent)
	}{
		{
			name:           "empty",
			windowSize:     1,
			keyCardinality: 1,
			chanSize:       1,
			do: func(t *testing.T, l *Lazy) {
			},
			after: func(t *testing.T, ch chan stringEvent) {
				assert.Len(t, ch, 0)
			},
		},
		{
			name:           "single",
			windowSize:     1,
			keyCardinality: 1,
			chanSize:       3,
			do: func(t *testing.T, l *Lazy) {
				acceptMany(t, l, "apple", "banana", "apricot")
			},
			after: func(t *testing.T, ch chan stringEvent) {
				testutils.Drain(t, []stringEvent{
					{1, "apple"},
					{2, "banana"},
					{3, "apricot"},
				}, ch)
			},
		},
		{
			name:           "more lazy",
			windowSize:     2,
			keyCardinality: 3,
			chanSize:       10,
			do: func(t *testing.T, l *Lazy) {
				acceptMany(t, l,
					"apple",
					"banana",
					"apricot",
					"blueberry",
					"cherry",
					"blackcurrant",
					"avocado",
					"breadfruit",
				)
			},
			after: func(t *testing.T, ch chan stringEvent) {
				testutils.Drain(t, []stringEvent{
					{1, "apple"},
					{2, "banana"},
					{1, "apricot"},
					{2, "blueberry"},
					{3, "cherry"},
					{2, "blackcurrant"},
					{4, "avocado"},
					{2, "breadfruit"},
				}, ch)
			},
		},
		{
			name:           "concurrent",
			windowSize:     10,
			keyCardinality: 4,
			chanSize:       10200,
			do: func(t *testing.T, l *Lazy) {
				fruits := []string{
					"apple", "apricot", "avocado",
					"banana", "blueberry", "blackcurrant",
					"cherry", "coconut", "cantaloupe",
					"durian", "date", "dragonfruit",
				}

				const (
					goroutines = 4
					iterations = 10000
				)

				rand.Seed(seed)

				count := new(uint64)
				barrier := make(chan struct{})
				var wg sync.WaitGroup

				wg.Add(goroutines)
				for i := 0; i < goroutines; i++ {
					go func() {
						<-barrier
						for {
							fruit := fruits[rand.Intn(len(fruits))]
							err := l.Accept(&stringKeyer{fruit})
							assert.NoError(t, err)
							if atomic.AddUint64(count, 1) >= iterations {
								wg.Done()
								return
							}
						}
					}()
				}

				close(barrier)
				wg.Wait()
			},
			after: func(t *testing.T, ch chan stringEvent) {
				events := make([]stringEvent, 0, len(ch))
				for e := range ch {
					events = append(events, e)
				}

				c := ctr.Counter(events)
				k := 100
				if len(c) < k {
					k = len(c)
				}
				top := ctr.TopK(c, k)
				for _, e := range top {
					t.Log(e)
				}
			},
		},
		{
			name:           "concurrent create",
			windowSize:     10,
			keyCardinality: 10,
			chanSize:       20,
			do: func(t *testing.T, l *Lazy) {
				const goroutines = 10
				barrier := make(chan struct{})
				var wg sync.WaitGroup
				wg.Add(goroutines)
				for i := 0; i < goroutines; i++ {
					go func() {
						<-barrier
						assert.NoError(t, l.Accept(&stringKeyer{"a"}))
						wg.Done()
					}()
				}
				close(barrier)
				wg.Wait()
			},
			after: func(t *testing.T, ch chan stringEvent) {
				events := make([]stringEvent, 0, len(ch))
				for e := range ch {
					events = append(events, e)
				}

				assert.Len(t, events, 10)

				assert.Equal(t, map[stringEvent]int{
					{id: 1, value: "a"}: 10,
				}, ctr.Counter(events))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outchan := make(chan stringEvent, tt.chanSize)

			id := new(uint64)

			l := NewLazy(func(s string) (Acceptor, error) {
				return &stringAcceptor{
					t:   t,
					id:  atomic.AddUint64(id, 1),
					key: s,
					ev:  outchan}, nil
			}, tt.windowSize, tt.keyCardinality)

			tt.do(t, l)
			l.Close()
			close(outchan)
			tt.after(t, outchan)
		})
	}
}

func checkLazy(t *testing.T, l *Lazy, ss string) {
	all := l.window.(interface{ GetAll() map[string]int }).GetAll()
	w := maps.Clone(l.active)

	assert.ElementsMatchf(t, maps.Keys(all), maps.Keys(w),
		"map broken after accepting %q", ss)
}

func acceptMany(t *testing.T, l *Lazy, s ...string) {
	for _, ss := range s {
		l.Accept(&stringKeyer{ss})
		checkLazy(t, l, ss)
	}
}

func TestLazy_FactoryError(t *testing.T) {
	var l *Lazy

	l = NewLazy(func(s string) (Acceptor, error) {
		return nil, errors.New("oops")
	}, 1, 1)
	assert.ErrorContains(t, l.Accept(&stringKeyer{"a"}), "oops")

	l = NewLazy(func(s string) (Acceptor, error) {
		panic("oops")
	}, 1, 1)
	assert.ErrorContains(t, l.Accept(&stringKeyer{"a"}), "oops")
}

func TestLazy_AcceptError(t *testing.T) {
	l := NewLazy(func(s string) (Acceptor, error) {
		return &errorAcceptor{t}, nil
	}, 1, 1)
	assert.ErrorContains(t, l.Accept(&stringKeyer{"a"}), "oops")
}
