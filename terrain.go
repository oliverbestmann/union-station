package main

import (
	"github.com/furui/fastnoiselite-go"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/vector"
	. "github.com/quasilyte/gmath"
	"iter"
	"math/rand/v2"
	"slices"
)

type TerrainGenerator struct {
	Noise           *fastnoiselite.FastNoiseLite
	rng             *rand.Rand
	world           Rect
	debugNoiseImage *ebiten.Image
	path            []Vec

	// vertices to render
	vertices []ebiten.Vertex
	indices  []uint16

	scratch []ebiten.Vertex
}

func NewTerrainGenerator(rng *rand.Rand, worldSize Rect) *TerrainGenerator {
	noise := fastnoiselite.NewNoise()
	noise.Seed = rng.Int32()
	noise.Frequency = 0.0001

	return &TerrainGenerator{
		Noise: noise,
		rng:   rng,
		world: worldSize,
	}
}

func (t *TerrainGenerator) Generate() {
	stepSize := t.world.Width() / 100

	pointsIter := walk(t.rng, t.Noise, t.world, stepSize)

outer:
	for {
		var points []Vec
		var insideCount int

		for point := range pointsIter {
			points = append(points, point)

			if len(points) > 1000 {
				// endless loop maybe?
				continue outer
			}

			if t.world.Contains(point) {
				insideCount += 1
			}
		}

		// broken river
		if hasLoop(t.path, stepSize*0.99) {
			continue
		}

		// river not really touching the map
		if insideCount < 40 {
			continue
		}

		// keep this one
		t.path = points
		break
	}

	// render river to path
	path := pathOf(t.path, false)

	// and generate vertices in world space
	t.vertices, t.indices = path.AppendVerticesAndIndicesForStroke(nil, nil, &vector.StrokeOptions{
		Width:    randf[float32](t.rng, float32(300), 600),
		LineJoin: vector.LineJoinRound,
	})
}

func (t *TerrainGenerator) Draw(target *ebiten.Image, toScreen ebiten.GeoM) {
	// bring vertices to screen
	trVertices := TransformVertices(toScreen, t.vertices, &t.scratch)

	var color colorm.ColorM
	color.ScaleWithColor(WaterColor)

	var top colorm.DrawTrianglesOptions
	top.AntiAlias = true

	colorm.DrawTriangles(target, trVertices, t.indices, whiteImage, color, &top)
}

func (t *TerrainGenerator) DebugDraw(target *ebiten.Image, toScreen ebiten.GeoM) {
	if t.debugNoiseImage == nil {
		toWorld := toScreen
		toWorld.Invert()
		t.debugNoiseImage = noiseToImage(t.Noise, target.Bounds().Dx(), target.Bounds().Dy(), toWorld)
	}

	target.DrawImage(t.debugNoiseImage, nil)

	path := pathOf(t.path, false)

	for _, pos := range t.path {
		posScreen := TransformVec(toScreen, pos).AsVec32()
		path.LineTo(posScreen.X, posScreen.Y)
	}

	vertices, indices := path.AppendVerticesAndIndicesForStroke(nil, nil, &vector.StrokeOptions{
		Width:    2.0,
		LineJoin: vector.LineJoinRound,
	})

	{
		// bring vertices to screen
		trVertices := TransformVertices(toScreen, vertices, &t.scratch)

		var color colorm.ColorM
		color.ScaleWithColor(DebugColor)
		colorm.DrawTriangles(target, trVertices, indices, whiteImage, color, nil)
	}

	{
		// bring vertices to screen
		trVertices := TransformVertices(toScreen, t.vertices, &t.scratch)

		var color colorm.ColorM
		color.ScaleWithColor(DebugColor)
		colorm.DrawTriangles(target, trVertices, t.indices, whiteImage, color, nil)
	}
}

func (t *TerrainGenerator) Water() []Vec {
	return t.path
}

func walk(rng *rand.Rand, noise *fastnoiselite.FastNoiseLite, world Rect, stepSize float64) iter.Seq[Vec] {
	type F = fastnoiselite.FNLfloat

	return func(yield func(Vec) bool) {
		// increase rectangle size slightly
		outer := Rect{
			Min: world.Min.Sub(world.Size().Mulf(0.1)),
			Max: world.Max.Add(world.Size().Mulf(0.1)),
		}

		// find a starting point that is in outer, but not in world
		pos := rectStart(rng, outer, world)

		// target the center of the screen
		dir := pos.DirectionTo(world.Center()).Mulf(-1)

		lookAhead := stepSize * 10

		var angles []Rad
		for deg := -10.0; deg <= 10; deg += 0.1 {
			angles = append(angles, DegToRad(deg))
		}

		// yield initial point
		if !yield(pos) {
			return
		}

		for {
			// calculate a new point
			pos = pos.Add(dir.Normalized().Mulf(stepSize))

			if !yield(pos) {
				return
			}

			if !outer.Contains(pos) {
				return
			}

			// check candidates in step direction
			angle := MaxOf(slices.Values(angles), func(angle Rad) float64 {
				pos := pos.Add(dir.Rotated(angle).Normalized().Mulf(lookAhead))
				return noise.GetNoise2D(F(pos.X), F(pos.Y))
			})

			// calculate new direction
			dir = dir.Rotated(angle)
		}
	}
}

func hasLoop(points []Vec, distThreshold float64) bool {
	distThresholdSqr := distThreshold * distThreshold

	for ia, a := range points {
		for ib, b := range points {
			if ia != ib && a.DistanceSquaredTo(b) < distThresholdSqr {
				return true
			}
		}
	}

	return false
}

func rectStart(rng *rand.Rand, outer, inner Rect) Vec {
	for {
		point := Vec{
			X: randf(rng, outer.Min.X, outer.Max.X),
			Y: randf(rng, outer.Min.Y, outer.Max.Y),
		}

		if outer.Contains(point) && !inner.Contains(point) {
			return point
		}
	}
}
