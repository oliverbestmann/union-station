package main

type StationGraph struct {
	edges []*StationEdge
}

func (sg *StationGraph) Insert(one, two *Station) *StationEdge {
	edge := &StationEdge{
		One: one,
		Two: two,
	}

	sg.edges = append(sg.edges, edge)

	return edge
}

func (sg *StationGraph) Edges() []*StationEdge {
	return sg.edges
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
	One *Station
	Two *Station
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
