package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	. "github.com/quasilyte/gmath"
)

type Button struct {
	Disabled bool
	text     string
	rect     Rect
	hover    bool
	clicked  bool
}

func NewButton(text string, loc Vec) *Button {
	rect := Rect{
		Min: loc,
		Max: loc.Add(Vec{X: 80, Y: 32}),
	}

	button := &Button{
		rect: rect,
		text: text,
	}

	return button
}

func (b *Button) Hover(loc Vec) bool {
	if b == nil {
		return false
	}

	hover := b.rect.Contains(loc)

	b.hover = hover && !b.Disabled
	return hover
}

func (b *Button) IsClicked(loc Vec, clicked bool) bool {
	if b == nil || b.Disabled {
		return false
	}

	b.clicked = clicked && b.rect.Contains(loc)
	return b.clicked
}

func (b *Button) Draw(target *ebiten.Image) {
	if b == nil {
		return
	}

	fillColor := rgbaOf(0x6f8b6eff)
	switch {
	case b.Disabled:
		fillColor = rgbaOf(0x808080ff)
	case b.clicked:
		fillColor = rgbaOf(0xb089abff)
	case b.hover:
		fillColor = rgbaOf(0x87a985ff)
	}

	loc := b.rect.Min.AsVec32()
	width := float32(b.rect.Width())
	height := float32(b.rect.Height())
	vector.DrawFilledRect(target, loc.X, loc.Y, width, height, fillColor, true)

	// draw text on the button image
	x := int(b.rect.Min.X + 12)
	y := int(height/2 + float32(Font.Metrics().Ascent.Round())/2 + loc.Y)
	etext.Draw(target, b.text, Font, x, y, rgbaOf(0xffffffff))

}
