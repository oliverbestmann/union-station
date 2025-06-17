package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"image/color"
)

var whiteImage *ebiten.Image

func init() {
	img := ebiten.NewImage(3, 3)
	img.Fill(color.White)
	whiteImage = img.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
}
