package main

import (
	"github.com/hajimehoshi/ebiten/v2"
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
	Position Vec
	Size     Vec
	hover    bool
}

func NewButton(text string, colors ButtonColors) *Button {
	button := &Button{
		Colors: colors,
		Size:   Vec{X: 192, Y: 48},
		Text:   text,
	}

	return button
}

func (b *Button) Hover(loc Vec) bool {
	if b == nil {
		return false
	}

	rect := Rect{Min: b.Position, Max: b.Position.Add(b.Size)}
	hover := rect.Contains(loc)

	b.hover = hover && !b.Disabled
	return hover
}

func (b *Button) IsClicked(loc Vec, clicked bool) bool {
	if b != nil && b.Disabled {
		return false
	}

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

	// draw a shadow for the rectangle
	DrawRoundRect(target, b.Position.Add(vecSplat(4)), b.Size, ShadowColor)

	// draw the rectangle
	hoverOffset := vecSplat(iff(b.hover, 2.0, 0))
	DrawRoundRect(target, b.Position.Add(hoverOffset), b.Size, fillColor)

	// draw the text
	pos := b.Position.Add(b.Size.Mulf(0.5).Add(hoverOffset))
	DrawTextCenter(target, b.Text, Font24, pos, BackgroundColor)
}

func LayoutButtonsColumn(origin Vec, gap float64, buttons ...*Button) {
	pos := origin

	for _, button := range buttons {
		button.Position = pos
		pos.Y += button.Size.Y + gap
	}
}
