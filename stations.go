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
		newStations, _, _ := MaxOf(
			Repeat(5, func() []*Station { return generateStations(rng, clip, village) }),
			stationScore,
		)

		stations = append(stations, newStations...)
	}

	return stations
}

func stationScore(stations []*Station) float64 {
	var distanceSum float64

	for _, a := range stations {
		for _, b := range stations {
			distanceSum += a.Position.DistanceTo(b.Position)
		}
	}

	return distanceSum
}

func generateStations(rng *rand.Rand, clip Rect, village *Village) []*Station {
	var loc Vec

	// get the segments that lay within the clip bounds
	segments := segmentsWithinClip(village, clip)
	if len(segments) < 10 {
		return nil
	}

	populationCount := populationCountOf(segments)
	if populationCount < 50 {
		return nil
	}

	stationCount := populationCount/1000 + 1

	var stations []*Station

	for range stationCount {
		for {
			loc = Choose(rng, segments...).BBox().Center()

			if !clip.Contains(loc) {
				// discard this, it is outside of the region
				continue
			}

			if PointInConvexHull(village.Hull, loc) {
				break
			}
		}

		stations = append(stations, &Station{
			Position: loc,
			Village:  village,
		})
	}

	return stations
}

func segmentsWithinClip(village *Village, clip Rect) []*Segment {
	var segments []*Segment

	// The village is outside of the given rectangle if no point of the
	// villages hull is inside the rect. This is not 100% fool proof, but good enough
	for _, segment := range village.Segments {
		if clip.Contains(segment.Center()) {
			segments = append(segments, segment)
		}
	}

	return segments
}
