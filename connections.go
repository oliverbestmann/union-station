package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"image/color"
	"math"
	"time"
)

func DrawStationConnection(target *ebiten.Image, toScreen ebiten.GeoM, one, two *Station, offset time.Duration, thin bool, color color.Color) {
	// work in screen space
	start := TransformVec(toScreen, one.Position).AsVec32()
	end := TransformVec(toScreen, two.Position).AsVec32()

	// calculate length & direction to lerp across the screen
	length := end.Sub(start).Len()
	direction := end.Sub(start).Normalized()

	var path vector.Path

	const segmentLen = 20

	// calculate starting offset
	rem := math.Remainder(offset.Seconds()*5.0, segmentLen)
	f := float32(rem)

	for ; f < length; f += segmentLen {
		a := start.Add(direction.Mulf(f))
		b := start.Add(direction.Mulf(min(f+segmentLen/2, length)))

		path.MoveTo(a.X, a.Y)
		path.LineTo(b.X, b.Y)
	}

	vop := &vector.StrokeOptions{
		Width: 4.0,
	}

	if thin {
		vop.Width = 2.0
	}

	StrokePath(target, path, ebiten.GeoM{}, color, vop)
}
