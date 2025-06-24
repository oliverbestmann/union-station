package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	. "github.com/quasilyte/gmath"
	"image/color"
)

type ButtonColors struct {
	Normal   color.Color
	Hover    color.Color
	Text     color.Color
	Disabled color.Color
	Shadow   color.Color
}

type Button struct {
	Colors   ButtonColors
	Text     string
	Position Vec
	Size     Vec
	Alpha    float64
	Disabled bool
	hover    bool
	pressed  bool
}

func NewButton(text string, colors ButtonColors) *Button {
	button := &Button{
		Colors: colors,
		Size:   Vec{X: 192, Y: 48},
		Text:   text,
		Alpha:  1,
	}

	return button
}

func (b *Button) Hover(cursor CursorState) bool {
	if b == nil {
		return false
	}

	rect := Rect{Min: b.Position, Max: b.Position.Add(b.Size)}
	hover := rect.Contains(cursor.Position)

	b.hover = hover && !b.Disabled
	return hover
}

func (b *Button) IsClicked(cursor CursorState) bool {
	if b == nil || b.Disabled {
		return false
	}

	hover := b.Hover(cursor)

	switch {
	case !hover:
		b.pressed = false

	case cursor.JustPressed:
		b.pressed = true

	case b.pressed && cursor.JustReleased:
		b.pressed = false
		return true
	}

	return false
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

	if b.Colors.Shadow != nil {
		// draw a shadow for the rectangle
		DrawRoundRect(target, b.Position.Add(vecSplat(4)), b.Size, scaleColorWithAlpha(b.Colors.Shadow, b.Alpha))
	}

	// draw the rectangle
	hoverOffset := vecSplat(iff(b.pressed, 2.0, 0))
	DrawRoundRect(target, b.Position.Add(hoverOffset), b.Size, scaleColorWithAlpha(fillColor, b.Alpha))

	// draw the text
	pos := b.Position.Add(b.Size.Mulf(0.5).Add(hoverOffset))
	DrawTextCenter(target, b.Text, Font24, pos, scaleColorWithAlpha(b.Colors.Text, b.Alpha))
}

func LayoutButtonsColumn(origin Vec, gap float64, buttons ...*Button) {
	pos := origin

	for _, button := range buttons {
		button.Position = pos
		pos.Y += button.Size.Y + gap
	}
}

func ColorToRGBA64(c color.Color) (r, g, b, a float64) {
	ir, ig, ib, ia := c.RGBA()

	r = float64(ir) / 0xffff
	g = float64(ig) / 0xffff
	b = float64(ib) / 0xffff
	a = float64(ia) / 0xffff

	return
}

func ColorToRGBA32(c color.Color) (r, g, b, a float32) {
	ir, ig, ib, ia := c.RGBA()

	r = float32(ir) / 0xffff
	g = float32(ig) / 0xffff
	b = float32(ib) / 0xffff
	a = float32(ia) / 0xffff

	return
}

func ApplyColorToVertices(vertices []ebiten.Vertex, c color.Color) {
	r, g, b, a := ColorToRGBA32(c)

	for idx := range vertices {
		v := &vertices[idx]
		v.ColorR, v.ColorG, v.ColorB, v.ColorA = r, g, b, a
	}
}

func scaleColorWithAlpha(c color.Color, alpha float64) color.Color {
	r, g, b, a := ColorToRGBA64(c)

	return color.RGBA64{
		R: uint16(r * alpha * 0xffff),
		G: uint16(g * alpha * 0xffff),
		B: uint16(b * alpha * 0xffff),
		A: uint16(a * alpha * 0xffff),
	}
}
