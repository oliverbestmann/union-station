package main

import (
	. "github.com/quasilyte/gmath"
	"math/rand/v2"
)

type Station struct {
	Position Vec

	// the village that belongs to this station
	Village *Village
}

func GenerateStations(rng *rand.Rand, villages []*Village) []*Station {
	var stations []*Station

	for _, village := range villages {
		stations = append(stations, GenerateStationsForVillage(rng, village)...)
	}

	return stations
}

func GenerateStationsForVillage(rng *rand.Rand, village *Village) []*Station {
	var loc Vec

	for {
		loc = Vec{
			X: randf(rng, village.BBox.Min.X, village.BBox.Max.X),
			Y: randf(rng, village.BBox.Min.Y, village.BBox.Max.Y),
		}

		if PointInConvexHull(village.Hull, loc) {
			break
		}
	}

	return []*Station{
		{
			Position: loc,
			Village:  village,
		},
	}
}
