package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	. "github.com/quasilyte/gmath"
)

var touchIds []ebiten.TouchID

func Clicked() (Vec, bool) {
	// re-use touchId buffer
	touchIds = inpututil.AppendJustPressedTouchIDs(touchIds[:0])
	for _, touchId := range touchIds {
		touchX, touchY := ebiten.TouchPosition(touchId)
		return intToVec(touchX, touchY), true
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mouseX, mouseY := ebiten.CursorPosition()
		return intToVec(mouseX, mouseY), true
	}

	return Vec{}, false
}

func CursorPosition() Vec {
	touchIds = ebiten.AppendTouchIDs(touchIds[:0])
	for _, touchId := range touchIds {
		touchX, touchY := ebiten.TouchPosition(touchId)
		return intToVec(touchX, touchY)
	}

	mouseX, mouseY := ebiten.CursorPosition()
	return intToVec(mouseX, mouseY)
}

func intToVec(x, y int) Vec {
	return Vec{X: float64(x), Y: float64(y)}
}
