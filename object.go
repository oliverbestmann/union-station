package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	. "github.com/quasilyte/gmath"
)

type Drawable interface {
	Draw(target *ebiten.Image, toScreen ebiten.GeoM)
}

type Clickable interface {
	Click(toScreen ebiten.GeoM, screenClick Vec) bool
}

type InterceptClicks struct {
	Callable   func()
	ScreenRect Rect
}

func (i *InterceptClicks) Click(toScreen ebiten.GeoM, screenClick Vec) bool {
	if !i.ScreenRect.Contains(screenClick) {
		return false
	}

	i.Callable()
	return true
}
