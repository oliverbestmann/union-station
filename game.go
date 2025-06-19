package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	. "github.com/quasilyte/gmath"
	"math"
	"math/rand/v2"
	"slices"
	"time"
)

type VillageCalculation struct {
	EndTime  time.Time
	Image    *ebiten.Image
	Villages []*Village
	Stations []*Station
}

// Game implements ebiten.Game interface.
type Game struct {
	initialized bool

	screenWidth  int
	screenHeight int
	worldScale   float64

	debug bool

	startTime               time.Time
	streetGenerationEndTime time.Time

	worldSize Rect

	lastUpdate time.Time

	noise         *ebiten.Image
	streets       *ebiten.Image
	villagesAsync Promise[VillageCalculation, string]

	hoveredStation     *Station
	selectedStationOne *Station
	selectedStationTwo *Station

	toScreen ebiten.GeoM
	toWorld  ebiten.GeoM

	clicked      bool
	cursorWorld  Vec
	cursorScreen Vec

	rng  *rand.Rand
	gen  StreetGenerator
	seed uint64

	btnAcceptConnection *Button
	btnDesignConnection *Button
	btnCancelConnection *Button

	stationGraph StationGraph
	audio        Audio
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	_ = outsideWidth
	_ = outsideHeight

	return g.screenWidth, g.screenHeight
}

func (g *Game) Reset(seed uint64) {
	*g = Game{
		initialized:  true,
		seed:         seed,
		audio:        g.audio,
		screenWidth:  g.screenWidth,
		screenHeight: g.screenHeight,
		debug:        true,
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

func (g *Game) Update() error {
	// initialize the game if needed
	if !g.initialized {
		g.Reset(g.seed)

		// start music audio playback only the first time
		g.audio.PlayMusic()
	}

	// step to the next seed
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.Reset(g.seed + 1)
	}

	// calculate delta time for animations
	now := time.Now()
	dt := now.Sub(g.lastUpdate).Seconds()
	g.lastUpdate = now

	_ = dt

	var newSegmentCount int

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
		g.villagesAsync = AsyncTask(g.computeVillages)
	}

	if res := g.villagesAsync.Get(); res != nil {
		g.streets = res.Image
	}

	// get click information
	g.cursorScreen, g.clicked = Clicked()
	if !g.clicked {
		g.cursorScreen = CursorPosition()
	}

	// derive screen cursor position
	g.cursorWorld = TransformVec(g.toWorld, g.cursorScreen)

	// play sound if click was done
	if g.clicked {
		g.audio.Play(g.audio.ButtonPress)
	}

	// now process input
	g.Input()

	return nil
}

func (g *Game) Input() {
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		g.debug = !g.debug
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		g.audio.ToggleMute()
	}

	var inputIntercepted bool

	//goland:noinspection GoDfaConstantCondition
	inputIntercepted = g.btnAcceptConnection.Hover(g.cursorScreen) || inputIntercepted
	inputIntercepted = g.btnDesignConnection.Hover(g.cursorScreen) || inputIntercepted
	inputIntercepted = g.btnCancelConnection.Hover(g.cursorScreen) || inputIntercepted

	// check button inputs
	if g.btnAcceptConnection.IsClicked(g.cursorScreen, g.clicked) {
		// accept the station
		g.stationGraph.Insert(g.selectedStationOne, g.selectedStationTwo)
		g.resetInput()
	}

	if g.btnDesignConnection.IsClicked(g.cursorScreen, g.clicked) {
		// hide all buttons
		g.resetInput()
	}

	if g.btnCancelConnection.IsClicked(g.cursorScreen, g.clicked) {
		// hide all buttons
		g.resetInput()
	}

	var currentStation *Station

	if !inputIntercepted {
		if result := g.villagesAsync.Get(); result != nil {
			// get the station that is nearest to the mouse
			station := MaxOf(slices.Values(result.Stations), func(station *Station) float64 {
				return -g.cursorWorld.DistanceSquaredTo(station.Position)
			})

			// calculate distance to station in screen space
			stationScreen := TransformVec(g.toScreen, station.Position)
			distanceToStation := g.cursorScreen.DistanceTo(stationScreen)
			isNear := distanceToStation < 32.0
			isNotSelected := station != g.selectedStationOne && station != g.selectedStationTwo

			if isNotSelected && (isNear || station.Village.Contains(g.cursorWorld)) {
				currentStation = station
			}
		}

		noStationSelected := currentStation == nil

		// if the hovered station is already connected to the first station, we do not allow to hover or click it
		if g.selectedStationOne != nil && g.stationGraph.Has(g.selectedStationOne, currentStation) {
			currentStation = nil
		}

		if g.clicked {
			switch {
			case noStationSelected:
				g.resetInput()

			case g.selectedStationOne == nil:
				// select the clicked village (or nil, if none was clicked)
				g.selectedStationOne = currentStation

			case g.selectedStationOne != nil && currentStation != nil && currentStation != g.selectedStationOne:
				// select the clicked village (or nil, if none was clicked)
				g.selectedStationTwo = currentStation

				// show the buttons near the click location
				buttonVec := g.cursorScreen.Add(vecSplat(-16))
				g.btnAcceptConnection = NewButton("Build", buttonVec)
				g.btnDesignConnection = NewButton("Plan", buttonVec.Add(Vec{Y: 32 + 8}))
				g.btnCancelConnection = NewButton("Cancel", buttonVec.Add(Vec{Y: 2 * (32 + 8)}))
			}
		}
	}

	g.hoveredStation = currentStation

	if currentStation == g.selectedStationOne || currentStation == g.selectedStationTwo {
		// actually, do not hover one of the selected villages
		g.hoveredStation = nil
	}
}

