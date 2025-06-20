package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type Loader[T any] struct {
	Next    func(T) ebiten.Game
	Promise Promise[T, string]

	loaded      bool
	playing     bool
	initialized bool
	game        ebiten.Game

	screenWidth, screenHeight int
}

func (l *Loader[T]) Update() error {
	switch {
	case l.playing:
		l.initialized = true
		return l.game.Update()

	case l.loaded:
		if AudioContext().IsReady() {
			l.playing = true
		}

		if _, clicked := Clicked(); clicked {
			l.playing = true
		}

	default:
		if result := l.Promise.Get(); result != nil {
			l.game = l.Next(*result)
			l.game.Layout(l.screenWidth, l.screenHeight)

			l.loaded = true

			if AudioContext().IsReady() {
				// directly jump to playing as fast as possible
				l.playing = true
			}
		}
	}

	return nil
}

func (l *Loader[T]) Draw(screen *ebiten.Image) {
	switch {
	case l.initialized:
		l.game.Draw(screen)

	case l.loaded:
		l.drawText(screen, "click anywhere to continue")

	default:
		desc := "loading..."
		if status := l.Promise.Status(); status != nil {
			desc = "loading: " + *status + "..."
		}

		l.drawText(screen, desc)
	}
}

func (l *Loader[T]) drawText(screen *ebiten.Image, t string) {
	center := imageSizeOf(screen).Mulf(0.5)
	DrawTextCenter(screen, t, Font16, center, BackgroundColor)
}

func (l *Loader[T]) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	l.screenWidth = outsideWidth
	l.screenHeight = outsideHeight

	if l.playing {
		return l.game.Layout(outsideWidth, outsideHeight)
	}

	return outsideWidth, outsideHeight
}
