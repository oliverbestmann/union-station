package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/vector"
	. "github.com/quasilyte/gmath"
	"image/color"
	"maps"
	"math"
	"math/rand/v2"
	"slices"
)

type Village struct {
	Id int

	// name of the village
	Name string

	// convex hull of the village
	Hull []Vec

	// all segments that belong to this village
	Segments []*Segment

	// Bounding box of the village
	BBox Rect

	/// Number of people living this village
	PopulationCount int
}

func (v *Village) Contains(pos Vec) bool {
	if !v.BBox.Contains(pos) {
		return false
	}

	return PointInConvexHull(v.Hull, pos)
}

type GridIndex struct {
	grid             Grid[*Segment]
	Remaining        Set[*Segment]
	remainingOrdered []*Segment
}

func NewGridIndex(grid Grid[*Segment]) GridIndex {
	pg := GridIndex{grid: grid}

	// need to walk the grid in deterministic order
	keysSorted := slices.SortedFunc(maps.Keys(grid.cells), func(a, b cellId) int {
		if a.X != b.X {
			// compare by x
			return int(a.X) - int(b.X)
		} else {
			// if equal, compare by y
			return int(a.Y) - int(b.Y)
		}
	})

	for _, key := range keysSorted {
		cell := grid.cells[key]

		for _, segment := range cell.Objects {
			if segment.Type != StreetTypeLocal {
				continue
			}

			pg.Remaining.Insert(segment)

			// need to keep deterministic order of segments for
			// query purposes too
			pg.remainingOrdered = append(pg.remainingOrdered, segment)
		}
	}

	return pg
}

func (idx *GridIndex) PopOne() *Segment {
	for i, value := range idx.remainingOrdered {
		if idx.Remaining.Has(value) {
			idx.Remaining.Remove(value)
			idx.remainingOrdered = idx.remainingOrdered[i+1:]
			return value
		}
	}

	return nil
}

func (idx *GridIndex) Extract(query *Segment, distThreshold float64) []*Segment {
	bbox := query.BBox()
	bbox.Min.X -= distThreshold
	bbox.Min.Y -= distThreshold
	bbox.Max.X += distThreshold
	bbox.Max.X += distThreshold

	var result []*Segment

	// query the grid for segments within that range
	for cell := range idx.grid.CellsOf(bbox, false) {
		for _, segment := range cell.Objects {
			if idx.Remaining.Has(segment) && query.DistanceTo(segment.Line) <= distThreshold {
				// add segment to the result
				result = append(result, segment)

				// remove segment from the index
				idx.Remaining.Remove(segment)
			}
		}
	}

	return result
}

func CollectVillages(rng *rand.Rand, grid Grid[*Segment]) []*Village {
	names := Shuffled(rng, names)

	index := NewGridIndex(grid)

	var villages []*Village
	var idle IdleSuspend

	for index.Remaining.Len() > 1 {
		// get a point from the remaining points, this starts the next village
		cluster := []*Segment{index.PopOne()}

		// now grow the village
		for idx := 0; idx < len(cluster) && index.Remaining.Len() > 0; idx++ {
			// get all segments near to the one we're looking at right now
			near := index.Extract(cluster[idx], 100)

			// add near points to the current village
			cluster = append(cluster, near...)

			if idx%50 == 0 {
				// maybe suspend to give the browser time to update the next frame
				idle.MaybeSuspend()
			}
		}

		pointCluster := pointsOf(cluster)
		hull := ConvexHull(pointCluster)

		// only call it a village if we have some actual points
		if len(cluster) > 32 && len(hull) >= 3 {
			villages = append(villages, &Village{
				Id:       len(villages) + 1,
				Name:     pop(&names),
				Hull:     hull,
				BBox:     bboxOf(hull),
				Segments: cluster,

				PopulationCount: populationCountOf(cluster),
			})
		}
	}

	return villages
}

func populationCountOf(segments []*Segment) int {
	var sum float64
	for _, segment := range segments {
		// we count one person for every 100m street length
		sum += segment.Length() / 100
	}

	return int(math.Ceil(sum))
}

func pointsOf(segments []*Segment) []Vec {
	vecs := make([]Vec, 0, len(segments)*2)

	for _, segment := range segments {
		vecs = append(vecs, segment.Start, segment.End)
	}

	return vecs
}

func bboxOf(vecs []Vec) Rect {
	var minX = math.MaxFloat64
	var minY = math.MaxFloat64
	var maxX, maxY float64

	for _, vec := range vecs {
		minX = min(minX, vec.X)
		maxX = max(maxX, vec.X)
		minY = min(minY, vec.Y)
		maxY = max(maxY, vec.Y)
	}

	return Rect{
		Min: Vec{X: minX, Y: minY},
		Max: Vec{X: maxX, Y: maxY},
	}
}

type DrawVillageBoundsOptions struct {
	ToScreen    ebiten.GeoM
	StrokeWidth float64
	StrokeColor color.NRGBA
	FillColor   color.NRGBA
}

