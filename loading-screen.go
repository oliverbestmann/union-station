package main

import (
	"github.com/fogleman/ease"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/oliverbestmann/union-station/tween"
	. "github.com/quasilyte/gmath"
	"image/color"
	"strings"
	"time"
	"unicode"
)

type TheLoadingScreen struct {
	progressText string
	now          time.Time

	visibleWorldCount float64
	howtoText         string
	howtoTextComplete bool

	btnContinue *Button
	tweens      tween.Tweens
}

func (l *TheLoadingScreen) Update(ready bool, progress *string) (startGame bool) {
	dt := time.Since(l.now)
	dtSecs := dt.Seconds()
	l.now = l.now.Add(dt)

	if ready {
		if l.btnContinue == nil {
			l.btnContinue = NewButton("Start the game", StartGameButtonColors)

			l.tweens.Add(&tween.Simple{
				Duration: 250 * time.Millisecond,
				Ease:     ease.OutCubic,
				Target:   tween.LerpValue(&l.btnContinue.Alpha, 0, 1),
			})
		}

		cursor := Cursor()

		l.btnContinue.Hover(cursor)

		if l.btnContinue.IsClicked(cursor) {
			startGame = true
		}
	}

	l.tweens.Update(dt)

	switch {
	case ready:
		l.progressText = ""

	case progress != nil:
		l.progressText = *progress
	}

	l.visibleWorldCount += dtSecs * 10
	l.howtoText = trimWords(howto, int(l.visibleWorldCount))
	l.howtoTextComplete = len(l.howtoText) == len(howto)

	return
}

func (l *TheLoadingScreen) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)

	screenSize := imageSizeOf(screen)

	// measure the full text, even if we just render a part of it
	textSize := MeasureText(Font24, howto)

	// add some space for button + spacing
	contentSize := textSize.Add(Vec{Y: 32 + 48})

	// draw the text
	posText := screenSize.Mulf(0.5).Sub(contentSize.Mulf(0.5))
	DrawText(screen, l.howtoText, Font24, posText, LightTextColor, text.AlignStart, text.AlignStart)

	if l.btnContinue != nil {
		// draw the button below the text, 32px spacing
		pos := Vec{X: screenSize.X/2 - 128, Y: posText.Y + textSize.Y + 32}
		l.btnContinue.Position = pos
		l.btnContinue.Size = Vec{X: 256, Y: 48}
		l.btnContinue.Draw(screen)

	} else if l.howtoTextComplete {
		pos := Vec{X: screenSize.X / 2, Y: posText.X + textSize.Y + 32}
		DrawTextCenter(screen, "The train is running late, please have a little patience...", Font16, pos, LightTextColor)
	}

	if l.progressText != "" {
		// progress text
		pos := Vec{X: screenSize.X * 0.5, Y: screenSize.Y - 64}
		DrawTextCenter(screen, l.progressText, Font16, pos, LightTextColor)
	}
}

func trimWords(text string, wordCount int) string {
	if wordCount <= 0 {
		return ""
	}

	inSpace := unicode.IsSpace(rune(text[0]))

	for idx, ch := range text {
		isSpace := unicode.IsSpace(ch)

		switch {
		case isSpace == inSpace:
			// no change, just look at the next char
			continue

		case isSpace && !inSpace:
			// we just entered space
			inSpace = true
			wordCount -= 1

			// if we've reached the number of words to render,
			// we've finished
			if wordCount == 0 {
				return text[:idx-1]
			}

		case !isSpace && inSpace:
			// we've just left space.
			inSpace = false

		}
	}

	return text
}

var howto = strings.TrimSpace(`
Welcome to Union Station - a strategic railway builder
set in the rolling hills of the British countryside.

Your mission? Unite distant towns by constructing efficient
train routes on a limited budget. Plan your network carefully,
balancing cost with connectivity. Activate routes early
to boost your public reputation and climb the global leaderboard.

Every decision counts.
Will you be the one to unite the nation, one rail at a time?
`)
