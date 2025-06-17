package main

import (
	"fmt"
	"github.com/hajimehoshi/bitmapfont"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	. "github.com/quasilyte/gmath"
	"math"
	"math/rand/v2"
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

type GridIndex struct {
	grid      Grid
	Remaining Set[*Segment]
}

func NewGridIndex(grid Grid) GridIndex {
	pg := GridIndex{grid: grid}

	for _, cell := range grid.cells {
		for _, segment := range cell.Segments {
			if segment.Type != StreetTypeLocal {
				continue
			}

			pg.Remaining.Insert(segment)
		}
	}

	return pg
}

func (idx *GridIndex) PopOne() *Segment {
	segment, _ := idx.Remaining.PopOne()
	return segment
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
		for _, segment := range cell.Segments {
			if idx.Remaining.Has(segment) && query.DistanceTo(segment) <= distThreshold {
				// add segment to the result
				result = append(result, segment)

				// remove segment from the index
				idx.Remaining.Remove(segment)
			}
		}
	}

	return result
}

func VillagesOf(rng *rand.Rand, grid Grid, segments []*Segment) []*Village {
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
				Id:   len(villages) + 1,
				Name: pop(&names),
				Hull: hull,
				BBox: bboxOf(hull),

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

func MarkVillage(target *ebiten.Image, tr ebiten.GeoM, village *Village) {
	trInv := tr
	trInv.Invert()

	hull := village.Hull

	var path vector.Path

	// start with last point
	n := len(hull) - 1
	path.MoveTo(float32(hull[n].X), float32(hull[n].Y))

	for _, point := range hull {
		path.LineTo(float32(point.X), float32(point.Y))
	}

	FillPath(target, path, tr, rgbaOf(0x83838320))

	bounds := MeasureText(bitmapfont.Gothic12r, village.Name).Mulf(0.5)
	_ = bounds

	// paint the name of the village
	center := TransformVec(tr, village.BBox.Center())

	{
		var op ebiten.DrawImageOptions
		op.GeoM.Scale(2.0, 2.0)
		op.GeoM.Translate(center.X, center.Y)
		op.ColorScale.ScaleWithColor(rgbaOf(0xa05e5eff))
		text.DrawWithOptions(target, village.Name, bitmapfont.Gothic12r, &op)
	}

	{
		var op ebiten.DrawImageOptions
		op.GeoM.Translate(center.X-4, center.Y+16.0)
		op.ColorScale.ScaleWithColor(rgbaOf(0xa05e5eff))
		t := fmt.Sprintf("Population: %d", village.PopulationCount)
		text.DrawWithOptions(target, t, bitmapfont.Gothic12r, &op)
	}
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
