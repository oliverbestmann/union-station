package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	. "github.com/quasilyte/gmath"
	"log"
	"math"
	"math/rand/v2"
	"time"
)

const windowScale = 2

var TimeOrigin = time.Now()

type Drawable interface {
	Draw(target *ebiten.Image, tr ebiten.GeoM)
}

type VillageCalculation struct {
	Image    *ebiten.Image
	Villages []*Village
	Stations []*Station
}

// Game implements ebiten.Game interface.
type Game struct {
	screenWidth  int
	screenHeight int
	worldScale   float64

	startTime               time.Time
	streetGenerationEndTime time.Time

	worldSize Rect

	lastUpdate time.Time
	drawables  []Drawable

	noise         *ebiten.Image
	streets       *ebiten.Image
	villagesAsync Promise[VillageCalculation, string]

	hoveredVillage     *Village
	selectedVillageOne *Village
	selectedVillageTwo *Village

	toScreen ebiten.GeoM
	toWorld  ebiten.GeoM

	cursorWorld  Vec
	cursorScreen Vec

	rng  *rand.Rand
	gen  StreetGenerator
	seed uint64
}

func (g *Game) Reset(seed uint64) {
	*g = Game{
		seed:         seed,
		screenWidth:  g.screenWidth,
		screenHeight: g.screenHeight,
	}

	g.startTime = time.Now()
	g.lastUpdate = time.Now()

	// base size, used for scaling
	worldWidth := 32000.0

	scale := float64(g.screenWidth) / worldWidth

	g.worldScale = scale
	g.toScreen.Scale(scale, scale)

	// create an inverse of the transform to transform from screen coordinates
	// to world coordinates
	g.toWorld = g.toScreen
	g.toWorld.Invert()

	// calculate world size based on transformed screen size
	x0, y0 := g.toWorld.Apply(0, 0)
	x1, y1 := g.toWorld.Apply(float64(g.screenWidth), float64(g.screenHeight))
	g.worldSize = Rect{Min: Vec{X: x0, Y: y0}, Max: Vec{X: x1, Y: y1}}

	g.rng = RandWithSeed(seed)

	// discard streets outside of the visible world
	g.gen = NewStreetGenerator(g.rng, g.worldSize)

	// generate an image from noise
	g.noise = noiseToImage(g.gen.Noise(), g.screenWidth, g.screenHeight, g.toWorld)

	// create an empty image for the streets
	g.streets = ebiten.NewImage(g.screenWidth, g.screenHeight)
	g.streets.Fill(rgbaOf(0xdbcfb1ff))

	// enqueue a starting point for the street generator
	g.gen.Push(PendingSegment{
		Point: g.worldSize.Center(),
		Angle: 0,
	})

	g.gen.Push(PendingSegment{
		Point: g.worldSize.Center(),
		Angle: DegToRad(180),
	})
}

