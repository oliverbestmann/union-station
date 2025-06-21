package main

import (
	"sort"
)

func computeMST(graph StationGraph) StationGraph {
	n := len(graph.Stations)

	uf := NewUnionFind(graph.Stations)

	// Step 1: Pre-union existing edges
	for _, edge := range graph.Edges() {
		uf.Union(edge.One, edge.Two)
	}

	// Step 2: Generate all possible edges
	allEdges := make([]StationEdge, 0, n*(n-1)/2)

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			allEdges = append(allEdges, StationEdge{
				One: graph.Stations[i],
				Two: graph.Stations[j],
			})
		}
	}

	// Step 3: Sort all edges by weight
	sort.Slice(allEdges, func(i, j int) bool {
		return allEdges[i].Price() < allEdges[j].Price()
	})

	// Step 4: Add edges without forming cycles

	// start with the existing graph
	mst := graph.Clone()

	for _, edge := range allEdges {
		if uf.Union(edge.One, edge.Two) {
			mst.Insert(edge)
		}
	}

	return mst
}

type UnionFind struct {
	parent map[*Station]*Station
}

func NewUnionFind(stations []*Station) *UnionFind {
	uf := &UnionFind{
		parent: map[*Station]*Station{},
	}

	for _, station := range stations {
		uf.parent[station] = station
	}

	return uf
}

func (uf *UnionFind) Find(x *Station) *Station {
	root := x
	for uf.parent[root] != root {
		root = uf.parent[root]
	}

	for x != uf.parent[x] {
		parent := uf.parent[x]
		uf.parent[x] = root
		x = parent
	}

	return root
}

func (uf *UnionFind) Union(x, y *Station) bool {
	rootX := uf.Find(x)
	rootY := uf.Find(y)

	if rootX == rootY {
		return false
	}

	uf.parent[rootY] = rootX
	return true
}
