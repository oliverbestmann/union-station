package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func DrawStationConnection(target *ebiten.Image, toScreen ebiten.GeoM, one, two *Station, stationColor StationColor) {
	// work in screen space
	start := TransformVec(toScreen, one.Position).AsVec32()
	end := TransformVec(toScreen, two.Position).AsVec32()

	// calculate length & direction to lerp across the screen
	length := end.Sub(start).Len()
	direction := end.Sub(start).Normalized()

	var path vector.Path

	const segmentLen = 20

	for f := float32(0); f < length; f += segmentLen {
		a := start.Add(direction.Mulf(f))
		b := start.Add(direction.Mulf(min(f+segmentLen/2, length)))

		path.MoveTo(a.X, a.Y)
		path.LineTo(b.X, b.Y)
	}

	StrokePath(target, path, ebiten.GeoM{}, stationColor.Stroke, &vector.StrokeOptions{
		Width: 4.0,
	})
}