// Update proceeds the game state.
// Update is called every tick (1/60 [s] by default).
func (g *Game) Update() error {
	if g.noise == nil {
		// initialize the game
		g.Reset(7)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.Reset(g.seed + 1)
	}

	now := time.Now()
	dt := now.Sub(g.lastUpdate).Seconds()
	g.lastUpdate = now

	_ = dt

	var newSegmentCount int

	g.cursorWorld = CursorPosition(g.toWorld)
	g.cursorScreen = CursorPosition(ebiten.GeoM{})

	for g.gen.More() && time.Since(now) < 12*time.Millisecond {
		if segment := g.gen.Next(); segment != nil {
			// draw the segment to the street image
			segment.Draw(g.streets, g.toScreen)
			newSegmentCount += 1
		}
	}

	// check if we've finished remaining generation
	if newSegmentCount > 0 && !g.gen.More() {
		g.streetGenerationEndTime = time.Now()

		// asynchronously calculate the villages
		g.villagesAsync = AsyncTask(func(yield func(string)) VillageCalculation {
			yield("Drawing streets")

			image := ebiten.NewImage(g.screenWidth, g.screenHeight)
			image.Fill(rgbaOf(0xdbcfb1ff))

			var idle IdleSuspend

			// paint the streets again
			for idx, segment := range g.gen.segments {
				segment.Draw(image, g.toScreen)

				if idx%1_000 == 0 {
					// suspend on wasm if needed
					idle.MaybeSuspend()
				}
			}

			// find villages
			yield("Collecting villages")
			villages := CollectVillages(g.rng, g.gen.grid, g.gen.Segments())

			yield("Calculate clip rectangle")

			// do not place anything near the edge of the screen
			clipThreshold := TransformScalar(g.toWorld, 128)
			clip := Rect{
				Min: g.worldSize.Min.Add(Vec{X: clipThreshold, Y: clipThreshold}),
				Max: g.worldSize.Max.Sub(Vec{X: clipThreshold, Y: clipThreshold}),
			}

			yield("Generating stations")
			stations := GenerateStations(g.rng, clip, villages)

			return VillageCalculation{
				Image:    image,
				Villages: villages,
				Stations: stations,
			}
		})
	}

	if res := g.villagesAsync.Get(); res != nil {
		g.streets = res.Image
	}

	worldClickedAt, clicked := Clicked(g.toWorld)

	var clickedVillage, hoveredVillage *Village

	if villages := g.villagesAsync.Get(); villages != nil {
		for _, village := range villages.Villages {
			if village.Contains(g.cursorWorld) {
				hoveredVillage = village
			}

			if clicked && village.Contains(worldClickedAt) {
				clickedVillage = village
			}
		}
	}

	if clicked {
		switch {
		case clickedVillage == nil:
			// clicked outside of any village
			g.selectedVillageOne = nil
			g.selectedVillageTwo = nil

		case g.selectedVillageOne == nil:
			// select the clicked village (or nil, if none was clicked)
			g.selectedVillageOne = clickedVillage

		case g.selectedVillageTwo == nil:
			// select the clicked village (or nil, if none was clicked)
			g.selectedVillageTwo = clickedVillage

		case g.selectedVillageOne != nil && g.selectedVillageTwo != nil:
			// re-select only the second village
			g.selectedVillageTwo = clickedVillage
		}
	}

	if hoveredVillage == g.selectedVillageOne || hoveredVillage == g.selectedVillageTwo {
		// do not hover one of the selected villages
		hoveredVillage = nil
	}

	g.hoveredVillage = hoveredVillage

	return nil
}

