package main

import (
	"slices"
	"time"
)

type StationGraph struct {
	edges []*StationEdge
}

func (sg *StationGraph) Insert(one, two *Station) (*StationEdge, bool) {
	if one == nil || two == nil {
		panic("stations must not be nil")
	}

	if existing := sg.Get(one, two); existing != nil {
		return existing, false
	}

	edge := &StationEdge{
		One: one,
		Two: two,
	}

	sg.edges = append(sg.edges, edge)

	return edge, true
}

func (sg *StationGraph) Remove(one *Station, two *Station) {
	sg.edges = slices.DeleteFunc(sg.edges, func(edge *StationEdge) bool {
		return edge.Is(one, two)
	})
}

func (sg *StationGraph) Edges() []*StationEdge {
	return sg.edges
}

func (sg *StationGraph) Get(one, two *Station) *StationEdge {
	for _, edge := range sg.edges {
		if edge.Is(one, two) {
			return edge
		}
	}

	return nil
}

func (sg *StationGraph) Has(one, two *Station) bool {
	return sg.Get(one, two) != nil
}

func (sg *StationGraph) EdgesOf(station *Station) []*StationEdge {
	var edges []*StationEdge
	for _, edge := range sg.edges {
		if edge.Contains(station) {
			edges = append(edges, edge)
		}
	}

	return edges
}

type StationEdge struct {
	Created time.Time
	One     *Station
	Two     *Station
}

func (edge *StationEdge) Contains(other *Station) bool {
	return edge.One == other || edge.Two == other
}

func (edge *StationEdge) OtherStation(station *Station) *Station {
	if !edge.Contains(station) {
		panic("station is not part of the edge")
	}

	if edge.One == station {
		return edge.Two
	} else {
		return edge.One
	}
}

func (edge *StationEdge) Is(one, two *Station) bool {
	return edge.One == one && edge.Two == two ||
		edge.One == two && edge.Two == one
}
