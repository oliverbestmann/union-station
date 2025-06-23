package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	. "github.com/quasilyte/gmath"
)

var touchIds []ebiten.TouchID

type CursorState struct {
	Position     Vec
	JustPressed  bool
	JustReleased bool
}

func Cursor() CursorState {
	// re-use touchId buffer
	touchIds = ebiten.AppendTouchIDs(touchIds[:0])
	for _, touchId := range touchIds {
		// check if this one was just pressed or released
		pressed := inpututil.TouchPressDuration(touchId) == 0
		released := inpututil.IsTouchJustReleased(touchId)

		// calculate position
		pos := intToVec(ebiten.TouchPosition(touchId))

		return CursorState{
			Position:     pos,
			JustPressed:  pressed,
			JustReleased: released,
		}
	}

	// check if mouse was just pressed or released
	pressed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	released := inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)

	// get mouse position
	pos := intToVec(ebiten.CursorPosition())

	return CursorState{
		Position:     pos,
		JustPressed:  pressed,
		JustReleased: released,
	}
}

func intToVec(x, y int) Vec {
	return Vec{X: float64(x), Y: float64(y)}
}
