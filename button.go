package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	. "github.com/quasilyte/gmath"
	"image/color"
)

type ButtonColors struct {
	Normal   color.Color
	Hover    color.Color
	Disabled color.Color
}

type Button struct {
	Disabled bool
	Colors   ButtonColors
	Text     string
	Rect     Rect
	hover    bool
}

func NewButton(text string, loc Vec, colors ButtonColors) *Button {
	rect := Rect{
		Min: loc,
		Max: loc.Add(Vec{X: 128, Y: 32}),
	}

	button := &Button{
		Colors: colors,
		Rect:   rect,
		Text:   text,
	}

	return button
}

func (b *Button) Hover(loc Vec) bool {
	if b == nil {
		return false
	}

	hover := b.Rect.Contains(loc)

	b.hover = hover && !b.Disabled
	return hover
}

func (b *Button) IsClicked(loc Vec, clicked bool) bool {
	return clicked && b.Hover(loc)
}

func (b *Button) Draw(target *ebiten.Image) {
	if b == nil {
		return
	}

	fillColor := b.Colors.Normal
	switch {
	case b.Disabled:
		fillColor = b.Colors.Disabled
	case b.hover:
		fillColor = b.Colors.Hover
	}

	// draw the rectangle
	loc := b.Rect.Min.AsVec32()
	width := float32(b.Rect.Width())
	height := float32(b.Rect.Height())
	vector.DrawFilledRect(target, loc.X, loc.Y, width, height, fillColor, true)

	// draw the text
	DrawTextCenter(target, b.Text, Font16, b.Rect.Center(), color.White)
}
