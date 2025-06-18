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
	screenWidth, screenHeight := 800*windowScale, 480*windowScale

	game := &Game{
		seed:         14,
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
	}

	// Specify the window size as you like. Here, a doubled size is specified.
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Union Station")
	ebiten.SetVsyncEnabled(true)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// Call ebiten.RunGame to start your game loop.
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
