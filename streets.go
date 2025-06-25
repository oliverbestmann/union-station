package main

import (
	"github.com/furui/fastnoiselite-go"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	. "github.com/quasilyte/gmath"
	"iter"
	"math"
	"math/rand/v2"
)

type StreetType uint8

const StreetTypeHighway = 0
const StreetTypeLocal = 1

type PendingSegment struct {
	PreviousSegment        *Segment
	Point                  Vec
	Angle                  Rad
	DistanceToPreviousFork float64
	Type                   StreetType
	AtStep                 int
}

type Line struct {
	Start Vec
	End   Vec
}

func (l Line) BBox() Rect {
	minX := min(l.Start.X, l.End.X)
	maxX := max(l.Start.X, l.End.X)
	minY := min(l.Start.Y, l.End.Y)
	maxY := max(l.Start.Y, l.End.Y)

	return Rect{
		Min: Vec{X: minX, Y: minY},
		Max: Vec{X: maxX, Y: maxY},
	}
}

func (l Line) Intersects(other Line) bool {
	return lineIntersect(l.Start, l.End, other.Start, other.End)
}

func (l Line) Intersection(other Line) (Vec, bool) {
	return lineIntersection(l.Start, l.End, other.Start, other.End)
}

func (l Line) Direction() Vec {
	return directionTo(l.Start, l.End)
}

func (l Line) Angle() Rad {
	return l.Start.AngleToPoint(l.End)
}

func (l Line) Length() float64 {
	return l.Start.DistanceTo(l.End)
}

func (l Line) Center() Vec {
	return l.Start.Add(l.End).Mulf(0.5)
}

func (l Line) DistanceToVec(vec Vec) float64 {
	ab := l.End.Sub(l.Start)
	ap := vec.Sub(l.Start)

	var t float64
	if !ab.IsZero() {
		t = Clamp(ap.Dot(ab)/ab.LenSquared(), 0, 1)
	}

	closest := l.Start.Add(ab.Mulf(t))
	return closest.DistanceTo(vec)
}

func (l Line) DistanceToOther(other Line) float64 {
	distanceSqr := min(
		l.Start.DistanceSquaredTo(other.Start),
		l.Start.DistanceSquaredTo(other.End),
		l.End.DistanceSquaredTo(other.Start),
		l.End.DistanceSquaredTo(other.End),
	)

	return math.Sqrt(distanceSqr)
}

type Segment struct {
	Line
	Connections []*Segment
	Type        StreetType
}

func (s *Segment) Intersects(other *Segment) bool {
	return s.Line.Intersects(other.Line)
}

func (s *Segment) Intersection(other *Segment) (Vec, bool) {
	return s.Line.Intersection(other.Line)
}

func (s *Segment) Draw(target *ebiten.Image, g ebiten.GeoM) {
	dir := s.End.Sub(s.Start).Normalized()

	start := s.Start.Sub(dir.Mulf(4.0))
	end := s.End.Add(dir.Mulf(4.0))

	x0, y0 := g.Apply(start.X, start.Y)
	x1, y1 := g.Apply(end.X, end.Y)

	strokeWidth := 2.0
	strokeColor := rgbaOf(0x978c63ff)
	if s.Type == StreetTypeLocal {
		strokeWidth = 1.0
		strokeColor = rgbaOf(0xb9ab73ff)
	}

	vector.StrokeLine(target, float32(x0), float32(y0), float32(x1), float32(y1), float32(strokeWidth), strokeColor, true)
}

func (s *Segment) IsConnected(other *Segment) bool {
	for _, connected := range s.Connections {
		if connected == other {
			return true
		}
	}

	return false
}

func (s *Segment) Connect(other *Segment) {
	if !s.IsConnected(other) {
		s.Connections = append(s.Connections, other)
	}

	if !other.IsConnected(s) {
		other.Connections = append(other.Connections, s)
	}
}

func NewPendingSegmentQueue() Heap[PendingSegment] {
	return MakeHeap[PendingSegment](func(lhs, rhs PendingSegment) bool {
		return lhs.AtStep < rhs.AtStep
	})
}

type StreetGenerator struct {
	Clip          Rect
	rng           *rand.Rand
	noise         *fastnoiselite.FastNoiseLite
	segmentsQueue Heap[PendingSegment]
	segments      []*Segment
	grid          Grid[*Segment]
	terrain       Terrain
}

