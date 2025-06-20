package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
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
	screenSize := imageSizeOf(screen)
	x := screenSize.X/2 - 64
	y := screenSize.Y/2 - float64(Font.Metrics().Ascent.Round()/2)

	var op ebiten.DrawImageOptions
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(x, y)

	text.DrawWithOptions(screen, t, Font, &op)
}

func (l *Loader[T]) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	l.screenWidth = outsideWidth
	l.screenHeight = outsideHeight

	if l.playing {
		return l.game.Layout(outsideWidth, outsideHeight)
	}

	return outsideWidth, outsideHeight
}
