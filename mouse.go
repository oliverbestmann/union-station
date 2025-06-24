package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	. "github.com/quasilyte/gmath"
	"math"
)

var touchIds []ebiten.TouchID

type CursorState struct {
	Position     Vec
	JustPressed  bool
	JustReleased bool
}

var activeTouchId ebiten.TouchID = math.MinInt
var activeTouchPosition Vec

func Cursor() CursorState {
	if activeTouchId >= 0 {
		released := inpututil.IsTouchJustReleased(activeTouchId)
		if released {
			activeTouchId = math.MinInt
		} else {
			activeTouchPosition = intToVec(ebiten.TouchPosition(activeTouchId))
		}

		return CursorState{
			Position:     activeTouchPosition,
			JustReleased: released,
		}
	}

	touchIds = inpututil.AppendJustPressedTouchIDs(touchIds[:0])
	for _, touchId := range touchIds {
		activeTouchId = touchId
		pos := intToVec(ebiten.TouchPosition(touchId))

		return CursorState{
			Position:    pos,
			JustPressed: true,
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