func NewStreetGenerator(rng *rand.Rand, clip Rect, terrain Terrain) StreetGenerator {
	noise := fastnoiselite.NewNoise()
	noise.SetNoiseType(fastnoiselite.NoiseTypeValueCubic)
	noise.Seed = rng.Int32()
	noise.Frequency = 0.0008

	return StreetGenerator{
		Clip:          clip,
		rng:           rng,
		noise:         noise,
		terrain:       terrain,
		segmentsQueue: NewPendingSegmentQueue(),
		grid:          NewGrid[*Segment](vecSplat(50), nil),
	}
}

func (gen *StreetGenerator) Grid() Grid[*Segment] {
	return gen.grid
}

func (gen *StreetGenerator) Noise() *fastnoiselite.FastNoiseLite {
	return gen.noise
}

func (gen *StreetGenerator) More() bool {
	return !gen.segmentsQueue.IsEmpty()
}

func (gen *StreetGenerator) Next() *Segment {
	if gen.segmentsQueue.IsEmpty() {
		return nil
	}

	// get the next segment to start from
	prev := gen.segmentsQueue.Pop()

	distanceToPreviousFork := prev.DistanceToPreviousFork

	segment := gen.nextSegment(prev, DegToRad(1))
	segment.Type = prev.Type

	if !gen.Clip.Contains(segment.Start) && !gen.Clip.Contains(segment.End) {
		// skip if out of the screen
		return nil
	}

	// kill the segment if it reaches the river
	if line, _, ok := gen.intersectsWater(segment); ok {
		if segment.Type == StreetTypeLocal {
			// discard, small streets never cross water
			return nil
		}

		// direction of the line we've hit
		dirWater := line.Direction()

		// direction of the street
		dirSegment := segment.Direction()

		if math.Abs(dirWater.Dot(dirSegment)) > 0.2 {
			return nil
		}

		segment.End = segment.End.Add(dirSegment.Mulf(1500))
	}

	gen.segments = append(gen.segments, segment)

	// only add to index at the end, we might still change
	// the points
	defer gen.grid.Insert(segment)

	// max distance when to connect to existing segments
	const connectThreshold = 30

	// check if we can find a point very near to our segments end
	bbox5 := segment.BBox()
	bbox5.Min = bbox5.Min.Sub(Vec{X: connectThreshold, Y: connectThreshold})
	bbox5.Max = bbox5.Max.Add(Vec{X: connectThreshold, Y: connectThreshold})
	for existing := range gen.grid.Candidates(bbox5) {
		if segment.IsConnected(existing) {
			continue
		}

		if existing.End.DistanceSquaredTo(segment.End) < connectThreshold*connectThreshold {
			segment.Connect(existing)
			segment.End = existing.End
			return segment
		}

		if existing.Start.DistanceSquaredTo(segment.End) < connectThreshold*connectThreshold {
			segment.Connect(existing)
			segment.End = existing.Start
			return segment
		}
	}

	for existing := range gen.grid.Candidates(segment.BBox()) {
		if segment.Intersects(existing) && !segment.IsConnected(existing) {
			// hit another segment.
			// we take the previous segment and connect it with the
			// segment that we have just hit. We also adjust its end to connect
			// to one of the points of the one we hit
			if prev := prev.PreviousSegment; prev != nil {
				// connect it with the segment we've hit
				segment.Connect(existing)

				// calculate the intersection point
				point, _ := segment.Intersection(existing)

				// shorten the segment to terminate at the intersection point
				segment.End = point
			}

			return segment
		}
	}

	const localStreetDensityThreshold = 0.25
	const highwayForkThreshold = 0.1

	if prev.Type == StreetTypeLocal {
		if gen.PopulationAt(prev.Point) < localStreetDensityThreshold || prob(gen.rng, 0.1) {
			// population not dense enough, stop here
			return nil
		}
	}

	if segment.Type == StreetTypeHighway {
		densityTrigger := prev.DistanceToPreviousFork > 350.0 && gen.PopulationAt(prev.Point) > highwayForkThreshold
		randomTrigger := prev.DistanceToPreviousFork > 500.0 && prob(gen.rng, 0.01)
		if densityTrigger || randomTrigger {
			for _, sign := range []Rad{1, -1} {
				if prob(gen.rng, 0.01) {
					// fork a highway from here in a 90 degree angle
					gen.segmentsQueue.Push(PendingSegment{
						PreviousSegment: segment,
						Point:           segment.End,
						Angle:           segment.Angle() + DegToRad(90)*sign,
						Type:            StreetTypeHighway,
						AtStep:          prev.AtStep + 20,
					})

					distanceToPreviousFork = 0
				}
			}
		}
	}

	gen.segmentsQueue.Push(PendingSegment{
		PreviousSegment:        segment,
		Point:                  segment.End,
		Angle:                  segment.Angle(),
		DistanceToPreviousFork: distanceToPreviousFork + segment.Length(),
		Type:                   prev.Type,
		AtStep:                 prev.AtStep + 10,
	})

	// if this is a high population neighbourhood, we create a small street
	if gen.PopulationAt(segment.End) > localStreetDensityThreshold && distanceToPreviousFork > 100.0 {
		var nextAtStep int

		if prev.Type == StreetTypeHighway {
			nextAtStep = prev.AtStep + 200000
		} else {
			nextAtStep = prev.AtStep + 200
		}

		gen.segmentsQueue.Push(PendingSegment{
			PreviousSegment:        segment,
			Point:                  segment.End,
			Angle:                  segment.Angle() + DegToRad(90)*Rad(Choose(gen.rng, -1, +1)),
			DistanceToPreviousFork: 0,
			Type:                   StreetTypeLocal,
			AtStep:                 nextAtStep,
		})
	}

	// tell the caller if we need to be called again
	return segment
}

