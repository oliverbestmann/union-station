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

type Terrain struct {
	Rivers []River

	scratch []ebiten.Vertex
}

type River struct {
	Lines    []Line
	Vertices []ebiten.Vertex
	Indices  []uint16

	Outline     []Line
	OutlineGrid Grid[Line]
}

type TerrainGenerator struct {
	noise *fastnoiselite.FastNoiseLite
	rng   *rand.Rand

	// clip rect of the world. No need to generate outside of the world
	world Rect

	// noise put into an image after generating
	debugNoiseImage *ebiten.Image

	// the generated terrain
	terrain Terrain
}

func NewTerrainGenerator(rng *rand.Rand, worldSize Rect) *TerrainGenerator {
	noise := fastnoiselite.NewNoise()
	noise.Seed = rng.Int32()
	noise.Frequency = 0.0001

	return &TerrainGenerator{
		noise: noise,
		rng:   rng,
		world: worldSize,
	}
}

func (t *Terrain) Draw(target *ebiten.Image, toScreen ebiten.GeoM) {
	var color colorm.ColorM
	color.ScaleWithColor(WaterColor)

	var top colorm.DrawTrianglesOptions
	top.AntiAlias = true

	for _, river := range t.Rivers {
		// bring vertices to screen
		trVertices := TransformVertices(toScreen, river.Vertices, &t.scratch)
		colorm.DrawTriangles(target, trVertices, river.Indices, whiteImage, color, &top)
	}
}

func (t *TerrainGenerator) DebugDraw(target *ebiten.Image, toScreen ebiten.GeoM) {
	if t.debugNoiseImage == nil {
		toWorld := toScreen
		toWorld.Invert()

		t.debugNoiseImage = noiseToImage(t.noise, target.Bounds().Dx(), target.Bounds().Dy(), toWorld)
	}

	target.DrawImage(t.debugNoiseImage, nil)
}

func (t *TerrainGenerator) Terrain() Terrain {
	return t.terrain
}

func (t *TerrainGenerator) GenerateRiver() {
	var lines []Line

	for {
		// generate a valid river path
		lines = t.riverCandidate()
		if lines == nil {
			continue
		}

		// check if we intersect another river
		for _, river := range t.terrain.Rivers {
		lines:
			for idx, line := range lines {
				for _, intersectionCandidate := range river.Lines {
					intersection, ok := intersectionCandidate.Intersection(line)
					if !ok {
						continue
					}

					// we got an intersection with a different river. shorten the current
					// line segment and stop the new river here
					lines[idx].End = intersection
					lines = lines[:idx]
					break lines
				}
			}
		}

		break
	}

	// create path from river lines
	path := pathOf(linesToVecs(lines), false)

	// now we have a river, create a path from it that has a given width
	width := Randf[float32](t.rng, float32(300), 600)

	// and generate vertices in world space
	vertices, indices := path.AppendVerticesAndIndicesForStroke(nil, nil, &vector.StrokeOptions{
		Width:    width,
		LineJoin: vector.LineJoinRound,
	})

	outline := verticesToLines(vertices, indices)

	river := River{
		Lines:       lines,
		Vertices:    vertices,
		Indices:     indices,
		Outline:     outline,
		OutlineGrid: NewGrid(splatVec(50), outline),
	}

	t.terrain.Rivers = append(t.terrain.Rivers, river)
}

func (t *TerrainGenerator) riverCandidate() []Line {
	stepSize := t.world.Width() / 100
	pointsIter := walk(t.rng, t.noise, t.world, stepSize)

	var points []Vec
	var insideCount int

	// collect points but limit if we reach an endless loop
	for point := range pointsIter {
		points = append(points, point)

		if len(points) > 1000 {
			// endless loop maybe, try a different path
			return nil
		}

		if t.world.Contains(point) {
			// count the points that are inside the world so we can score the river later
			insideCount += 1
		}
	}

	// river not really touching the map
	if insideCount < 40 {
		return nil
	}

	// broken river, discard this one
	if hasLoop(points, stepSize*0.99) {
		return nil
	}

	return vecsToLines(points)
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
		dir := directionTo(pos, world.Center())

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
		point := RandVecIn(rng, outer)

		if outer.Contains(point) && !inner.Contains(point) {
			return point
		}
	}
}

func vecsToLines(points []Vec) []Line {
	if len(points) < 2 {
		return nil
	}

	var lines []Line

	prev := points[0]
	for _, point := range points {
		lines = append(lines, Line{Start: prev, End: point})
		prev = point
	}

	return lines
}

func linesToVecs(lines []Line) []Vec {
	if len(lines) == 0 {
		return nil
	}

	points := []Vec{
		lines[0].Start,
	}

	for _, line := range lines {
		points = append(points, line.End)
	}

	return points
}

func verticesToLines(vertices []ebiten.Vertex, indices []uint16) []Line {
	if len(indices)%3 != 0 {
		panic("number of indices must be dividable by three")
	}

	lines := make([]Line, 0, len(indices))

	for idx := 0; idx < len(indices); idx += 3 {
		lines = append(lines, Line{
			Start: Vec{
				X: float64(vertices[indices[idx+0]].DstX),
				Y: float64(vertices[indices[idx+0]].DstY),
			},
			End: Vec{
				X: float64(vertices[indices[idx+1]].DstX),
				Y: float64(vertices[indices[idx+1]].DstY),
			},
		})

		lines = append(lines, Line{
			Start: Vec{
				X: float64(vertices[indices[idx+1]].DstX),
				Y: float64(vertices[indices[idx+1]].DstY),
			},
			End: Vec{
				X: float64(vertices[indices[idx+2]].DstX),
				Y: float64(vertices[indices[idx+2]].DstY),
			},
		})

		lines = append(lines, Line{
			Start: Vec{
				X: float64(vertices[indices[idx+2]].DstX),
				Y: float64(vertices[indices[idx+2]].DstY),
			},
			End: Vec{
				X: float64(vertices[indices[idx+0]].DstX),
				Y: float64(vertices[indices[idx+0]].DstY),
			},
		})
	}

	return lines
}