// Draw draws the game screen.
// Draw is called every frame (typically 1/60[s] for 60Hz display).
func (g *Game) Draw(screen *ebiten.Image) {
	tr := g.toScreen

	screen.DrawImage(g.streets, nil)

	if ebiten.IsKeyPressed(ebiten.KeyN) {
		screen.DrawImage(g.noise, nil)
	}

	for _, d := range g.drawables {
		d.Draw(screen, tr)
	}

	// check if any village should be highlighted
	if result := g.villagesAsync.Get(); result != nil {
		// paint the stations
		for _, station := range result.Stations {
			loc := TransformVec(g.toScreen, station.Position).AsVec32()

			color := rgbaOf(0x6d838eff)
			vector.DrawFilledCircle(screen, loc.X, loc.Y, 10, color, true)

			color = rgbaOf(0x839ca9ff)
			vector.DrawFilledCircle(screen, loc.X, loc.Y, 8, color, true)
		}

		if g.hoveredVillage != nil {
			DrawVillageBounds(screen, g.hoveredVillage, DrawVillageBoundsOptions{
				ToScreen:  g.toScreen,
				FillColor: rgbaOf(0x83838320),
			})

			DrawVillageTooltip(screen, g.cursorScreen.Add(Vec{X: 16, Y: 16}), g.hoveredVillage)
		}

		if g.selectedVillageOne != nil {
			DrawVillageBounds(screen, g.selectedVillageOne, DrawVillageBoundsOptions{
				ToScreen:    g.toScreen,
				FillColor:   rgbaOf(0xb089ab50),
				StrokeColor: rgbaOf(0xb089abff),
				StrokeWidth: 2,
			})
		}

		if g.selectedVillageTwo != nil {
			DrawVillageBounds(screen, g.selectedVillageTwo, DrawVillageBoundsOptions{
				ToScreen:    g.toScreen,
				FillColor:   rgbaOf(0xb089ab50),
				StrokeColor: rgbaOf(0xb089abff),
				StrokeWidth: 2,
			})
		}

		if g.selectedVillageOne != nil && g.selectedVillageTwo != nil {
			// we have two selected villages, draw a dummy connection between them
			DrawVillageConnection(screen, g.toScreen, result.Stations, g.selectedVillageOne, g.selectedVillageTwo)
		}
	}

	// if we're busy, paint a busy indicator
	if g.villagesAsync.Waiting() {
		offsetY := 4 * math.Sin(time.Since(TimeOrigin).Seconds()*5)
		bounds := MeasureText(Font, "please wait...")

		var op ebiten.DrawImageOptions
		op.GeoM.Translate(-0.5*bounds.X, -0.5*bounds.Y)
		op.GeoM.Scale(3.0, 3.0)
		op.GeoM.Translate(float64(g.screenWidth)/2, float64(g.screenHeight)/2+offsetY)
		op.ColorScale.ScaleWithColor(rgbaOf(0xa05e5eff))
		text.DrawWithOptions(screen, "please wait...", Font, &op)
	}

	var op ebiten.DrawImageOptions
	op.GeoM.Translate(32, 32)
	op.ColorScale.ScaleWithColor(DebugColor)

	t := fmt.Sprintf("%1.1f fps, seed %d", ebiten.ActualFPS(), g.seed)
	text.DrawWithOptions(screen, t, Font, &op)
	op.GeoM.Translate(0, 16)

	t = fmt.Sprintf("Street Segments: %d", len(g.gen.segments))
	text.DrawWithOptions(screen, t, Font, &op)
	op.GeoM.Translate(0, 16)

	if !g.streetGenerationEndTime.IsZero() {
		t = fmt.Sprintf("Street generation took %s", g.streetGenerationEndTime.Sub(g.startTime))
		text.DrawWithOptions(screen, t, Font, &op)
		op.GeoM.Translate(0, 16)
	}

	if progress := g.villagesAsync.Progress(); progress != nil {
		t = *progress + "..."
		text.DrawWithOptions(screen, t, Font, &op)
		op.GeoM.Translate(0, 16)
	}
}

func DrawVillageConnection(target *ebiten.Image, toScreen ebiten.GeoM, stations []*Station, one *Village, two *Village) {
	var stationOne, stationTwo *Station
	for _, station := range stations {
		if station.Village == one {
			stationOne = station
		}

		if station.Village == two {
			stationTwo = station
		}
	}

	if stationOne == nil || stationTwo == nil {
		return
	}

	// work in screen space
	start := TransformVec(toScreen, stationOne.Position).AsVec32()
	end := TransformVec(toScreen, stationTwo.Position).AsVec32()

	// calculate length & direction to lerp across the screen
	length := end.Sub(start).Len()
	direction := end.Sub(start).Normalized()

	var path vector.Path

	const segmentLen = 20

	for f := float32(0); f < length; f += segmentLen {
		a := start.Add(direction.Mulf(f))
		b := start.Add(direction.Mulf(min(f+segmentLen/2, length)))

		path.MoveTo(a.X, a.Y)
		path.LineTo(b.X, b.Y)
	}

	StrokePath(target, path, ebiten.GeoM{}, rgbaOf(0x8e6d89ff), &vector.StrokeOptions{
		Width: 4.0,
	})
}

// Layout takes the outside size (e.g., the window size) and returns the (logical) screen size.
// If you don't have to adjust the screen size with the outside size, just return a fixed size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.screenWidth, g.screenHeight
}

func main() {
	defer ProfileCPU()()

	screenWidth, screenHeight := 800*windowScale, 480*windowScale

	game := &Game{
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