func (gen *StreetGenerator) Push(p PendingSegment) {
	gen.segmentsQueue.Push(p)
}

func (gen *StreetGenerator) Segments() []*Segment {
	return gen.segments
}

func (gen *StreetGenerator) nextSegment(previousSegment PendingSegment, maxAngle Rad) *Segment {
	start := previousSegment.Point
	previousAngle := previousSegment.Angle

	// get the next vector for the new segment
	end := gen.nextVec(start, previousAngle, maxAngle)

	newSegment := Segment{
		Line: Line{
			Start: start,
			End:   end,
		},
	}

	if previousSegment.PreviousSegment != nil {
		newSegment.Connect(previousSegment.PreviousSegment)
	}

	return &newSegment
}

func (gen *StreetGenerator) nextVec(pos Vec, prevAngle Rad, maxAngle Rad) Vec {
	var best Vec
	var bestValue float64 = -1

	// try 8 angles and take the one with the highest population value
	for range 8 {
		length := Randf(gen.rng, 50.0, 80.0)
		angle := prevAngle + Randf(gen.rng, -maxAngle, +maxAngle)

		// the segment offset from the start pos
		offset := Vec{X: length}.Rotated(angle)

		for scale := range 10 {
			// look a little ahead and sample the population values
			noiseValue := gen.PopulationAt(pos.Add(offset.Mulf(5.0 + 2.0*float64(scale))))
			if noiseValue > bestValue {
				bestValue = noiseValue
				best = pos.Add(offset)
			}
		}
	}

	return best
}

func (gen *StreetGenerator) PopulationAt(point Vec) float64 {
	return populationValueAt(gen.noise, point)
}

func (gen *StreetGenerator) intersectsWater(segment *Segment) (line Line, point Vec, ok bool) {
	for _, river := range gen.terrain.Rivers {
		for candidate := range river.OutlineGrid.Candidates(segment.BBox()) {
			if pos, ok := candidate.Intersection(segment.Line); ok {
				return candidate, pos, true
			}
		}
	}

	return Line{}, Vec{}, false
}

func (gen *StreetGenerator) StartOne(distanceThreshold float64) {
outer:
	for {
		loc := RandVecIn(gen.rng, gen.Clip)

		for pending := range gen.segmentsQueue.Values() {
			if pending.Point.DistanceTo(loc) < distanceThreshold {
				continue outer
			}
		}

		for _, river := range gen.terrain.Rivers {
			rect := Rect{
				Min: loc.Sub(Vec{X: distanceThreshold, Y: distanceThreshold}),
				Max: loc.Add(Vec{X: distanceThreshold, Y: distanceThreshold}),
			}

			for candidate := range river.OutlineGrid.Candidates(rect) {
				if candidate.DistanceToVec(loc) < distanceThreshold {
					continue outer
				}
			}
		}

		// random angle
		angle := Randf(gen.rng, Rad(0), 2*math.Pi)

		// enqueue a starting point for the street generator
		gen.Push(PendingSegment{
			Point: loc,
			Angle: angle - math.Pi,
		})

		gen.Push(PendingSegment{
			Point: loc,
			Angle: angle,
		})

		return
	}
}

