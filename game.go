package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/oliverbestmann/union-station/assets"
	. "github.com/quasilyte/gmath"
	"image/color"
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
	Render   RenderSegments
	Mst      StationGraph
	Stats    Stats
	RNGCheck int
}

// Game implements ebiten.Game interface.
type Game struct {
	initialized bool

	screenWidth  int
	screenHeight int
	sizeChanged  bool

	toScreen ebiten.GeoM
	toWorld  ebiten.GeoM

	worldScale float64
	worldSize  Rect

	debug bool

	startTime time.Time
	now       time.Time
	elapsed   time.Duration

	streetGenerationEndTime time.Time

	noise         *ebiten.Image
	villagesAsync Promise[VillageCalculation, string]

	render  RenderSegments
	streets *ebiten.Image

	terrain Terrain

	hoveredStation     *Station
	selectedStationOne *Station
	selectedStationTwo *Station

	clicked      bool
	cursorWorld  Vec
	cursorScreen Vec

	rng              *rand.Rand
	streetsGenerator StreetGenerator
	seed             uint64

	btnAcceptConnection   *Button
	btnPlanningConnection *Button
	btnCancelConnection   *Button

	acceptedGraph StationGraph
	planningGraph StationGraph

	audio            Audio
	stats            Stats
	terrainGenerator *TerrainGenerator
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	_ = outsideWidth
	_ = outsideHeight

	// stay with a fixed screen size
	return g.screenWidth, g.screenHeight
}