func (g *Game) resetInput() {
	g.selectedStationOne = nil
	g.selectedStationTwo = nil
	g.btnAcceptConnection = nil
	g.btnDesignConnection = nil
	g.btnCancelConnection = nil
}

func (g *Game) computeVillages(yield func(string)) VillageCalculation {
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
		EndTime:  time.Now(),
		Image:    image,
		Villages: villages,
		Stations: stations,
	}
}

// Draw draws the game screen.
// Draw is called every frame (typically 1/60[s] for 60Hz display).
func (g *Game) Draw(screen *ebiten.Image) {
	screen.DrawImage(g.streets, nil)

	if ebiten.IsKeyPressed(ebiten.KeyN) {
		if g.noise == nil {
			// generate an image from noise
			g.noise = noiseToImage(g.gen.Noise(), g.screenWidth, g.screenHeight, g.toWorld)
		}

		screen.DrawImage(g.noise, nil)
	}

	// check if any village should be highlighted
	if result := g.villagesAsync.Get(); result != nil {
		g.drawVillageCalculation(screen, result)
	}

	g.btnAcceptConnection.Draw(screen)
	g.btnDesignConnection.Draw(screen)
	g.btnCancelConnection.Draw(screen)

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

	if g.debug {
		g.DrawDebugLines(screen)
	}
}

func (g *Game) drawVillageCalculation(screen *ebiten.Image, result *VillageCalculation) {
	// walk through the edges we've added and paint them
	for _, edge := range g.stationGraph.Edges() {
		DrawStationConnection(screen, g.toScreen, edge.One, edge.Two, StationColorConstructed)
	}

	// paint the edges of the currently planed route
	if g.selectedStationOne != nil && g.selectedStationTwo != nil {
		// we have two selected villages, draw a dummy connection between them
		DrawStationConnection(screen, g.toScreen, g.selectedStationOne, g.selectedStationTwo, StationColorSelected)
	}

	// paint the stations
	for _, station := range result.Stations {
		loc := TransformVec(g.toScreen, station.Position).AsVec32()

		stationColor := g.stationColorOf(station)

		vector.DrawFilledCircle(screen, loc.X, loc.Y, 10, stationColor.Stroke, true)
		vector.DrawFilledCircle(screen, loc.X, loc.Y, 8, stationColor.Fill, true)
	}

	if station := g.hoveredStation; station != nil {
		DrawVillageBounds(screen, station.Village, DrawVillageBoundsOptions{
			ToScreen:  g.toScreen,
			FillColor: rgbaOf(0x83838320),
		})

		DrawVillageTooltip(screen, g.cursorScreen.Add(Vec{X: 16, Y: 16}), station.Village)
	}

	if station := g.selectedStationOne; station != nil {
		DrawVillageBounds(screen, station.Village, DrawVillageBoundsOptions{
			ToScreen:    g.toScreen,
			FillColor:   rgbaOf(0xb089ab50),
			StrokeColor: rgbaOf(0xb089abff),
			StrokeWidth: 2,
		})
	}

	if station := g.selectedStationTwo; station != nil {
		DrawVillageBounds(screen, station.Village, DrawVillageBoundsOptions{
			ToScreen:    g.toScreen,
			FillColor:   rgbaOf(0xb089ab50),
			StrokeColor: rgbaOf(0xb089abff),
			StrokeWidth: 2,
		})
	}

}

func (g *Game) stationColorOf(station *Station) StationColor {
	stationColor := StationColorIdle

	// if the circle is hovered, select a different color palette
	switch {
	case g.selectedStationOne == station, g.selectedStationTwo == station:
		stationColor = StationColorSelected

	case g.hoveredStation == station:
		stationColor = StationColorHover

	case len(g.stationGraph.EdgesOf(station)) > 0:
		stationColor = StationColorConstructed
	}

	return stationColor
}

func (g *Game) DrawDebugLines(screen *ebiten.Image) {
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

	if result := g.villagesAsync.Get(); result != nil {
		t = fmt.Sprintf("City generation took %s", result.EndTime.Sub(g.startTime))
		text.DrawWithOptions(screen, t, Font, &op)
		op.GeoM.Translate(0, 16)
	}

	if progress := g.villagesAsync.Status(); progress != nil {
		t = *progress + "..."
		text.DrawWithOptions(screen, t, Font, &op)
		op.GeoM.Translate(0, 16)
	}
}

func (g *Game) buttons() []*Button {
	buttons := [2]*Button{
		g.btnAcceptConnection,
		g.btnCancelConnection,
	}

	return slices.DeleteFunc(buttons[:], func(button *Button) bool {
		return button == nil
	})
}
