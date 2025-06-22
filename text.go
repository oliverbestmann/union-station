package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	. "github.com/quasilyte/gmath"
	"image/color"
)

type Text struct {
	Offset Vec
	Text   string
	Face   text.Face
	Color  color.Color
}

func MeasureTexts(texts []Text) Vec {
	var size Vec
	for _, t := range texts {
		width, height := text.Measure(t.Text, t.Face, t.Face.Metrics().XHeight*2)
		size.X = max(size.X, t.Offset.X+width)
		size.Y += t.Offset.Y + height
	}

	return size
}

func DrawTexts(target *ebiten.Image, offset Vec, texts []Text) {
	pos := offset

	for _, t := range texts {
		DrawTextLeft(target, t.Text, t.Face, pos.Add(t.Offset), t.Color)

		// measure text to advance position
		_, height := text.Measure(t.Text, t.Face, t.Face.Metrics().XHeight*2)
		pos.Y += t.Offset.Y + height
	}
}