func (g *Game) Reset(seed uint64) {
	*g = Game{
		initialized:  true,
		debug:        Debug,
		seed:         seed,
		audio:        g.audio,
		screenWidth:  g.screenWidth,
		screenHeight: g.screenHeight,
	}

	g.startTime = time.Now()
	g.now = time.Now()

	g.updateTransform()

	// calculate world size based on transformed screen size
	x0, y0 := g.toWorld.Apply(0, 0)
	x1, y1 := g.toWorld.Apply(float64(g.screenWidth), float64(g.screenHeight))
	g.worldSize = Rect{Min: Vec{X: x0, Y: y0}, Max: Vec{X: x1, Y: y1}}

	g.rng = RandWithSeed(seed)

	// generate terrain
	g.terrainGenerator = NewTerrainGenerator(g.rng, g.worldSize)
	g.terrainGenerator.GenerateRiver()
	g.terrainGenerator.GenerateRiver()
	g.terrain = g.terrainGenerator.Terrain()

	// discard streets outside of the visible world
	g.streetsGenerator = NewStreetGenerator(g.rng, g.worldSize, g.terrainGenerator.Terrain())

	g.streetsGenerator.StartOne(5_000)
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
	dt := now.Sub(g.now).Seconds()
	g.now = now
	g.elapsed = now.Sub(g.startTime)

	_ = dt

	var newSegmentCount int

	for g.streetsGenerator.More() && time.Since(now) < 12*time.Millisecond {
		if segment := g.streetsGenerator.Next(); segment != nil {
			// draw the segment to the street image
			g.render.Add(segment, g.toWorld)
			newSegmentCount += 1
		}
	}

	// check if we've finished remaining generation
	if newSegmentCount > 0 && !g.streetsGenerator.More() {
		g.streetGenerationEndTime = time.Now()

		// asynchronously calculate the villages
		g.villagesAsync = AsyncTask(g.computeVillages)
	}

	if res := g.villagesAsync.GetOnce(); res != nil {
		// keep updated values
		g.render = res.Render
		g.streets = res.Image
		g.stats = res.Stats
	}

	g.updateStreetsImage()

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

func (g *Game) updateStreetsImage() {
	dirty := g.render.Dirty

	// if we have no image, create one
	if g.streets == nil {
		g.streets = ebiten.NewImage(g.screenWidth, g.screenHeight)
		dirty = true
	}

	// re-render al streets if needed
	if dirty {
		g.streets.Fill(color.Transparent)

		// draw the streets to the image
		g.render.Draw(g.streets, g.toScreen)
	}
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
	inputIntercepted = g.btnPlanningConnection.Hover(g.cursorScreen) || inputIntercepted
	inputIntercepted = g.btnCancelConnection.Hover(g.cursorScreen) || inputIntercepted

	// check button inputs
	if g.btnAcceptConnection.IsClicked(g.cursorScreen, g.clicked) {
		// accept the station
		edge, _ := g.acceptedGraph.Insert(g.selectedStationOne, g.selectedStationTwo)
		edge.Created = g.now

		// and remove it from planning, if it is still in there
		g.planningGraph.Remove(g.selectedStationOne, g.selectedStationTwo)

		g.resetInput()
	}

	// update the amount of money spend
	g.stats.CoinsSpent = g.acceptedGraph.TotalPrice()
	g.stats.CoinsPlanned = g.planningGraph.TotalPrice()

	if g.btnPlanningConnection.IsClicked(g.cursorScreen, g.clicked) {
		// hide all buttons
		g.planningGraph.Insert(g.selectedStationOne, g.selectedStationTwo)
		g.resetInput()
	}

	if g.btnCancelConnection.IsClicked(g.cursorScreen, g.clicked) {
		// hide all buttons
		g.resetInput()
	}

	var currentStation *Station

	if !inputIntercepted {
		if result := g.villagesAsync.Get(); result != nil && len(result.Stations) > 0 {
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
		if g.selectedStationOne != nil && g.acceptedGraph.Has(g.selectedStationOne, currentStation) {
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

				// text should include the price
				price := priceOf(g.selectedStationOne, g.selectedStationTwo)
				acceptText := fmt.Sprintf("Build (%s)", price)

				// show the buttons near the click location
				buttonVec := g.cursorScreen.Add(splatVec(-16))
				g.btnAcceptConnection = NewButton(acceptText, buttonVec, BuildButtonColors)
				g.btnPlanningConnection = NewButton("Plan", buttonVec.Add(Vec{Y: 32 + 8}), PlanButtonColors)
				g.btnCancelConnection = NewButton("Cancel", buttonVec.Add(Vec{Y: 2 * (32 + 8)}), CancelButtonColors)

				// disable button if we do not have enough money
				g.btnAcceptConnection.Disabled = g.stats.CoinsAvailable() < price
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
	g.btnPlanningConnection = nil
	g.btnCancelConnection = nil
}

func (g *Game) computeVillages(yield func(string)) VillageCalculation {
	yield("Vectorize streets")
	var render RenderSegments
	for _, segment := range g.streetsGenerator.Segments() {
		render.Add(segment, g.toWorld)
	}

	yield("Render all streets to a new image")
	image := ebiten.NewImage(g.screenWidth, g.screenHeight)
	render.Draw(image, g.toScreen)

	// find villages
	yield("Collecting villages")
	villages := CollectVillages(g.rng, g.streetsGenerator.Grid())

	yield("Calculate clip rectangle")

	// do not place anything near the edge of the screen
	const clipThreshold = 1_500 // m
	clip := Rect{
		Min: g.worldSize.Min.Add(Vec{X: clipThreshold, Y: clipThreshold}),
		Max: g.worldSize.Max.Sub(Vec{X: clipThreshold, Y: clipThreshold}),
	}

	yield("Generating stations")
	stations := GenerateStations(g.rng, clip, villages)

	yield("Calculate mst")
	mst := computeMST(stations)

	return VillageCalculation{
		EndTime:  time.Now(),
		Villages: villages,
		Stations: stations,
		Image:    image,
		Render:   render,
		Mst:      mst,
		Stats: Stats{
			// calculate the amount of money the player should have available
			CoinsTotal: Coins(math.Ceil(float64(mst.TotalPrice())*1.05/10) * 10),
		},

		RNGCheck: g.rng.Int(),
	}
}

// Draw draws the game screen.
// Draw is called every frame (typically 1/60[s] for 60Hz display).
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(BackgroundColor)

	// draw river
	g.terrain.Draw(screen, g.toScreen)

	// draw background & streets
	screen.DrawImage(g.streets, nil)

	if g.debug {
		if ebiten.IsKeyPressed(ebiten.KeyN) {
			if g.noise == nil {
				// generate an image from noise
				g.noise = populationToImage(g.streetsGenerator.Noise(), g.screenWidth, g.screenHeight, g.toWorld)
			}

			screen.DrawImage(g.noise, nil)
		}

		if ebiten.IsKeyPressed(ebiten.KeyT) {
			g.terrainGenerator.DebugDraw(screen, g.toScreen)
		}
	}

	// check if any village should be highlighted
	if result := g.villagesAsync.Get(); result != nil {
		g.drawVillageCalculation(screen, result)
	}

	g.drawHUD(screen)

	g.btnAcceptConnection.Draw(screen)
	g.btnPlanningConnection.Draw(screen)
	g.btnCancelConnection.Draw(screen)

	if g.debug {
		g.DrawDebugText(screen)
	}
}

func (g *Game) drawHUD(screen *ebiten.Image) {
	textSpace := 100.0

	pos := Vec{X: float64(imageWidth(screen)-32) - textSpace, Y: 16}
	screenSize := imageSizeOf(screen)

	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleWithColor(rgbaOf(0xffffff40))
	op.GeoM.Scale(screenSize.X, 64)
	screen.DrawImage(whiteImage, op)

	if g.stats.CoinsTotal > 0 {
		msg := fmt.Sprintf("%d", g.stats.CoinsAvailable())
		DrawTextLeft(screen, msg, Font24, pos, HudTextColor)

		// draw the coin icon in-front of the text
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(pos.X-40, pos.Y)
		screen.DrawImage(assets.Coin(), op)
	}

	// if we're busy, paint a busy indicator
	if g.villagesAsync.Waiting() {
		center := imageSizeOf(screen).Mulf(0.5)
		pos := Vec{X: center.X, Y: pos.Y}
		DrawText(screen, "please wait...", Font24, pos, HudTextColor, text.AlignCenter, text.AlignStart)
	}
}

func (g *Game) drawVillageCalculation(screen *ebiten.Image, result *VillageCalculation) {
	if g.debug && ebiten.IsKeyPressed(ebiten.KeyS) {
		for _, edge := range result.Mst.Edges() {
			DrawStationConnection(screen, g.toScreen, edge.One, edge.Two, 0, true, StationColorHover)
		}
	}

	// walk through the edges we've planned and paint them
	for _, edge := range g.planningGraph.Edges() {
		DrawStationConnection(screen, g.toScreen, edge.One, edge.Two, 0, true, StationColorPlanned)
	}

	// walk through the edges we've constructed and paint them
	for _, edge := range g.acceptedGraph.Edges() {
		offset := time.Now().Sub(edge.Created)
		DrawStationConnection(screen, g.toScreen, edge.One, edge.Two, offset, false, StationColorConstructed)
	}

	// paint the edges of the currently planed route
	if g.selectedStationOne != nil && g.selectedStationTwo != nil {
		// we have two selected villages, draw a dummy connection between them
		DrawStationConnection(screen, g.toScreen, g.selectedStationOne, g.selectedStationTwo, 0, false, StationColorSelected)
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

	case len(g.acceptedGraph.EdgesOf(station)) > 0:
		stationColor = StationColorConstructed

	case len(g.planningGraph.EdgesOf(station)) > 0:
		stationColor = StationColorPlanned
	}

	return stationColor
}

func (g *Game) DrawDebugText(screen *ebiten.Image) {
	pos := splatVec(32)
	t := fmt.Sprintf("%1.1f fps, seed %d", ebiten.ActualFPS(), g.seed)
	DrawTextLeft(screen, t, Font16, pos, DebugColor)

	pos.Y += 24
	t = fmt.Sprintf("Street Segments: %d", len(g.streetsGenerator.segments))
	DrawTextLeft(screen, t, Font16, pos, DebugColor)

	if !g.streetGenerationEndTime.IsZero() {
		pos.Y += 24
		t = fmt.Sprintf("Street generation took %s", g.streetGenerationEndTime.Sub(g.startTime))
		DrawTextLeft(screen, t, Font16, pos, DebugColor)
	}

	if result := g.villagesAsync.Get(); result != nil {
		pos.Y += 24
		t = fmt.Sprintf("City generation took %s", result.EndTime.Sub(g.startTime))
		DrawTextLeft(screen, t, Font16, pos, DebugColor)

		pos.Y += 24
		checkValue := fmt.Sprintf("%x", result.RNGCheck)
		t = fmt.Sprintf("Random check value: %s", checkValue[:6])
		DrawTextLeft(screen, t, Font16, pos, DebugColor)
	}

	if progress := g.villagesAsync.Status(); progress != nil {
		pos.Y += 24
		t = *progress + "..."
		DrawTextLeft(screen, t, Font16, pos, DebugColor)
	}
}

func (g *Game) updateTransform() {
	// base size, used for scaling
	worldWidth := 32000.0

	scale := float64(g.screenWidth) / worldWidth
	g.worldScale = scale

	g.toScreen = ebiten.GeoM{}
	g.toScreen.Scale(scale, scale)

	// create an inverse of the transform to transform from screen coordinates
	// to world coordinates
	g.toWorld = g.toScreen
	g.toWorld.Invert()
}
