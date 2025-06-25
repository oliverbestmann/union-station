package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/oliverbestmann/union-station/assets"
	"log"
	"time"
)

var TimeOrigin = time.Now()

func main() {
	const windowScale = 2
	const renderScale = 2

	screenWidth, screenHeight := 800, 480

	// ensure we have an audio context
	AudioContext()

	game := &Loader[Audio]{
		ScreenWidth:  screenWidth * renderScale,
		ScreenHeight: screenHeight * renderScale,

		// load audio task
		Promise: AsyncTask(func(yield func(string)) Audio {
			yield("Loading audio data")
			buttonPress := assets.ButtonPress()
			buttonHover := assets.ButtonHover()

			return Audio{
				Songs:       assets.Songs(),
				ButtonPress: Samples(buttonPress),
				ButtonHover: Samples(buttonHover),
			}
		}),

		LoadingScreen: &TheLoadingScreen{
			now: TimeOrigin,
		},

		Next: func(audio Audio) ebiten.Game {
			return &Game{
				audio: audio,

				screenWidth:  screenWidth * renderScale,
				screenHeight: screenHeight * renderScale,
			}
		},
	}

	// Specify the window size as you like. Here, a doubled size is specified.
	ebiten.SetWindowSize(screenWidth*windowScale, screenHeight*windowScale)
	ebiten.SetWindowTitle("Union Station")
	ebiten.SetVsyncEnabled(true)
	ebiten.SetTPS(ebiten.SyncWithFPS)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// Call ebiten.RunGame to start your game loop.
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

func progressYield(yield func(string), desc string) func(float64) {
	return func(f float64) {
		yield(fmt.Sprintf("%s: %d%%", desc, int(f*100)))
	}
}
