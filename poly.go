package main

import (
	. "github.com/quasilyte/gmath"
	"sort"
)

// Cross product of OA and OB vectors (O, A, B are points)
func cross3(o, a, b Vec) float64 {
	return (a.X-o.X)*(b.Y-o.Y) - (a.Y-o.Y)*(b.X-o.X)
}

// ConvexHull returns the convex hull of a set of 2D points using Andrew's algorithm.
// The result is in counter-clockwise order.
func ConvexHull(points []Vec) []Vec {
	n := len(points)
	if n <= 1 {
		return append([]Vec(nil), points...)
	}

	// Sort points lexicographically (by X, then Y)
	sort.Slice(points, func(i, j int) bool {
		if points[i].X == points[j].X {
			return points[i].Y < points[j].Y
		}
		return points[i].X < points[j].X
	})

	var lower []Vec
	for _, p := range points {
		for len(lower) >= 2 && cross3(lower[len(lower)-2], lower[len(lower)-1], p) <= 0 {
			lower = lower[:len(lower)-1]
		}
		lower = append(lower, p)
	}

	var upper []Vec
	for i := n - 1; i >= 0; i-- {
		p := points[i]
		for len(upper) >= 2 && cross3(upper[len(upper)-2], upper[len(upper)-1], p) <= 0 {
			upper = upper[:len(upper)-1]
		}
		upper = append(upper, p)
	}

	// Concatenate lower and upper hulls, excluding the last point of each (repeats)
	hull := append(lower[:len(lower)-1], upper[:len(upper)-1]...)
	return hull
}

// PointInConvexHull checks whether point p is inside the convex hull defined by hull.
// The hull should be ordered counter-clockwise and be a closed or open loop (first == last optional).
func PointInConvexHull(hull []Vec, p Vec) bool {
	n := len(hull)
	if n < 3 {
		return false // Not a polygon
	}

	for i := 0; i < n; i++ {
		a := hull[i]
		b := hull[(i+1)%n]
		if cross3(a, b, p) < 0 {
			// Point is to the right of edge a->b, i.e., outside the hull
			return false
		}
	}
	return true
}
