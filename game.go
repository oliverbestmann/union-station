package main

import (
	"fmt"
	"github.com/fogleman/ease"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/oliverbestmann/union-station/assets"
	"github.com/oliverbestmann/union-station/tween"
	. "github.com/quasilyte/gmath"
	"image/color"
	"math"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
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

type ResetOnUpdate struct {
	WantSimple bool
	NextSeed   uint64
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

	hoveredConnection  *StationEdge
	selectedConnection *StationEdge

	cursor       CursorState
	cursorWorld  Vec
	cursorScreen Vec

	rng              *rand.Rand
	streetsGenerator StreetGenerator
	seed             uint64

	btnAcceptConnection   *Button
	btnPlanningConnection *Button

	menu []*Button

	acceptedGraph StationGraph
	planningGraph StationGraph

	audio            Audio
	stats            Stats
	terrainGenerator *TerrainGenerator

	dialogStack         DialogStack
	loosingIsGuaranteed bool
	stationSize         float64

	profileStop func()

	tweens tween.Tweens

	lost bool
	won  bool

	resetOnUpdate *ResetOnUpdate
	leaderboard   Promise[Leaderboard, struct{}]

	score       int
	btnSettings *Button
	isSimple    bool
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	_ = outsideWidth
	_ = outsideHeight

	// stay with a fixed screen size
	return g.screenWidth, g.screenHeight
}

func (g *Game) Reset(reset ResetOnUpdate) {
	if reset.NextSeed == 0 {
		reset.NextSeed = g.nextSeed(reset.WantSimple)
	}

	seed := reset.NextSeed

	*g = Game{
		initialized:  true,
		debug:        Debug,
		seed:         seed,
		isSimple:     reset.WantSimple,
		audio:        g.audio,
		screenWidth:  g.screenWidth,
		screenHeight: g.screenHeight,
		dialogStack:  g.dialogStack,
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

	g.dialogStack.Clear()

	g.dialogStack.Push(Dialog{
		Id:    "city-generation",
		Modal: true,
		Texts: []Text{
			{
				Face:  Font24,
				Text:  "City generation in progress...",
				Color: DarkTextColor,
			},
		},
	})

	g.btnSettings = NewButton("", HudButtonColors)
	g.btnSettings.Size = Vec{X: 48, Y: 48}
	g.btnSettings.Position = Vec{X: float64(g.screenWidth-16) - g.btnSettings.Size.X, Y: 8}
	g.btnSettings.Image = assets.Settings()
	g.btnSettings.OnClick = g.showSettings

	// force update once
	_ = g.Update()
}

func (g *Game) Update() error {
	// initialize the game if needed
	if !g.initialized {
		g.Reset(ResetOnUpdate{
			NextSeed:   g.seed,
			WantSimple: false,
		})

		// start music audio playback only the first time
		g.audio.PlayMusic()
	}

	if g.resetOnUpdate != nil {
		// a reset is scheduled, resetting now
		g.Reset(*g.resetOnUpdate)
	}

	// step at the next reset
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.resetOnUpdate = &ResetOnUpdate{
			NextSeed: g.seed + 1,
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyP) && g.profileStop == nil {
		g.profileStop = ProfileStart()
	}

	if inpututil.IsKeyJustReleased(ebiten.KeyP) && g.profileStop != nil {
		g.profileStop()
		g.profileStop = nil
	}

	// calculate delta time for animations
	now := time.Now()
	dt := now.Sub(g.now)
	g.now = now

	g.elapsed = now.Sub(g.startTime)

	dtSecs := dt.Seconds()

	g.tweens.Update(dt)

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

		g.acceptedGraph.Stations = res.Stations
		g.planningGraph.Stations = res.Stations

		g.dialogStack.CloseById("city-generation")

		g.stationSize = 0.0
	}

	g.checkLeaderboardResponse()

	g.stationSize = g.stationSize + dtSecs*2

	g.updateStreetsImage()

	// get click information
	g.cursor = Cursor()
	g.cursorScreen = g.cursor.Position

	// derive screen cursor position
	g.cursorWorld = TransformVec(g.toWorld, g.cursorScreen)

	// play sound if click was done
	if g.cursor.JustPressed {
		g.audio.Play(g.audio.ButtonPress)
	}

	modal := g.dialogStack.Update(dtSecs)

	if modal {
		g.hoveredStation = nil
		g.hoveredConnection = nil
	} else {
		// now process input
		g.Input()
	}

	// check if we can still finish the game
	g.updateWinCondition()

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
		// g.streets.Clear()

		// draw the streets to the image
		g.render.Draw(g.streets, g.toScreen)
		g.render.Clear()
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
	inputIntercepted = g.btnAcceptConnection.Hover(g.cursor) || inputIntercepted
	inputIntercepted = g.btnPlanningConnection.Hover(g.cursor) || inputIntercepted
	inputIntercepted = g.btnSettings.Hover(g.cursor) || inputIntercepted

	// handled via callback
	g.btnSettings.Clicked(g.cursor)

	// handled via callback
	for _, button := range g.menu {
		inputIntercepted = button.Hover(g.cursor) || inputIntercepted
		button.Clicked(g.cursor)
	}

	// check button inputs
	if g.btnAcceptConnection.Clicked(g.cursor) {
		accepted := &g.acceptedGraph

		var newlyConnectedCount int

		if !g.villageIsConnected(g.selectedStationOne.Village) {
			newlyConnectedCount += g.selectedStationOne.Village.PopulationCount
		}

		if !g.villageIsConnected(g.selectedStationTwo.Village) {
			if g.selectedStationOne.Village != g.selectedStationTwo.Village {
				newlyConnectedCount += g.selectedStationTwo.Village.PopulationCount
			}
		}

		// accept the station
		accepted.Insert(StationEdge{
			One:     g.selectedStationOne,
			Two:     g.selectedStationTwo,
			Created: g.now,
		})

		// and remove it from planning, if it is still in there
		g.planningGraph.Remove(g.selectedStationOne, g.selectedStationTwo)

		// count the number of stations connected
		var stationsConnected int

		for _, station := range accepted.Stations {
			if accepted.HasConnections(station) {
				stationsConnected += 1
			}
		}

		// calculate score increase
		stationCount := len(accepted.Stations)
		scoreUpdate := (stationCount - (stationsConnected - 1)) * newlyConnectedCount / stationCount

		// update the score based on the number of stations already connected and the number
		// of newly connected peopled
		g.stats.Score += scoreUpdate
		g.stats.StationsConnected = stationsConnected

		g.resetInput()
	}

	// update the amount of money spend
	g.stats.CoinsSpent = g.acceptedGraph.TotalPrice()
	g.stats.CoinsPlanned = g.planningGraph.TotalPrice()

	if g.btnPlanningConnection.Clicked(g.cursor) {
		if g.planningGraph.Has(g.selectedStationOne, g.selectedStationTwo) {
			// was already planed, remove it from the graph
			g.planningGraph.Remove(g.selectedStationOne, g.selectedStationTwo)
		} else {
			// add it to the planning graph
			g.planningGraph.Insert(StationEdge{
				Created: g.now,
				One:     g.selectedStationOne,
				Two:     g.selectedStationTwo,
			})
		}

		g.resetInput()
	}

	var currentStation *Station

	var closestConnection *StationEdge
	var distanceToClosestConnection = math.Inf(1)

	if !inputIntercepted {
		// find the connection we are closest to
		if g.selectedStationOne == nil && g.selectedStationTwo == nil {
			edge, distance, ok := MaxOf(slices.Values(g.planningGraph.Edges()), func(value StationEdge) float64 {
				line := Line{
					Start: value.One.Position,
					End:   value.Two.Position,
				}

				return -line.DistanceToVec(g.cursorWorld)
			})

			// need to flip it due to MaxOf/MinOf, also scale to screen space
			distance = TransformScalar(g.toScreen, -distance)

			if ok && distance < 32 {
				closestConnection = &edge
				distanceToClosestConnection = distance
			}
		}

		if result := g.villagesAsync.Get(); result != nil && len(result.Stations) > 0 {
			// get the station that is nearest to the mouse
			station, _, _ := MaxOf(slices.Values(result.Stations), func(station *Station) float64 {
				return -g.cursorWorld.DistanceSquaredTo(station.Position)
			})

			// calculate distance to station in screen space
			stationScreen := TransformVec(g.toScreen, station.Position)
			distanceToStation := g.cursorScreen.DistanceTo(stationScreen)
			isNear := distanceToStation < 32.0
			isNotSelected := station != g.selectedStationOne && station != g.selectedStationTwo

			edgeIsCloser := distanceToClosestConnection < distanceToStation

			if isNotSelected && (isNear || (!edgeIsCloser && station.Village.Contains(g.cursorWorld))) {
				currentStation = station
			}
		}

		if currentStation != nil {
			// we either select a station or a connection, but not both
			closestConnection = nil
			distanceToClosestConnection = math.Inf(1)
		}

		noStationSelected := currentStation == nil

		// if the hovered station is already connected to the first station, we do not allow to hover or click it
		if g.selectedStationOne != nil && g.acceptedGraph.Has(g.selectedStationOne, currentStation) {
			currentStation = nil
		}

		if g.cursor.JustPressed {
			var twoSelected = false

			switch {
			case closestConnection != nil:
				g.selectedConnection = closestConnection
				g.selectedStationOne = closestConnection.One
				g.selectedStationTwo = closestConnection.Two
				twoSelected = true

			case noStationSelected:
				g.resetInput()

			case g.selectedStationOne == nil:
				// select the clicked village (or nil, if none was clicked)
				g.selectedStationOne = currentStation

			case g.selectedStationOne != nil && currentStation != nil && currentStation != g.selectedStationOne:
				// select the clicked village (or nil, if none was clicked)
				g.selectedStationTwo = currentStation
				twoSelected = true
			}

			if twoSelected {
				if g.selectedConnection == nil {
					// check if we have actually a planned connection in the graph
					edge, ok := g.planningGraph.Get(g.selectedStationOne, g.selectedStationTwo)
					if ok {
						g.selectedConnection = &edge
					}
				}

				// text should include the price
				price := priceOf(g.selectedStationOne, g.selectedStationTwo)
				acceptText := fmt.Sprintf("Build (%s)", price)

				g.btnAcceptConnection = NewButton(acceptText, BuildButtonColors)
				g.btnPlanningConnection = NewButton("Plan", PlanButtonColors)

				if g.selectedConnection != nil {
					g.btnPlanningConnection.Text = "Remove"
				}

				// show the buttons near the click location
				buttonsOrigin := g.cursorScreen.Add(Vec{X: -64, Y: -24})

				buttons := []*Button{
					g.btnAcceptConnection,
					g.btnPlanningConnection,
				}

				LayoutButtonsColumn(buttonsOrigin, 8, buttons...)

				for idx, button := range buttons {
					delay := time.Duration(idx * 250)

					g.slideIn(button, delay)

					button.Alpha = 0
					button.Position.X -= 16
				}

				// disable button if we do not have enough money
				g.btnAcceptConnection.Disabled = g.stats.CoinsAvailable() < price
			}
		}
	}

	g.hoveredStation = currentStation
	g.hoveredConnection = closestConnection

	if currentStation == g.selectedStationOne || currentStation == g.selectedStationTwo {
		// actually, do not hover one of the selected villages
		g.hoveredStation = nil
	}

	if g.selectedStationOne != nil && g.selectedStationTwo != nil {
		// both are selected, do not hover anything else
		g.hoveredStation = nil
	}

	if pointToEqual(g.hoveredConnection, g.selectedConnection) {
		g.hoveredConnection = nil
	}
}

func pointToEqual[T comparable](a, b *T) bool {
	if a != nil && b != nil {
		return *a == *b
	}

	return a == b
}

func (g *Game) resetInput() {
	g.selectedStationOne = nil
	g.selectedStationTwo = nil
	g.selectedConnection = nil
	g.btnAcceptConnection = nil
	g.btnPlanningConnection = nil
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
	mst := BuildMST(StationGraph{Stations: stations})

	return VillageCalculation{
		EndTime:  time.Now(),
		Villages: villages,
		Stations: stations,
		Image:    image,
		Render:   render,
		Mst:      mst,
		Stats: Stats{
			// calculate the amount of money the player should have available
			CoinsTotal:    Coins(math.Ceil(float64(mst.TotalPrice())*1.05/10) * 10),
			StationsTotal: len(stations),
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

	if !g.debug {
		pos := imageSizeOf(screen).Mulf(0.5)
		DrawTextCenter(screen, "THIS GAME IS\nWORK IN PROGRESS", Font64, pos, rgbaOf(0x00000030))
	}

	pos := imageSizeOf(screen).Sub(Vec{X: 16, Y: 16 + 12})
	DrawTextRight(screen, fmt.Sprintf("Level: %d", g.seed), Font12, pos, rgbaOf(0x00000030))

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

	g.dialogStack.Draw(screen)

	g.btnAcceptConnection.Draw(screen)
	g.btnPlanningConnection.Draw(screen)

	if g.debug {
		g.DrawDebugText(screen)
	}
}

func (g *Game) dotsByTime(text string) string {
	trimmed := strings.TrimRight(text, ".")
	diff := len(text) - len(trimmed)

	cut := int(g.elapsed.Seconds()*8) % (diff + 1)
	return text[:len(trimmed)+cut]
}

func (g *Game) drawVillageCalculation(screen *ebiten.Image, result *VillageCalculation) {
	// walk through the edges we've planned and paint them
	for _, edge := range g.planningGraph.Edges() {
		hovered := g.hoveredConnection != nil && *g.hoveredConnection == edge
		selected := g.selectedConnection != nil && *g.selectedConnection == edge

		var c color.Color
		switch {
		case selected:
			c = StationColorSelected.Stroke

		case hovered:
			c = StationColorHover.Stroke

		default:
			c = StationColorPlanned.Stroke
		}

		DrawStationConnection(screen, g.toScreen, edge.One, edge.Two, 0, true, c)
	}

	// walk through the edges we've constructed and paint them
	for _, edge := range g.acceptedGraph.Edges() {
		offset := time.Now().Sub(edge.Created)
		DrawStationConnection(screen, g.toScreen, edge.One, edge.Two, offset, false, StationColorConstructed.Stroke)
	}

	if g.debug {
		// remaining best solution
		if ebiten.IsKeyPressed(ebiten.KeyS) {
			mst := BuildMST(g.acceptedGraph)
			for _, edge := range mst.Edges() {
				DrawStationConnection(screen, g.toScreen, edge.One, edge.Two, 0, true, DebugColor)
			}
		}

		// best solution
		if ebiten.IsKeyPressed(ebiten.KeyB) {
			for _, edge := range result.Mst.Edges() {
				DrawStationConnection(screen, g.toScreen, edge.One, edge.Two, 0, true, DebugColor)
			}
		}
	}

	// paint the edges of the currently planed route
	if g.selectedStationOne != nil && g.selectedStationTwo != nil {
		// we have two selected villages, draw a dummy connection between them
		DrawStationConnection(screen, g.toScreen, g.selectedStationOne, g.selectedStationTwo, 0, false, StationColorSelected.Stroke)
	}

	if station := g.hoveredStation; station != nil {
		DrawVillageBounds(screen, station.Village, DrawVillageBoundsOptions{
			ToScreen:  g.toScreen,
			FillColor: color.RGBA{R: 0xb0, G: 0x89, B: 0xab, A: 0x30},
		})
	}

	if station := g.selectedStationOne; station != nil {
		DrawVillageBounds(screen, station.Village, DrawVillageBoundsOptions{
			ToScreen:  g.toScreen,
			FillColor: color.RGBA{R: 0xb0, G: 0x89, B: 0xab, A: 0x50},
		})
	}

	if station := g.selectedStationTwo; station != nil {
		DrawVillageBounds(screen, station.Village, DrawVillageBoundsOptions{
			ToScreen:  g.toScreen,
			FillColor: color.RGBA{R: 0xb0, G: 0x89, B: 0xab, A: 0x50},
		})
	}

	// paint the stations
	for idx, station := range result.Stations {
		loc := TransformVec(g.toScreen, station.Position)

		stationColor, pressed := g.stationColorOf(station)

		f := ease.OutElastic(max(0, min(1, g.stationSize-float64(idx)*0.1)))

		rOuter := 10 * f
		rInner := 8 * f

		DrawFillCircle(screen, loc.Add(vecSplat(2)), rOuter, ShadowColor)

		offset := vecSplat(iff(pressed, 1.0, 0))
		DrawFillCircle(screen, loc.Add(offset), rOuter, stationColor.Stroke)
		DrawFillCircle(screen, loc.Add(offset), rInner, stationColor.Fill)
	}

	if g.btnAcceptConnection == nil {
		if station := g.hoveredStation; station != nil {
			g.drawVillageTooltip(screen, g.cursorScreen, station.Village)
		}
	}
}

func (g *Game) stationColorOf(station *Station) (StationColor, bool) {
	// if the circle is hovered, select a different color palette
	switch {
	case g.selectedStationOne == station, g.selectedStationTwo == station:
		return StationColorSelected, true

	case g.hoveredStation == station:
		return StationColorHover, true

	case len(g.acceptedGraph.EdgesOf(station)) > 0:
		return StationColorConstructed, false

	case len(g.planningGraph.EdgesOf(station)) > 0:
		return StationColorPlanned, false

	default:
		return StationColorIdle, false
	}
}

func (g *Game) DrawDebugText(screen *ebiten.Image) {
	pos := vecSplat(32)
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

func (g *Game) updateWinCondition() {
	if g.lost || g.won || len(g.acceptedGraph.Stations) == 0 {
		return
	}

	var actionAvailable bool
	var hasConnected bool
	var hasUnconnected bool

	// check if there is no station left that we can connect
	// to any connected station
outer:
	for _, station := range g.acceptedGraph.Stations {
		if g.acceptedGraph.HasConnections(station) {
			hasConnected = true
			continue
		}

		hasUnconnected = true

		// station is not yet connected, check for the chepest connection to
		// an already connected node
		for _, other := range g.acceptedGraph.Stations {
			if !g.acceptedGraph.HasConnections(other) {
				continue
			}

			if priceOf(station, other) < g.stats.CoinsAvailable() {
				// reachable
				actionAvailable = true
				break outer
			}
		}
	}

	// check if all stations are connected

	// no unconnected station.
	if !hasUnconnected {
		// check if we have seen all stations
		if g.allStationsConnected() {
			// player has won
			g.won = true

			g.dialogStack.Push(Dialog{
				Id:    "won",
				Modal: true,
				Texts: []Text{
					{
						Face:  Font24,
						Text:  "Brilliant work, Engineer!",
						Color: DarkTextColor,
					},

					{
						Face:   Font16,
						Text:   "You’ve gone and done it — the countryside’s all linked up, and the rails are running",
						Color:  DarkTextColor,
						Offset: Vec{Y: 8},
					},

					{
						Face:  Font16,
						Text:  "smoother than a fresh cuppa on a rainy day. Top marks for a job well done! Now",
						Color: DarkTextColor,
					},

					{
						Face:  Font16,
						Text:  "why not pop down below and check the leaderboard?",
						Color: DarkTextColor,
					},

					{
						Face:  Font16,
						Text:  "Let’s see how your brilliant network stacks up against the rest!",
						Color: DarkTextColor,
					},

					{
						Face:   Font16,
						Text:   "Please hold tight, loading the leaderboard now...",
						Color:  DarkTextColor,
						Offset: Vec{X: 8},
					},
				},

				Buttons: []*Button{
					NewButton("Onwards!", AcceptButtonColors).WithOnClick(func() {
						g.resetOnUpdate = &ResetOnUpdate{
							NextSeed: g.nextSeed(g.isSimple),
						}
					}),
				},
			})

			g.reportScore()
		}

		return
	}

	// if there is no further action available, the player has lost
	if hasConnected && !actionAvailable {
		g.lost = true

		solution := BuildMST(g.acceptedGraph)
		missingConnections := len(solution.Edges()) - len(g.acceptedGraph.Edges())
		missingCoins := solution.TotalPrice() - g.acceptedGraph.TotalPrice() - g.stats.CoinsAvailable()

		g.dialogStack.Push(Dialog{
			Id:    "lost",
			Modal: true,
			Texts: []Text{
				{
					Face:  Font24,
					Text:  "Unlucky this time, mate",
					Color: DarkTextColor,
				},

				{
					Face:   Font16,
					Text:   "You gave it a proper go, but the countryside’s still a bit disconnected and the rails",
					Color:  DarkTextColor,
					Offset: Vec{Y: 8},
				},

				{
					Face:  Font16,
					Text:  fmt.Sprintf("didn’t quite make it to glory. You’re %d connections and %d quid short", missingConnections, missingCoins),
					Color: DarkTextColor,
				},

				{
					Face:  Font16,
					Text:  "of the mark, I’m afraid. Still, no shame in trying — even the best conductors",
					Color: DarkTextColor,
				},

				{
					Face:  Font16,
					Text:  "have a bumpy ride now and then. Have another crack, and maybe next time your",
					Color: DarkTextColor,
				},

				{
					Face:  Font16,
					Text:  "brilliant network will make it onto the leaderboard!",
					Color: DarkTextColor,
				},
			},

			Buttons: []*Button{
				NewButton("Have another go", AcceptButtonColors).WithAutoSize().WithOnClick(func() {
					g.resetOnUpdate = &ResetOnUpdate{
						NextSeed: g.seed,
					}
				}),

				NewButton("Onwards!", AcceptButtonColors).WithAutoSize().WithOnClick(func() {
					g.resetOnUpdate = &ResetOnUpdate{
						NextSeed: g.nextSeed(g.isSimple),
					}
				}),
			},
		})

		return
	}
}

func (g *Game) allStationsConnected() bool {
	graph := g.acceptedGraph

	var seen Set[*Station]

	queue := make([]*Station, 0, len(graph.Stations))

	initial := graph.Stations[0]
	queue = append(queue, initial)
	seen.Insert(initial)

	for idx := 0; idx < len(queue); idx++ {
		current := queue[idx]

		for _, edge := range graph.EdgesOf(current) {
			other := edge.OtherStation(current)

			if seen.Has(other) {
				continue
			}

			queue = append(queue, other)
			seen.Insert(other)
		}
	}

	return seen.Len() == len(graph.Stations)
}

func (g *Game) nextSeed(wantSimple bool) uint64 {
	simple := []uint64{
		47,
		49,
		51,
		53,
		63,
	}

	nice := []uint64{
		18,
		17,
		35,
		48,
		62,
	}

	levels := iff(wantSimple, simple, nice)

	nextSeed := levels[0]

	idx := slices.Index(levels, g.seed) + 1
	if idx < len(levels) {
		nextSeed = levels[idx]
	} else {
		// we are out of maps, what now?
		nextSeed = levels[0]
	}

	return nextSeed
}

func (g *Game) reportScore() {
	playerName := PlayerName()
	g.leaderboard = ReportHighscore(g.seed, playerName, g.stats.Score)
}

func (g *Game) checkLeaderboardResponse() {
	if result := g.leaderboard.GetOnce(); result != nil {
		dialog := g.dialogStack.ById("won")
		if dialog == nil {
			return
		}

		// remove the last text segment
		dialog.Texts = dialog.Texts[:len(dialog.Texts)-1]

		availableWidth := MeasureTexts(dialog.Texts).X

		// limit items
		items := result.Items
		if len(items) > 20 {
			items = items[:20]
		}

		// and add one line per highscore entry
		for idx, item := range items {
			yOffset := iff(idx == 0, 8.0, 0)

			dialog.Texts = append(dialog.Texts, Text{
				Face:   Font16,
				Text:   item.Player,
				Color:  DarkTextColor,
				Height: new(float64),
				Offset: Vec{Y: yOffset},
			})

			// manually right align with availableWidth
			scoreStr := strconv.Itoa(item.Score)
			x := availableWidth - MeasureText(Font16, scoreStr).X

			dialog.Texts = append(dialog.Texts, Text{
				Face:   Font16,
				Text:   scoreStr,
				Color:  DarkTextColor,
				Offset: Vec{X: x},
			})
		}
	}
}

func (g *Game) villageIsConnected(village *Village) bool {
	for _, station := range g.acceptedGraph.Stations {
		if station.Village == village && g.acceptedGraph.HasConnections(station) {
			return true
		}
	}

	return false
}

func (g *Game) showSettings() {
	g.menu = g.menu[:0]

	buttonSize := NewButton("", HudButtonColors).Size
	pos := Vec{X: float64(g.screenWidth) - 32 - buttonSize.X, Y: 64 + 24}

	var delay time.Duration

	add := func(btn *Button) *Button {
		btn.Position = pos
		btn.Alpha = 0
		g.menu = append(g.menu, btn)
		g.slideIn(btn, delay)
		pos.Y += buttonSize.Y + 16
		delay += 50 * time.Millisecond
		return btn
	}

	add(NewButton("Random level", HudButtonColors)).WithOnClick(func() {
		g.resetOnUpdate = &ResetOnUpdate{
			NextSeed: rand.Uint64(),
		}
	})

	muteText := func() string { return iff(g.audio.Mute, "Unmute", "Mute") }
	mute := add(NewButton(muteText(), HudButtonColors))
	mute.OnClick = func() {
		g.audio.ToggleMute()
		mute.Text = muteText()
	}

	add(NewButton("Simple level", HudButtonColors)).WithOnClick(func() {
		g.resetOnUpdate = &ResetOnUpdate{
			WantSimple: true,
			NextSeed:   g.nextSeed(true),
		}
	})

	add(NewButton("Complex level", HudButtonColors)).WithOnClick(func() {
		g.resetOnUpdate = &ResetOnUpdate{
			WantSimple: false,
			NextSeed:   g.nextSeed(false),
		}
	})
}

func (g *Game) slideIn(button *Button, delay time.Duration) {
	g.tweens.Add(tween.Delay(delay, tween.Concurrent(
		&tween.Simple{
			Ease:     ease.OutQuad,
			Duration: 250 * time.Millisecond,
			Target:   tween.LerpValue(&button.Position.X, button.Position.X-16, button.Position.X),
		},
		&tween.Simple{
			Ease:     ease.OutQuad,
			Duration: 250 * time.Millisecond,
			Target:   tween.LerpValue(&button.Alpha, 0, 1),
		},
	)))
}
