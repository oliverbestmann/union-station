package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	. "github.com/quasilyte/gmath"
)

type Button struct {
	Image    *ebiten.Image
	Callback func()
	Rect     Rect
}

func NewButton(image *ebiten.Image, loc Vec, callback func(self *Button)) *Button {
	rect := Rect{
		Min: loc,
		Max: loc.Add(imageSizeOf(image)),
	}

	button := &Button{
		Image: image,
		Rect:  rect,
	}

	button.Callback = func() { callback(button) }

	return button
}

func (b *Button) Click(toScreen ebiten.GeoM, screenClick Vec) bool {
	if !b.Rect.Contains(screenClick) {
		return false
	}

	b.Callback()
	return true
}

func (b *Button) Draw(target *ebiten.Image, toScreen ebiten.GeoM) {
	imageSize := imageSizeOf(b.Image)

	op := ebiten.DrawImageOptions{}
	op.GeoM.Scale(1/imageSize.X, 1/imageSize.Y)
	op.GeoM.Scale(b.Rect.Width(), b.Rect.Height())
	op.GeoM.Translate(b.Rect.Min.X, b.Rect.Min.Y)
	target.DrawImage(b.Image, &op)
}

func imageSizeOf(image *ebiten.Image) Vec {
	return Vec{
		X: float64(image.Bounds().Dx()),
		Y: float64(image.Bounds().Dy()),
	}
}
