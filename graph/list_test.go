package graph

import (
	"testing"

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
