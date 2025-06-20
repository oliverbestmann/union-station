package main

import "container/heap"

type EdgeHeap []*StationEdge

func (h EdgeHeap) Len() int           { return len(h) }
func (h EdgeHeap) Less(i, j int) bool { return h[i].Price() < h[j].Price() }
func (h EdgeHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *EdgeHeap) Push(x any)        { *h = append(*h, x.(*StationEdge)) }
func (h *EdgeHeap) Pop() any {
	old := *h
	n := len(old)
	edge := old[n-1]
	*h = old[0 : n-1]
	return edge
}

// Prim's algorithm
func computeMST(stations []*Station) StationGraph {
	if len(stations) == 0 {
		return StationGraph{}
	}

	visited := make(map[*Station]bool)
	edgeHeap := &EdgeHeap{}

	start := stations[0]
	visited[start] = true

	// Initialize the heap with edges from the start node
	for _, s := range stations {
		if s != start {
			heap.Push(edgeHeap, &StationEdge{One: start, Two: s})
		}
	}

	var graph StationGraph

	for len(*edgeHeap) > 0 && len(visited) < len(stations) {
		edge := heap.Pop(edgeHeap).(*StationEdge)
		if visited[edge.Two] {
			continue
		}

		visited[edge.One] = true
		graph.Insert(edge.One, edge.Two)

		// Add new edges from the newly visited node
		for _, s := range stations {
			if !visited[s] {
				heap.Push(edgeHeap, &StationEdge{One: edge.Two, Two: s})
			}
		}
	}

	return graph
}
