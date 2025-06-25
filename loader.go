package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"runtime"
)

type LoadingScreen interface {
	Update(ready bool, progress *string) (startGame bool)
	Draw(screen *ebiten.Image)
}

type Loader[T any] struct {
	Next          func(T) ebiten.Game
	Promise       Promise[T, string]
	LoadingScreen LoadingScreen

	loaded       bool
	initializing bool
	playing      bool
	game         ebiten.Game

	ScreenWidth, ScreenHeight int
}

func (l *Loader[T]) Update() error {
	UpdateCursorState()

	switch {
	case l.playing:
		return l.game.Update()

	default:
		if result := l.Promise.GetOnce(); result != nil {
			l.game = l.Next(*result)

			runtime.GC()

			l.loaded = true
		}

		// update the loading screen
		done := l.LoadingScreen.Update(l.loaded, l.Promise.Status())

		if l.loaded && done {
			// game has loaded and the loading screen reported that it is done,
			// switch into playing state
			l.playing = true

			// layout is guaranteed to be called once before
			// the first call to Update
			l.game.Layout(l.ScreenWidth, l.ScreenHeight)

			// and initialize the actual game
			return l.game.Update()
		}

	}

	return nil
}

func (l *Loader[T]) Draw(screen *ebiten.Image) {
	switch {
	case l.playing:
		l.game.Draw(screen)

	default:
		l.LoadingScreen.Draw(screen)
	}
}

func (l *Loader[T]) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	if l.playing {
		return l.game.Layout(outsideWidth, outsideHeight)
	}

	return l.ScreenWidth, l.ScreenHeight
}
