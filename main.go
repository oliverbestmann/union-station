package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/oliverbestmann/union-station/assets"
	"log"
	"time"
)

var TimeOrigin = time.Now()

func main() {
	defer ProfileCPU()()

	const windowScale = 2
	const renderScale = 2

	screenWidth, screenHeight := 800, 480

	game := &Loader[Audio]{
		ScreenWidth:  screenWidth * renderScale,
		ScreenHeight: screenHeight * renderScale,

		// load audio task
		Promise: AsyncTask(func(yield func(string)) Audio {
			var idle IdleSuspend

			yield("downloading music")
			songs := []*vorbis.Stream{
				assets.Song1(),
				assets.Song2(),
			}

			var decodedSongs []Samples
			for idx := range songs {
				yield(fmt.Sprintf("decoding music %d of %d", idx+1, len(songs)))

				song := DecodeAudio(&idle, songs[idx])
				decodedSongs = append(decodedSongs, song)
			}

			yield("button-press")
			buttonPress := DecodeAudio(&idle, assets.ButtonPress())

			yield("button-hover")
			buttonHover := DecodeAudio(&idle, assets.ButtonHover())

			return Audio{
				Songs:       decodedSongs,
				ButtonPress: buttonPress,
				ButtonHover: buttonHover,
			}
		}),

		LoadingScreen: &TheLoadingScreen{
			now: TimeOrigin,
		},

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
	ebiten.SetTPS(ebiten.SyncWithFPS)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// Call ebiten.RunGame to start your game loop.
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
