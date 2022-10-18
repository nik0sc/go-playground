package graph

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"golang.org/x/exp/slices"
)

var ErrCycleDetected = errors.New("cycle detected")

// AdjacencyListDigraph is a directed graph that uses
// an adjacency list representation.
// V should be a small type (int-sized, one machine word)
// for best performance.
type AdjacencyListDigraph[V comparable] struct {
	adj map[V][]V
}

func NewAdjacencyListDigraph[V comparable]() *AdjacencyListDigraph[V] {
	return &AdjacencyListDigraph[V]{
		adj: make(map[V][]V),
	}
}

// AddNode tries to add a vertex to the graph, unconnected to any other vertex.
// It returns true if the node didn't exist and was successfully added.
func (g *AdjacencyListDigraph[V]) AddNode(node V) bool {
	_, ok := g.adj[node]
	if !ok {
		g.adj[node] = nil
	}

	return !ok
}

// AddEdge adds an edge to the graph.
func (g *AdjacencyListDigraph[V]) AddEdge(from, to V) {
	fromList := g.adj[from]
	if len(fromList) == 0 {
		g.adj[from] = []V{to}
		g.AddNode(to)
		return
	}

	if !g.AddNode(to) {
		for _, tail := range fromList {
			if tail == to {
				return
			}
		}
	}

	g.adj[from] = append(g.adj[from], to)
}

func (g *AdjacencyListDigraph[V]) RemoveNode(node V) bool {
	return false // TODO
}

func (g *AdjacencyListDigraph[V]) RemoveEdge(from, to V) bool {
	return false // TODO
}

func (g *AdjacencyListDigraph[V]) Nodes() []V {
	nodes := make([]V, 0, len(g.adj))

	for n := range g.adj {
		nodes = append(nodes, n)
	}

	return nodes
}

func (g *AdjacencyListDigraph[V]) Edges() [][2]V {
	edges := make([][2]V, 0, len(g.adj)) // just a guess for capacity

	for from, list := range g.adj {
		for _, to := range list {
			edges = append(edges, [2]V{from, to})
		}
	}

	return edges
}

func (g *AdjacencyListDigraph[V]) Has(node V) bool {
	_, ok := g.adj[node]
	return ok
}

func (g *AdjacencyListDigraph[V]) Neighbours(node V) ([]V, bool) {
	if l, ok := g.adj[node]; !ok {
		return nil, false
	} else if len(l) == 0 {
		return nil, true
	} else {
		return slices.Clone(l), true
	}
}

type line struct {
	node string
	outs []string
}

// String returns a string representation of the graph.
// If V implements [fmt.Stringer], it will be used, otherwise
// the default format for its underlying type is used.
// Lines are sorted in lexicographic order of their nodes.
// The neighbours of each node are also output in lexicographic order.
func (g *AdjacencyListDigraph[V]) String() string {
	var lines []line

	for node, to := range g.adj {
		toStr := make([]string, len(to))
		for i, neighbour := range to {
			toStr[i] = fmt.Sprint(neighbour)
		}
		slices.Sort(toStr)

		lines = append(lines, line{
			node: fmt.Sprint(node),
			outs: toStr,
		})
	}

	sort.Slice(lines, func(i, j int) bool {
		return lines[i].node < lines[j].node
	})

	var sb strings.Builder
	for i, line := range lines {
		sb.WriteString(line.node)
		sb.WriteString(" ->")
		for _, neighbour := range line.outs {
			sb.WriteRune(' ')
			sb.WriteString(neighbour)
		}
		if i < len(lines)-1 {
			sb.WriteRune('\n')
		}
	}

	return sb.String()
}

