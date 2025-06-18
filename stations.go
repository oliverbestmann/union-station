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

func GenerateStations(rng *rand.Rand, clip Rect, villages []*Village) []*Station {
	var stations []*Station

	for _, village := range villages {
		stations = append(stations, GenerateStationsForVillage(rng, clip, village)...)
	}

	return stations
}

func GenerateStationsForVillage(rng *rand.Rand, clip Rect, village *Village) []*Station {
	var loc Vec

	// check if the village is outside of the clip region
	var inside bool
	for _, point := range village.Hull {
		if clip.Contains(point) {
			inside = true
			break
		}
	}

	if !inside {
		return nil
	}

	for {
		loc = Vec{
			X: randf(rng, village.BBox.Min.X, village.BBox.Max.X),
			Y: randf(rng, village.BBox.Min.Y, village.BBox.Max.Y),
		}

		if !clip.Contains(loc) {
			// discard this, it is outside of the region
			continue
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
