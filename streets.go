package main

import (
	"container/heap"
	"encoding/binary"
	"github.com/furui/fastnoiselite-go"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	. "github.com/quasilyte/gmath"
	"iter"
	"math/rand/v2"
)

type StreetType int

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

	Center Vec
	Radius float64

	Type StreetType
}

func (s *Segment) Intersects(other *Segment) bool {
	// quick test by comparing the circles. They can not intersect if the
	// center point of both lines are too far apart
	if s.Center.DistanceTo(other.Center) > s.Radius+other.Radius {
		return false
	}

	return lineIntersect(s.Start, s.End, other.Start, other.End)
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

func (s *Segment) ConnectedTo(other *Segment) bool {
	for _, connected := range s.Connections {
		if connected == other {
			return true
		}
	}

	return false
}

func (s *Segment) Connect(other *Segment) {
	if !s.ConnectedTo(other) {
		s.Connections = append(s.Connections, other)
	}

	if !other.ConnectedTo(s) {
		other.Connections = append(other.Connections, s)
	}
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

func NewStreetGenerator(clip Rect, seed uint64) StreetGenerator {
	var buf [32]byte
	binary.LittleEndian.AppendUint64(buf[:0], seed)

	rng := rand.New(rand.NewChaCha8(buf))

	noise := fastnoiselite.NewNoise()
	noise = fastnoiselite.NewNoise()
	noise.SetNoiseType(fastnoiselite.NoiseTypeValueCubic)
	noise.Seed = rng.Int32()
	noise.Frequency = 0.001

	return StreetGenerator{
		Clip:  clip,
		rng:   rng,
		noise: noise,
	}
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

	for existing := range gen.grid.Candidates(segment) {
		if segment.Intersects(existing) && !segment.ConnectedTo(existing) {
			// hit another segment.
			// we take the previous segment and connect it with the
			// segment that we have just hit. We also adjust its end to connect
			// to one of the points of the one we hit
			if prev := prev.PreviousSegment; prev != nil {
				prev.Connect(existing)

				if prev.End.DistanceSquaredTo(existing.Start) < prev.End.DistanceSquaredTo(existing.End) {
					prev.End = existing.Start
				} else {
					prev.End = existing.End
				}
			}

			return nil
		}
	}

	// check if we can find a point very near to our segments end
	for _, existing := range gen.segments {
		const threshold = 5 * 5

		if existing.End.DistanceSquaredTo(segment.End) < threshold {
			segment.Connect(existing)
			segment.End = existing.End
			break
		}

		if existing.Start.DistanceSquaredTo(segment.End) < threshold {
			segment.Connect(existing)
			segment.End = existing.Start
			break
		}
	}

	if !gen.Clip.Contains(segment.Start) && !gen.Clip.Contains(segment.End) {
		// skip if out of the screen
		return nil
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

	gen.segments = append(gen.segments, segment)
	gen.grid.Insert(segment)

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

		Center: start.Add(end).Mulf(0.5),
		Radius: start.DistanceTo(end) / 2,
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
func lineIntersect(p1, p2, q1, q2 Vec) bool {
	r := p2.Sub(p1)
	s := q2.Sub(q1)
	denom := cross(r, s)

	if denom == 0 {
		// Lines are parallel
		return false
	}

	uNumerator := cross(q1.Sub(p1), r)
	tNumerator := cross(q1.Sub(p1), s)

	u := uNumerator / denom
	t := tNumerator / denom

	// Check if t and u are within (0, 1) for segment-segment intersection
	return t > 0 && t < 1 && u > 0 && u < 1
}

func noiseToImage(noise *fastnoiselite.FastNoiseLite, width, height int, tr ebiten.GeoM) *ebiten.Image {
	pixels := make([]uint8, width*height*4)

	var pos int
	for y := range height {
		for x := range width {
			trX, trY := tr.Apply(float64(x), float64(y))

			noiseValue := noiseValueAt(noise, Vec{X: trX, Y: trY})

			pxValue := uint8(noiseValue * 0xff)

			pixels[pos+0] = 0
			pixels[pos+1] = 0
			pixels[pos+2] = 0
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
	X uint16
	Y uint16
}

type Grid struct {
	cells map[cellId]*GridCell
}

func (g *Grid) cellIdOf(vec Vec) cellId {
	return cellId{
		X: uint16(max(0, vec.X/50)),
		Y: uint16(max(0, vec.Y/50)),
	}
}

func (g *Grid) getGridCell(id cellId) *GridCell {
	cell := g.cells[id]
	if cell == nil {
		if g.cells == nil {
			g.cells = make(map[cellId]*GridCell)
		}

		cell = &GridCell{}
		g.cells[id] = cell
	}

	return cell
}

func (g *Grid) cellsOf(bbox Rect) iter.Seq[*GridCell] {
	minId := g.cellIdOf(bbox.Min)
	maxId := g.cellIdOf(bbox.Max)

	return func(yield func(*GridCell) bool) {
		for y := minId.Y; y <= maxId.Y; y++ {
			for x := minId.X; x <= maxId.X; x++ {
				if !yield(g.getGridCell(cellId{X: x, Y: y})) {
					return
				}
			}
		}
	}

}

func (g *Grid) Insert(segment *Segment) {
	bbox := bboxOf([]Vec{segment.Start, segment.End})
	for cell := range g.cellsOf(bbox) {
		cell.Segments = append(cell.Segments, segment)
	}
}

func (g *Grid) Candidates(query *Segment) iter.Seq[*Segment] {
	bbox := bboxOf([]Vec{query.Start, query.End})

	return func(yield func(*Segment) bool) {
		seen := make(map[*Segment]struct{})

		for cell := range g.cellsOf(bbox) {
			for _, segment := range cell.Segments {
				_, dup := seen[segment]

				if segment != query && !dup {
					seen[segment] = struct{}{}

					if !yield(segment) {
						return
					}
				}
			}
		}

	}
}