func DrawVillageBounds(target *ebiten.Image, village *Village, opts DrawVillageBoundsOptions) {
	path := pathOf(village.Hull, true)

	if opts.FillColor.A > 0 {
		FillPath(target, path, opts.ToScreen, opts.FillColor)
	}

	if opts.StrokeWidth > 0 {
		StrokePath(target, path, opts.ToScreen, opts.StrokeColor, &vector.StrokeOptions{
			Width:    2,
			LineJoin: vector.LineJoinRound,
			LineCap:  vector.LineCapSquare,
		})
	}
}

func (g *Game) drawVillageTooltip(target *ebiten.Image, pos Vec, village *Village) {
	connectedText := "Not connected"
	for _, edge := range g.acceptedGraph.Edges() {
		if edge.One.Village == village || edge.Two.Village == village {
			connectedText = "Connected"
			break
		}
	}

	dialog := Dialog{
		Padding: vecSplat(16),
		Texts: []Text{
			{
				Text:  village.Name,
				Face:  Font24,
				Color: HudTextColor,
			},
			{
				Text:  fmt.Sprintf("Population: %d", village.PopulationCount),
				Face:  Font16,
				Color: HudTextColor,
			},
			{
				Text:  connectedText,
				Face:  Font16,
				Color: HudTextColor,
			},
		},
	}

	if int(pos.X) > imageWidth(target)*3/4 {
		// anchor tooltip top right corner of the dialog
		size := dialog.Size()
		pos = pos.Add(Vec{X: -size.X - 16, Y: 24})
	} else {
		// anchor tooltip at the top left corner
		pos = pos.Add(Vec{X: 16, Y: 24})
	}

	dialog.DrawAt(target, pos)
}

func DrawWindow(target *ebiten.Image, pos Vec, size Vec) {
	// shadow
	{
		pos := pos.Add(vecSplat(4))

		var cm colorm.ColorM
		cm.ScaleWithColor(ShadowColor)

		var op colorm.DrawImageOptions
		op.GeoM.Scale(size.X, size.Y)
		op.GeoM.Translate(pos.X, pos.Y)
		colorm.DrawImage(target, whiteImage, cm, &op)
	}

	// tooltip itself
	{
		var cm colorm.ColorM
		cm.ScaleWithColor(TooltipColor)

		var op colorm.DrawImageOptions
		op.GeoM.Scale(size.X, size.Y)
		op.GeoM.Translate(pos.X, pos.Y)
		colorm.DrawImage(target, whiteImage, cm, &op)
	}
}

func pathOf(points []Vec, close bool) vector.Path {
	var path vector.Path

	if len(points) < 2 {
		return path
	}

	path.MoveTo(float32(points[0].X), float32(points[0].Y))

	for _, point := range points[1:] {
		path.LineTo(float32(point.X), float32(point.Y))
	}

	if close {
		path.Close()
	}

	return path
}

//goland:noinspection ALL
var names = []string{
	"Ashcombe",
	"Thistlewick",
	"Darnley Hollow",
	"Bramblehurst",
	"Eastonmere",
	"Cragfen",
	"Wetherby Down",
	"Millbridge",
	"Gorsefield",
	"Elmbourne",
	"Haverleigh",
	"Wychcombe",
	"Bramwith",
	"Netherfold",
	"Greystone End",
	"Withercombe",
	"Aldenbrook",
	"Mistlewick",
	"Fernley Cross",
	"Oakhollow",
	"Ravensmere",
	"Foxleigh",
	"Norham St. Giles",
	"Tillinghurst",
	"Windlecombe",
	"Marlow Fen",
	"Thackworth",
	"Hollowmere",
	"Birchcombe",
	"East Peverell",
	"Hogsden",
	"Ironleigh",
	"Crowmarsh",
	"Emberwick",
	"Wrenfold",
	"Sallowby",
	"Dunthorp",
	"Maplewick",
	"Brockhurst",
	"Coldmere",
	"Stagbourne",
	"Wynthorpe",
	"Farley-under-Wold",
	"Heathbury",
	"Caxton Hollow",
	"Faircombe",
	"Woolston Edge",
	"Redgrave Moor",
	"Bexhill Hollow",
	"Cobblebury",
	"Grindleford",
	"Foxcombe Vale",
	"Holloway End",
	"Piddlestone",
	"Winmarleigh",
	"Crowleigh",
	"Tunstowe",
	"Quenby Marsh",
	"Kestrelcombe",
	"Ormsden",
	"Branthorpe",
	"Wexley Heath",
	"Hobbington",
	"Elmstead Rise",
	"Dapplemere",
	"Nethercombe",
	"Broomley End",
	"Westering Hollow",
	"Felsham Vale",
	"Oxley Dene",
	"Yarrowby",
	"Cinderbourne",
	"Applefold",
	"Beechmarsh",
	"Norleigh",
	"Thornwick",
	"Linwell Hollow",
	"Peverstone",
	"Stonethorpe",
	"Witham Vale",
	"Cherriton",
	"Grayscombe",
	"Whitlow Hill",
	"Otterby Fen",
	"Willowham",
	"Gildersby",
	"Aldermere",
	"Brockleigh",
	"Redlinch",
	"Stowbeck",
	"Fallowford",
	"East Bransley",
	"Crickmarsh",
	"Harkwell",
	"Duncombe Green",
	"Kingsmere",
	"Swandale",
	"Farthinglow",
	"Moorwick",
	"Harrowell",
}