func populationValueAt(noise *fastnoiselite.FastNoiseLite, point Vec) float64 {
	value := noise.GetNoise2D(fastnoiselite.FNLfloat(point.X), fastnoiselite.FNLfloat(point.Y))
	return max(0, value)
}

// Cross product of two vectors
func cross(a, b Vec) float64 {
	return a.X*b.Y - a.Y*b.X
}

// Check if two line segments (p1-p2 and q1-q2) intersect
func lineIntersectionValues(p1, p2, q1, q2 Vec) (t, u float64) {
	r := p2.Sub(p1)
	s := q2.Sub(q1)
	denom := cross(r, s)

	if denom == 0 {
		// Lines are parallel
		return -1, -1
	}

	uNumerator := cross(q1.Sub(p1), r)
	tNumerator := cross(q1.Sub(p1), s)

	u = uNumerator / denom
	t = tNumerator / denom

	return
}

func lineIntersect(p1, p2, q1, q2 Vec) bool {
	t, u := lineIntersectionValues(p1, p2, q1, q2)
	// Check if t and u are within (0, 1) for segment-segment intersection
	return t > 0 && t < 1 && u > 0 && u < 1
}

// Check if two line segments (p1-p2 and q1-q2) intersect
func lineIntersection(p1, p2, q1, q2 Vec) (Vec, bool) {
	t, u := lineIntersectionValues(p1, p2, q1, q2)

	// Check if t and u are within (0, 1) for segment-segment intersection
	ok := t > 0 && t < 1 && u > 0 && u < 1
	return p1.Add(p2.Sub(p1).Mulf(t)), ok
}

func populationToImage(noise *fastnoiselite.FastNoiseLite, width, height int, toWorld ebiten.GeoM) *ebiten.Image {
	pixels := make([]uint8, width*height*4)

	var pos int
	for y := range height {
		for x := range width {
			trX, trY := toWorld.Apply(float64(x), float64(y))

			noiseValue := populationValueAt(noise, Vec{X: trX, Y: trY})

			pxValue := uint8(noiseValue * 0xff)

			if noiseValue > 0.25 {
				pixels[pos+1] = pxValue
			}

			pixels[pos+3] = pxValue

			pos += 4
		}
	}

	img := ebiten.NewImage(width, height)
	img.WritePixels(pixels)
	return img
}

func noiseToImage(noise *fastnoiselite.FastNoiseLite, width, height int, toWorld ebiten.GeoM) *ebiten.Image {
	pixels := make([]uint8, width*height*4)

	var pos int
	for y := range height {
		for x := range width {
			trX, trY := toWorld.Apply(float64(x), float64(y))

			noiseValue := noise.GetNoise2D(fastnoiselite.FNLfloat(trX), fastnoiselite.FNLfloat(trY))

			pxValue := uint8((noiseValue + 1) / 2 * 0xff)
			pixels[pos+0] = pxValue
			pixels[pos+1] = pxValue
			pixels[pos+2] = pxValue
			pixels[pos+3] = 0xff

			pos += 4
		}
	}

	img := ebiten.NewImage(width, height)
	img.WritePixels(pixels)
	return img
}

type HasBBox interface {
	comparable
	BBox() Rect
}

type GridCell[T HasBBox] struct {
	Objects []T
}

type cellId struct {
	X int16
	Y int16
}

type Grid[T HasBBox] struct {
	cellSize Vec
	cells    map[cellId]*GridCell[T]
}

func NewGrid[T HasBBox](cellSize Vec, objects []T) Grid[T] {
	grid := Grid[T]{
		cellSize: cellSize,
		cells:    map[cellId]*GridCell[T]{},
	}

	for _, obj := range objects {
		grid.Insert(obj)
	}

	return grid
}

func (g *Grid[T]) CellsOf(bbox Rect, create bool) iter.Seq[*GridCell[T]] {
	minId := cellId{
		X: int16(bbox.Min.X / g.cellSize.X),
		Y: int16(bbox.Min.Y / g.cellSize.Y),
	}

	maxId := cellId{
		X: int16(math.Ceil(bbox.Max.X / g.cellSize.X)),
		Y: int16(math.Ceil(bbox.Max.Y / g.cellSize.Y)),
	}

	return func(yield func(*GridCell[T]) bool) {
		for y := minId.Y; y <= maxId.Y; y++ {
			for x := minId.X; x <= maxId.X; x++ {
				gridCell := g.innerCellOf(cellId{X: x, Y: y}, create)
				if gridCell != nil {
					if !yield(gridCell) {
						return
					}
				}
			}
		}
	}

}

