package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	. "github.com/quasilyte/gmath"
	"image/color"
	"math"
)

var circleVertices []ebiten.Vertex
var circleIndices []uint16

var circleScratch []ebiten.Vertex

func DrawFillCircle(target *ebiten.Image, center Vec, radius float64, c color.Color) {
	if circleVertices == nil {
		var path vector.Path
		path.Arc(0, 0, 100, 0, 2*math.Pi, vector.Clockwise)
		circleVertices, circleIndices = path.AppendVerticesAndIndicesForFilling(nil, nil)
	}

	var tr ebiten.GeoM
	tr.Scale(0.01*radius, 0.01*radius)
	tr.Translate(center.X, center.Y)

	vertices := TransformVertices(tr, circleVertices, &circleScratch)

	r, g, b, a := toFloatRGBA(c)
	for idx := range vertices {
		vertices[idx].ColorR = float32(r)
		vertices[idx].ColorG = float32(g)
		vertices[idx].ColorB = float32(b)
		vertices[idx].ColorA = float32(a)

	}

	op := &ebiten.DrawTrianglesOptions{AntiAlias: true}
	target.DrawTriangles(vertices, circleIndices, whiteImage, op)
}
