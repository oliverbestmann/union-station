package main

import (
	"slices"
	"time"
)

type StationGraph struct {
	Stations []*Station
	edges    []StationEdge
}

func (sg *StationGraph) Insert(edge StationEdge) {
	if edge.One == nil || edge.Two == nil {
		panic("stations must not be nil")
	}

	if sg.Has(edge.One, edge.Two) {
		return
	}

	sg.edges = append(sg.edges, edge)
}

func (sg *StationGraph) Remove(one *Station, two *Station) {
	sg.edges = slices.DeleteFunc(sg.edges, func(edge StationEdge) bool {
		return edge.Is(one, two)
	})
}

func (sg *StationGraph) Edges() []StationEdge {
	return sg.edges
}

func (sg *StationGraph) Get(one, two *Station) (StationEdge, bool) {
	for _, edge := range sg.edges {
		if edge.Is(one, two) {
			return edge, true
		}
	}

	return StationEdge{}, false
}

func (sg *StationGraph) Has(one, two *Station) bool {
	_, ok := sg.Get(one, two)
	return ok
}

func (sg *StationGraph) EdgesOf(station *Station) []StationEdge {
	var edges []StationEdge
	for _, edge := range sg.edges {
		if edge.Contains(station) {
			edges = append(edges, edge)
		}
	}

	return edges
}

func (sg *StationGraph) HasConnections(station *Station) bool {
	for _, edge := range sg.edges {
		if edge.Contains(station) {
			return true
		}
	}

	return false
}

func (sg *StationGraph) TotalPrice() Coins {
	var coinsTotal Coins
	for _, edge := range sg.Edges() {
		coinsTotal += edge.Price()
	}

	return coinsTotal
}

func (sg *StationGraph) Clone() StationGraph {
	return StationGraph{
		Stations: slices.Clone(sg.Stations),
		edges:    slices.Clone(sg.edges),
	}
}

type StationEdge struct {
	Created time.Time
	One     *Station
	Two     *Station
}

func (edge StationEdge) Price() Coins {
	return priceOf(edge.One, edge.Two)
}

func (edge StationEdge) Contains(other *Station) bool {
	return edge.One == other || edge.Two == other
}

func (edge StationEdge) OtherStation(station *Station) *Station {
	if !edge.Contains(station) {
		panic("station is not part of the edge")
	}

	if edge.One == station {
		return edge.Two
	} else {
		return edge.One
	}
}

func (edge StationEdge) Is(one, two *Station) bool {
	return edge.One == one && edge.Two == two ||
		edge.One == two && edge.Two == one
}

func (edge StationEdge) Reversed() StationEdge {
	return StationEdge{One: edge.Two, Two: edge.One}
}

func (edge StationEdge) StartAt(one *Station) StationEdge {
	switch {
	case edge.One == one:
		return edge

	case edge.Two == one:
		return edge.Reversed()

	default:
		panic("edge is missing node One")
	}
}
