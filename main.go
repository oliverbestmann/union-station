package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"log"
	"time"
)

var TimeOrigin = time.Now()

func main() {
	// defer ProfileCPU()()

	const windowScale = 2
	const renderScale = 2

	screenWidth, screenHeight := 800, 480

	game := &Game{
		seed:         14,
		screenWidth:  screenWidth * renderScale,
		screenHeight: screenHeight * renderScale,
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
