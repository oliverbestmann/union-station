package main

import (
	"container/heap"
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

type Segment struct {
	Connections []*Segment
	Start       Vec
	End         Vec
	Type        StreetType
}

func (s *Segment) BBox() Rect {
	minX := min(s.Start.X, s.End.X)
	maxX := max(s.Start.X, s.End.X)
	minY := min(s.Start.Y, s.End.Y)
	maxY := max(s.Start.Y, s.End.Y)

	return Rect{
		Min: Vec{X: minX, Y: minY},
		Max: Vec{X: maxX, Y: maxY},
	}
}

func (s *Segment) Intersects(other *Segment) bool {
	return lineIntersect(s.Start, s.End, other.Start, other.End)
}

func (s *Segment) Intersection(other *Segment) (Vec, bool) {
	return lineIntersection(s.Start, s.End, other.Start, other.End)
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

func (s *Segment) Angle() Rad {
	return s.Start.AngleToPoint(s.End)
}

func (s *Segment) Length() float64 {
	return s.Start.DistanceTo(s.End)
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

func (s *Segment) DistanceTo(other *Segment) float64 {
	distanceSqr := min(
		s.Start.DistanceSquaredTo(other.Start),
		s.Start.DistanceSquaredTo(other.End),
		s.End.DistanceSquaredTo(other.Start),
		s.End.DistanceSquaredTo(other.End),
	)

	return math.Sqrt(distanceSqr)
}

func (s *Segment) Center() Vec {
	return s.Start.Add(s.End).Mulf(0.5)
}

type PendingSegmentQueue []PendingSegment

func (p *PendingSegmentQueue) Len() int {
	return len(*p)
}

func (p *PendingSegmentQueue) Less(i, j int) bool {
	return (*p)[i].AtStep < (*p)[j].AtStep
}

func (p *PendingSegmentQueue) Swap(i, j int) {
	(*p)[i], (*p)[j] = (*p)[j], (*p)[i]
}

func (p *PendingSegmentQueue) Push(x any) {
	*p = append(*p, x.(PendingSegment))
}

func (p *PendingSegmentQueue) Pop() any {
	old := *p
	n := len(old)
	item := old[n-1]
	old[n-1] = PendingSegment{}
	*p = old[0 : n-1]
	return item
}

type StreetGenerator struct {
	Clip          Rect
	rng           *rand.Rand
	noise         *fastnoiselite.FastNoiseLite
	segmentsQueue PendingSegmentQueue
	segments      []*Segment
	grid          Grid
}

func NewStreetGenerator(rng *rand.Rand, clip Rect) StreetGenerator {
	noise := fastnoiselite.NewNoise()
	noise = fastnoiselite.NewNoise()
	noise.SetNoiseType(fastnoiselite.NoiseTypeValueCubic)
	noise.Seed = rng.Int32()
	noise.Frequency = 0.0008

	return StreetGenerator{
		Clip:  clip,
		rng:   rng,
		noise: noise,
	}
}

func (gen *StreetGenerator) Noise() *fastnoiselite.FastNoiseLite {
	return gen.noise
}

func (gen *StreetGenerator) More() bool {
	return len(gen.segmentsQueue) > 0
}

func (gen *StreetGenerator) Next() *Segment {
	if len(gen.segmentsQueue) == 0 {
		return nil
	}

	// get the next segment to start from
	prev := heap.Pop(&gen.segmentsQueue).(PendingSegment)

	distanceToPreviousFork := prev.DistanceToPreviousFork

	segment := gen.nextSegment(prev, DegToRad(1))
	segment.Type = prev.Type

	if !gen.Clip.Contains(segment.Start) && !gen.Clip.Contains(segment.End) {
		// skip if out of the screen
		return nil
	}

	gen.segments = append(gen.segments, segment)
	gen.grid.Insert(segment)

	// max distance when to connect to existing segments
	const connectThreshold = 30

	// check if we can find a point very near to our segments end
	bbox5 := segment.BBox()
	bbox5.Min = bbox5.Min.Sub(Vec{X: connectThreshold, Y: connectThreshold})
	bbox5.Max = bbox5.Max.Add(Vec{X: connectThreshold, Y: connectThreshold})
	for existing := range gen.grid.Candidates(segment, bbox5) {
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

	for existing := range gen.grid.Candidates(segment, Rect{}) {
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
					heap.Push(&gen.segmentsQueue, PendingSegment{
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

	heap.Push(&gen.segmentsQueue, PendingSegment{
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

		heap.Push(&gen.segmentsQueue, PendingSegment{
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
	heap.Push(&gen.segmentsQueue, p)
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
		Start: start,
		End:   end,
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
		length := randf(gen.rng, 50.0, 80.0)
		angle := prevAngle + randf(gen.rng, -maxAngle, +maxAngle)

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
	return noiseValueAt(gen.noise, point)
}

func noiseValueAt(noise *fastnoiselite.FastNoiseLite, point Vec) float64 {
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

func noiseToImage(noise *fastnoiselite.FastNoiseLite, width, height int, toWorld ebiten.GeoM) *ebiten.Image {
	pixels := make([]uint8, width*height*4)

	var pos int
	for y := range height {
		for x := range width {
			trX, trY := toWorld.Apply(float64(x), float64(y))

			noiseValue := noiseValueAt(noise, Vec{X: trX, Y: trY})

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

type GridCell struct {
	Segments []*Segment
}

type cellId struct {
	X int16
	Y int16
}

type Grid struct {
	cells map[cellId]*GridCell
}

func (g *Grid) getGridCell(id cellId, create bool) *GridCell {
	cell := g.cells[id]

	if cell == nil && create {
		if g.cells == nil {
			g.cells = make(map[cellId]*GridCell)
		}

		cell = &GridCell{}
		g.cells[id] = cell
	}

	return cell
}

func (g *Grid) CellsOf(bbox Rect, create bool) iter.Seq[*GridCell] {
	minId := cellId{
		X: int16(bbox.Min.X / 50),
		Y: int16(bbox.Min.Y / 50),
	}

	maxId := cellId{
		X: int16(math.Ceil(bbox.Max.X / 50)),
		Y: int16(math.Ceil(bbox.Max.Y / 50)),
	}

	return func(yield func(*GridCell) bool) {
		for y := minId.Y; y <= maxId.Y; y++ {
			for x := minId.X; x <= maxId.X; x++ {
				gridCell := g.getGridCell(cellId{X: x, Y: y}, create)
				if gridCell != nil {
					if !yield(gridCell) {
						return
					}
				}
			}
		}
	}

}

func (g *Grid) Insert(segment *Segment) {
	for cell := range g.CellsOf(segment.BBox(), true) {
		cell.Segments = append(cell.Segments, segment)
	}
}

func (g *Grid) Candidates(query *Segment, bbox Rect) iter.Seq[*Segment] {
	if bbox.IsZero() {
		bbox = query.BBox()
	}

	return func(yield func(*Segment) bool) {
		seen := make(map[*Segment]struct{})

		for cell := range g.CellsOf(bbox, false) {
			for _, segment := range cell.Segments {
				_, dup := seen[segment]

				if segment != query && !dup {
					// mark as seen
					seen[segment] = struct{}{}

					if !yield(segment) {
						return
					}
				}
			}
		}

	}
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

		// transform vertices to screen
		trVertices := r.tempVertices[:0]
		for _, vertex := range vertices {
			x, y := toScreen.Apply(float64(vertex.DstX), float64(vertex.DstY))
			vertex.DstX, vertex.DstY = float32(x), float32(y)
			trVertices = append(trVertices, vertex)
		}

		// render vertices
		op := &ebiten.DrawTrianglesOptions{}
		op.AntiAlias = true
		screen.DrawTriangles(trVertices, indices, whiteImage, op)

		// keep the slice to re-use
		r.tempVertices = trVertices[:0]
	}
}
