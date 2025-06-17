package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	. "github.com/quasilyte/gmath"
)

var touchIds []ebiten.TouchID

func Clicked(toWorld ebiten.GeoM) (Vec, bool) {
	// re-use touchId buffer
	touchIds = inpututil.AppendJustPressedTouchIDs(touchIds[:0])
	for _, touchId := range touchIds {
		touchX, touchY := ebiten.TouchPosition(touchId)
		return transformCursor(toWorld, touchX, touchY), true
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mouseX, mouseY := ebiten.CursorPosition()
		return transformCursor(toWorld, mouseX, mouseY), true
	}

	return Vec{}, false
}

func CursorPosition(toWorld ebiten.GeoM) Vec {
	touchIds = ebiten.AppendTouchIDs(touchIds[:0])
	for _, touchId := range touchIds {
		touchX, touchY := ebiten.TouchPosition(touchId)
		return transformCursor(toWorld, touchX, touchY)
	}

	mouseX, mouseY := ebiten.CursorPosition()
	return transformCursor(toWorld, mouseX, mouseY)
}

func transformCursor(tr ebiten.GeoM, x, y int) Vec {
	wx, wy := tr.Apply(float64(x), float64(y))
	return Vec{X: wx, Y: wy}
}
