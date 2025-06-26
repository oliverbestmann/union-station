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
	OnClick  func()
	Image    *ebiten.Image
	hover    bool
	pressed  bool

	imageCached       *ebiten.Image
	imageCachedValues buttonCacheValues
}

type buttonCacheValues struct {
	Colors   ButtonColors
	Text     string
	Size     Vec
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

func (b *Button) Clicked(cursor CursorState) bool {
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

		if b.OnClick != nil {
			b.OnClick()
		}

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

	values := buttonCacheValues{
		Colors:   b.Colors,
		Text:     b.Text,
		Size:     b.Size,
		Disabled: b.Disabled,
		hover:    b.hover,
		pressed:  b.pressed,
	}

	if b.imageCachedValues != values {
		b.imageCachedValues = values

		buttonSize := b.Size.Add(vecSplat(4))

		if b.imageCached == nil || !imageSizeOf(b.imageCached).EqualApprox(buttonSize) {
			if b.imageCached != nil {
				b.imageCached.Deallocate()
			}

			b.imageCached = ebiten.NewImage(int(buttonSize.X), int(buttonSize.Y))
		}

		b.imageCached.Clear()

		if b.Colors.Shadow != nil {
			// draw a shadow for the rectangle
			DrawRoundRect(b.imageCached, vecSplat(4), b.Size, b.Colors.Shadow)
		}

		// draw the rectangle
		hoverOffset := vecSplat(iff(b.pressed, 2.0, 0))
		DrawRoundRect(b.imageCached, hoverOffset, b.Size, fillColor)

		if b.Image != nil {
			pos := b.Size.Mulf(0.5).Add(hoverOffset).Sub(imageSizeOf(b.Image).Mulf(0.5))
			var op ebiten.DrawImageOptions
			op.GeoM.Translate(pos.X, pos.Y)
			b.imageCached.DrawImage(b.Image, &op)
		} else {
			// draw the text
			pos := b.Size.Mulf(0.5).Add(hoverOffset)
			DrawTextCenter(b.imageCached, b.Text, Font24, pos, b.Colors.Text)
		}
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(b.Position.X, b.Position.Y)
	op.ColorScale.ScaleAlpha(float32(b.Alpha))
	target.DrawImage(b.imageCached, op)
}

func (b *Button) WithOnClick(onClick func()) *Button {
	b.OnClick = onClick
	return b
}

func (b *Button) WithAutoSize() *Button {
	width := MeasureText(Font24, b.Text).X
	b.Size = Vec{X: width + 64, Y: 48}
	return b
}

func LayoutButtonsColumn(origin Vec, gap float64, buttons ...*Button) {
	pos := origin

	var maxWidth float64

	for _, button := range buttons {
		button.Position = pos
		pos.Y += button.Size.Y + gap

		maxWidth = max(maxWidth, button.Size.X)
	}

	for _, button := range buttons {
		button.Size.X = maxWidth
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