func (g *Grid[T]) Insert(obj T) {
	for cell := range g.CellsOf(obj.BBox(), true) {
		cell.Objects = append(cell.Objects, obj)
	}
}

func (g *Grid[T]) Candidates(bbox Rect) iter.Seq[T] {
	return func(yield func(T) bool) {
		var seen Set[T]

		for cell := range g.CellsOf(bbox, false) {
			for _, obj := range cell.Objects {
				if seen.Has(obj) {
					continue
				}

				// mark as seen
				seen.Insert(obj)

				if !yield(obj) {
					return
				}
			}
		}

	}
}

func (g *Grid[T]) innerCellOf(id cellId, create bool) *GridCell[T] {
	cell := g.cells[id]

	if cell == nil && create {
		if g.cells == nil {
			g.cells = make(map[cellId]*GridCell[T])
		}

		cell = &GridCell[T]{}
		g.cells[id] = cell
	}

	return cell
}

type RenderSegments struct {
	VerticesChunks [][]ebiten.Vertex
	IndicesChunks  [][]uint16

	Dirty        bool
	tempVertices []ebiten.Vertex
}

func (r *RenderSegments) Add(s *Segment, toWorld ebiten.GeoM) {
	r.Dirty = true

	dir := s.End.Sub(s.Start).Normalized()

	start := s.Start.Sub(dir.Mulf(4.0)).AsVec32()
	end := s.End.Add(dir.Mulf(4.0)).AsVec32()

	strokeWidth := 2.0
	strokeColor := rgbaOf(0x978c63ff)
	if s.Type == StreetTypeLocal {
		strokeWidth = 1.0
		strokeColor = rgbaOf(0xb9ab73ff)
	}

	chunksCount := len(r.VerticesChunks)
	if chunksCount == 0 || len(r.VerticesChunks[chunksCount-1]) > math.MaxUint16-128 {
		r.VerticesChunks = append(r.VerticesChunks, nil)
		r.IndicesChunks = append(r.IndicesChunks, nil)
	}

	chunkIdx := len(r.VerticesChunks) - 1

	// find a chunk that we'll write the segments to
	vertices := &r.VerticesChunks[chunkIdx]
	indices := &r.IndicesChunks[chunkIdx]

	// get the index where we place the new vertices
	vertexStart := len(*vertices)

	// create a path
	var path vector.Path
	path.MoveTo(start.X, start.Y)
	path.LineTo(end.X, end.Y)

	// append vertices from path to chunks
	strokeOp := &vector.StrokeOptions{}
	strokeOp.Width = float32(TransformScalar(toWorld, strokeWidth))
	*vertices, *indices = path.AppendVerticesAndIndicesForStroke(*vertices, *indices, strokeOp)

	for v := vertexStart; v < len(*vertices); v++ {
		(*vertices)[v].ColorR = float32(strokeColor.R) / 255
		(*vertices)[v].ColorG = float32(strokeColor.G) / 255
		(*vertices)[v].ColorB = float32(strokeColor.B) / 255
		(*vertices)[v].ColorA = float32(strokeColor.A) / 255
	}
}

func (r *RenderSegments) Draw(screen *ebiten.Image, toScreen ebiten.GeoM) {
	r.Dirty = false

	for chunk := range r.VerticesChunks {
		vertices := r.VerticesChunks[chunk]
		indices := r.IndicesChunks[chunk]

		r.tempVertices = TransformVertices(toScreen, vertices, r.tempVertices[:0])

		// render vertices
		op := &ebiten.DrawTrianglesOptions{}
		op.AntiAlias = true
		screen.DrawTriangles(r.tempVertices, indices, whiteImage, op)
	}
}

func (r *RenderSegments) Clear() {
	r.Dirty = false

	if len(r.VerticesChunks) > 0 {
		r.VerticesChunks = [][]ebiten.Vertex{r.VerticesChunks[0][:0]}
	}

	if len(r.IndicesChunks) > 0 {
		r.IndicesChunks = [][]uint16{r.IndicesChunks[0][:0]}
	}
}
