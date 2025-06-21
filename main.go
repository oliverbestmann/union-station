package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/oliverbestmann/union-station/assets"
	"log"
	"time"
)

var TimeOrigin = time.Now()

func main() {
	// defer ProfileCPU()()

	const windowScale = 2
	const renderScale = 2

	screenWidth, screenHeight := 800, 480

	game := &Loader[Audio]{
		// load audio task
		Promise: AsyncTask(func(yield func(string)) Audio {
			var idle IdleSuspend

			yield("music")
			music := DecodeAudio(&idle, assets.Music())

			yield("button-press")
			buttonPress := DecodeAudio(&idle, assets.ButtonPress())

			yield("button-hover")
			buttonHover := DecodeAudio(&idle, assets.ButtonHover())

			return Audio{
				Music:       music,
				ButtonPress: buttonPress,
				ButtonHover: buttonHover,
			}
		}),

		Next: func(audio Audio) ebiten.Game {
			return &Game{
				audio: audio,
				seed:  17,

				screenWidth:  screenWidth * renderScale,
				screenHeight: screenHeight * renderScale,
			}
		},
	}

	// Specify the window size as you like. Here, a doubled size is specified.
	ebiten.SetWindowSize(screenWidth*windowScale, screenHeight*windowScale)
	ebiten.SetWindowTitle("Union Station")
	ebiten.SetVsyncEnabled(true)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// Call ebiten.RunGame to start your game loop.
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