// ShortestDistance takes a node in the graph and returns a map of
// all nodes reachable from the first node to their shortest distance
// from that node, as well as a subgraph of reachable nodes.
// If the node doesn't exist, only that node is returned in the map
// with distance 0, and only that node is returned in the subgraph.
// This works with cyclic graphs as well.
func (g *AdjacencyListDigraph[V]) ShortestDistance(from V) (
	distances map[V]int, subgraph *AdjacencyListDigraph[V]) {
	// BFS as described in CLRS

	distances = make(map[V]int) // presence in this map = node is "greyed"
	distances[from] = 0
	subgraph = NewAdjacencyListDigraph[V]()
	subgraph.AddNode(from)
	q := []V{from}

	for len(q) != 0 {
		current := q[0]
		q = q[1:]

		for _, next := range g.adj[current] {
			_, ok := distances[next]
			if !ok {
				distances[next] = distances[current] + 1
				q = append(q, next)
				subgraph.AddEdge(current, next)
			}
		}
	}

	return
}

type DepthFirstNode[V comparable] struct {
	discover, finish int
	parent           V
}

func (d *DepthFirstNode[V]) String() string {
	return fmt.Sprintf("%d/%d (parent:%v)", d.discover, d.finish, d.parent)
}

type depthFirst[V comparable] struct {
	g      *AdjacencyListDigraph[V]
	states map[V]*DepthFirstNode[V] // node in map = coloured
	time   int
}

// DepthFirst ...
func (g *AdjacencyListDigraph[V]) DepthFirst(less func(a, b V) bool) map[V]*DepthFirstNode[V] {
	s := &depthFirst[V]{
		g:      g,
		states: make(map[V]*DepthFirstNode[V]),
	}

	vorder := g.Nodes()
	if less != nil {
		slices.SortFunc(vorder, less)
	}

	// depth first is not deterministic!
	for _, node := range vorder {
		if _, ok := s.states[node]; !ok {
			s.visit(node)
		}
	}

	return s.states
}

// TopologicalOrder tries to generate a topological order for all vertices.
// It may return [ErrCycleDetected] if the order cannot be generated because
// the graph contains a cycle.
func (g *AdjacencyListDigraph[V]) TopologicalOrder() (order []V, err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		if err2, ok := r.(error); ok {
			if errors.Is(err2, ErrCycleDetected) {
				order = nil
				err = err2
				return
			}
		}

		panic(r)
	}()

	// eventually all nodes will end up in this map.
	// value 0 (zero value, not present in map): not seen yet
	// value 1: seen in this branch of exploration that is ongoing
	// value 2: seen in a previous branch of exploration
	seen := make(map[V]int)

	// nodes to explore next
	toVisit := make(map[V]struct{}, len(g.adj))
	for v := range g.adj {
		toVisit[v] = struct{}{}
	}

	// order is filled up in reverse
	i := len(toVisit) - 1
	order = make([]V, len(toVisit))

	// name must be declared before function body
	var visit func(v V)
	visit = func(v V) {
		switch seen[v] {
		case 1:
			panic(ErrCycleDetected)
		case 2:
			// finished this node
			return
		default:
		}
		seen[v] = 1 // following the branch of this node

		for _, neighbour := range g.adj[v] {
			visit(neighbour)
		}

		order[i] = v
		i--
		if i == -1 && len(toVisit) != 1 {
			panic("reached start of order, but after the current node, toVisit still contains nodes")
		}

		seen[v] = 2 // fully explored the branch

		if _, ok := toVisit[v]; !ok {
			panic(fmt.Errorf("dequeue v not found: v=%v toVisit=%v", v, toVisit))
		}
		delete(toVisit, v)
	}

	for v := range toVisit {
		// map entries may be removed during iteration,
		// and if they haven't been reached by the iterator yet,
		// they won't be produced
		// note: visit does remove map entries
		visit(v)
	}

	return order, nil
}

func (df *depthFirst[V]) visit(node V) {
	df.time++
	df.states[node] = &DepthFirstNode[V]{
		discover: df.time,
	}

	for _, next := range df.g.adj[node] {
		if _, ok := df.states[next]; !ok {
			df.visit(next)
			df.states[next].parent = node
		}
	}

	df.time++
	df.states[node].finish = df.time
}
