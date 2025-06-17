package main

import (
	"fmt"
	"github.com/hajimehoshi/bitmapfont"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	. "github.com/quasilyte/gmath"
	"log"
	"math"
	"slices"
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
	villagesAsync Promise[VillageCalculation]

	selectedVillage *Village

	tr    ebiten.GeoM
	trInv ebiten.GeoM

	gen StreetGenerator
}

func (g *Game) Init() {
	g.startTime = time.Now()
	g.lastUpdate = time.Now()

	// base size, used for scaling
	worldWidth := 32000.0

	scale := float64(g.screenWidth) / worldWidth

	g.worldScale = scale
	g.tr.Scale(scale, scale)

	// create an inverse of the transform to paint the noise data based
	// on the pixel position
	g.trInv = g.tr
	g.trInv.Invert()

	// calculate world size based on transformed screen size
	x0, y0 := g.trInv.Apply(0, 0)
	x1, y1 := g.trInv.Apply(float64(g.screenWidth), float64(g.screenHeight))
	g.worldSize = Rect{Min: Vec{X: x0, Y: y0}, Max: Vec{X: x1, Y: y1}}

	// discard streets outside of the visible world
	g.gen = NewStreetGenerator(g.worldSize, 1)

	// generate an image from noise
	g.noise = noiseToImage(g.gen.noise, g.screenWidth, g.screenHeight, g.trInv)

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
	now := time.Now()
	dt := now.Sub(g.lastUpdate).Seconds()
	g.lastUpdate = now

	_ = dt

	var newSegmentCount int

	for g.gen.More() && time.Since(now) < 12*time.Millisecond {
		if segment := g.gen.Next(); segment != nil {
			// draw the segment to the street image
			segment.Draw(g.streets, g.tr)
			newSegmentCount += 1
		}
	}

	// check if we've finished remaining generation
	if newSegmentCount > 0 && !g.gen.More() {
		g.streetGenerationEndTime = time.Now()

		// asynchronously calculate the villages
		g.villagesAsync = AsyncTask(func() VillageCalculation {
			image := ebiten.NewImage(g.screenWidth, g.screenHeight)
			image.Fill(rgbaOf(0xdbcfb1ff))

			var idle IdleSuspend

			// paint the streets again
			for idx, segment := range g.gen.segments {
				segment.Draw(image, g.tr)

				if idx%1_000 == 0 {
					// suspend on wasm if needed
					idle.MaybeSuspend()
				}
			}

			// find villages
			villages := VillagesOf(g.gen.rng, g.gen.grid, g.gen.Segments())

			// find one or more stations per village
			stations := GenerateStations(g.gen.rng, villages)

			stations = slices.DeleteFunc(stations, func(station *Station) bool {
				// remove stations that are near the border
				clip := Rect{
					Min: g.worldSize.Min.Add(g.worldSize.Size().Mulf(0.1)),
					Max: g.worldSize.Max.Sub(g.worldSize.Size().Mulf(0.1)),
				}

				return !clip.Contains(station.Position)
			})

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

	// check if any village should be highlighted
	if villages := g.villagesAsync.Get(); villages != nil {
		worldCursor := CursorPosition(g.trInv)

		// reset selected villages
		g.selectedVillage = nil

		for _, village := range villages.Villages {
			if !village.BBox.Contains(worldCursor) {
				continue
			}

			if PointInConvexHull(village.Hull, worldCursor) {
				g.selectedVillage = village
			}
		}
	}

	return nil
}

// Draw draws the game screen.
// Draw is called every frame (typically 1/60[s] for 60Hz display).
func (g *Game) Draw(screen *ebiten.Image) {
	tr := g.tr

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
			loc := TransformVec(g.tr, station.Position).AsVec32()

			color := rgbaOf(0x6d838eff)
			vector.DrawFilledCircle(screen, loc.X, loc.Y, 10, color, true)

			color = rgbaOf(0x839ca9ff)
			vector.DrawFilledCircle(screen, loc.X, loc.Y, 8, color, true)
		}

		if g.selectedVillage != nil {
			MarkVillage(screen, g.tr, g.selectedVillage)
		}
	}

	// if we're busy, paint a busy indicator
	if g.villagesAsync.Waiting() {
		offsetY := 4 * math.Sin(time.Since(TimeOrigin).Seconds()*5)
		bounds := MeasureText(bitmapfont.Gothic12r, "please wait...")

		var op ebiten.DrawImageOptions
		op.GeoM.Translate(-0.5*bounds.X, -0.5*bounds.Y)
		op.GeoM.Scale(3.0, 3.0)
		op.GeoM.Translate(float64(g.screenWidth)/2, float64(g.screenHeight)/2+offsetY)
		op.ColorScale.ScaleWithColor(rgbaOf(0xa05e5eff))
		text.DrawWithOptions(screen, "please wait...", bitmapfont.Gothic12r, &op)
	}

	var op ebiten.DrawImageOptions
	op.GeoM.Translate(32, 32)
	op.ColorScale.ScaleWithColor(DebugColor)

	t := fmt.Sprintf("%1.1f fps", ebiten.ActualFPS())
	text.DrawWithOptions(screen, t, bitmapfont.Gothic12r, &op)
	op.GeoM.Translate(0, 16)

	t = fmt.Sprintf("Street Segments: %d", len(g.gen.segments))
	text.DrawWithOptions(screen, t, bitmapfont.Gothic12r, &op)
	op.GeoM.Translate(0, 16)

	if !g.streetGenerationEndTime.IsZero() {
		t = fmt.Sprintf("Street generation took %s", g.streetGenerationEndTime.Sub(g.startTime))
		text.DrawWithOptions(screen, t, bitmapfont.Gothic12r, &op)
		op.GeoM.Translate(0, 16)
	}

	if g.villagesAsync.Waiting() {
		t = "Calculating villages"
		text.DrawWithOptions(screen, t, bitmapfont.Gothic12r, &op)
		op.GeoM.Translate(0, 16)
	}

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

	game.Init()

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
