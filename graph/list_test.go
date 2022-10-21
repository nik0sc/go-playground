package graph

import (
	"reflect"
	"runtime"
	"sync/atomic"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestAdjacencyListDigraph_Create(t *testing.T) {
	g := NewAdjacencyListDigraph[string]()

	g.AddEdge("a", "b")
	g.AddEdge("a", "c")
	g.AddEdge("b", "a")
	g.AddEdge("c", "d")

	g.AddNode("z")

	assert.ElementsMatch(t,
		g.Nodes(),
		[]string{"a", "b", "c", "d", "z"})

	assert.ElementsMatch(t,
		g.Edges(),
		[][2]string{
			{"a", "b"},
			{"a", "c"},
			{"b", "a"},
			{"c", "d"},
		})

	t.Log(g.String())
}

func clrs1() *AdjacencyListDigraph[int] {
	g := NewAdjacencyListDigraph[int]()

	g.AddEdge(1, 2)
	g.AddEdge(1, 4)
	g.AddEdge(2, 5)
	g.AddEdge(3, 5)
	g.AddEdge(3, 6)
	g.AddEdge(4, 2)
	g.AddEdge(5, 4)
	g.AddEdge(6, 6)

	return g
}

func dag() *AdjacencyListDigraph[int] {
	g := NewAdjacencyListDigraph[int]()

	g.AddEdge(1, 2)
	g.AddEdge(2, 3)
	g.AddEdge(3, 4)
	g.AddEdge(2, 5)
	g.AddEdge(6, 5)
	g.AddEdge(5, 7)
	g.AddEdge(7, 4)

	return g
}

func linkedlist() *AdjacencyListDigraph[int] {
	g := NewAdjacencyListDigraph[int]()

	g.AddEdge(1, 2)
	g.AddEdge(2, 3)
	g.AddEdge(3, 4)
	g.AddEdge(4, 5)

	return g
}

func TestAdjacencyListDigraph_ShortestDistance(t *testing.T) {
	g := clrs1()

	distances, subgraph := g.ShortestDistance(1)

	assert.Equal(t, map[int]int{
		1: 0,
		2: 1,
		4: 1,
		5: 2,
	}, distances)

	assert.ElementsMatch(t,
		subgraph.Nodes(),
		[]int{1, 2, 4, 5})

	assert.ElementsMatch(t,
		subgraph.Edges(),
		[][2]int{
			{1, 2},
			{1, 4},
			{2, 5},
		})

	distances, subgraph = g.ShortestDistance(3)

	assert.Equal(t, map[int]int{
		3: 0,
		5: 1,
		6: 1,
		4: 2,
		2: 3,
	}, distances)

	assert.ElementsMatch(t,
		subgraph.Nodes(),
		[]int{2, 3, 4, 5, 6})

	assert.ElementsMatch(t,
		subgraph.Edges(),
		[][2]int{
			{3, 5},
			{3, 6},
			{5, 4},
			{4, 2},
		})
}

func TestAdjacencyListDigraph_TopologicalOrder(t *testing.T) {
	var order []int
	var err error

	order, err = dag().TopologicalOrder()
	assert.NoError(t, err)
	t.Log("dag:", order)

	order, err = linkedlist().TopologicalOrder()
	assert.NoError(t, err)
	t.Log("linkedlist:", order)

	order, err = clrs1().TopologicalOrder()
	assert.Empty(t, order)
	assert.ErrorIs(t, err, ErrCycleDetected)
}

func TestAdjacencyListDigraph_RemoveNode(t *testing.T) {
	tests := []struct {
		name   string
		setup  func() *AdjacencyListDigraph[int]
		remove int
		want   bool
		then   func(g *AdjacencyListDigraph[int])
	}{
		{
			name: "Remove not found",
			setup: func() *AdjacencyListDigraph[int] {
				g := NewAdjacencyListDigraph[int]()

				g.AddEdge(1, 2)

				return g
			},
			remove: 3,
			want:   false,
			then: func(g *AdjacencyListDigraph[int]) {
				assert.EqualValues(t, map[int][]int{
					1: {2},
					2: nil,
				}, g.adj)
			},
		},
		{
			name:   "Remove middle, parent has single child and it is the node",
			setup:  linkedlist,
			remove: 3,
			want:   true,
			then: func(g *AdjacencyListDigraph[int]) {
				assert.EqualValues(t, map[int][]int{
					1: {2},
					2: {}, // not nil
					4: {5},
					5: nil,
				}, g.adj)

				assert.ElementsMatch(t, g.Edges(), [][2]int{{1, 2}, {4, 5}})

				l := g.adj[2]
				assert.GreaterOrEqual(t, 1, cap(l))
				buf := (*int)(unsafe.Pointer(reflect.ValueOf(l).Pointer()))
				assert.Equal(t, *buf, 0, "truncated data was not zeroed")
				assert.Equal(t, l[:1], []int{0}, "truncated data was not zeroed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.setup()
			assert.Equal(t, tt.want, g.RemoveNode(tt.remove))
			tt.then(g)
		})
	}
}

func TestAdjacencyListDigraph_RemoveNode_GCFinalizer(t *testing.T) {
	g := NewAdjacencyListDigraph[*int]()

	one, two, three := 1, 2, 3

	var threeFinalized int64

	g.AddEdge(&one, &two)
	g.AddEdge(&one, &three)

	runtime.SetFinalizer(&three, func(_ *int) {
		atomic.StoreInt64(&threeFinalized, 1)
	})

	g.RemoveNode(&three)

	// runtime.KeepAlive(&three)

	runtime.GC()

	assert.EqualValues(t, 1, atomic.LoadInt64(&threeFinalized))
}
