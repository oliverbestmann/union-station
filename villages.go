package main

import (
	"github.com/hajimehoshi/bitmapfont"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	. "github.com/quasilyte/gmath"
	"math"
	"math/rand/v2"
	"time"
)

type Village struct {
	Id int

	// name of the village
	Name string

	// convex hull of the village
	Hull []Vec

	// Bounding box of the village
	BBox Rect
}

func VillagesOf(rng *rand.Rand, segments []*Segment) []Village {
	names := Shuffled(rng, names)

	var remaining []Vec

	for _, segment := range segments {
		if segment.Type != StreetTypeLocal {
			continue
		}

		remaining = append(remaining, segment.Start, segment.End)
	}

	var villages []Village

	for len(remaining) > 1 {
		// get a point from the remaining points, this starts the next village
		cluster := []Vec{pop(&remaining)}

		// now grow the village
		for idx := 0; idx < len(cluster) && len(remaining) > 0; idx++ {
			// now partition the remaining points for near/far to the village
			near, far := partition(remaining, cluster[idx], 100.0, nil, remaining[:0])

			// add near points to the current village
			cluster = append(cluster, near...)

			// and only keep the far points
			remaining = far
		}

		hull := ConvexHull(cluster)

		// only call it a village if we have some actual points
		if len(cluster) > 32 && len(hull) >= 3 {
			villages = append(villages, Village{
				Id:   len(villages) + 1,
				Name: pop(&names),
				Hull: hull,
				BBox: bboxOf(cluster),
			})
		}

		if len(remaining)%1000 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	return villages
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

func centerOf(vecs []Vec) Vec {
	center := vecs[0]
	for _, vec := range vecs {
		center = center.Add(vec)
	}

	return center.Divf(float64(len(vecs)))
}

func partition(haystack []Vec, needle Vec, distThreshold float64, near []Vec, far []Vec) ([]Vec, []Vec) {
	distThreshold *= distThreshold

	for _, point := range haystack {
		distSqr := needle.DistanceSquaredTo(point)
		if distSqr <= distThreshold {
			near = append(near, point)
		} else {
			far = append(far, point)
		}
	}

	return near, far
}

func MarkVillage(target *ebiten.Image, tr ebiten.GeoM, village Village) {
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

	// paint the name of the village
	center := TransformVec(tr, village.BBox.Center())

	var op ebiten.DrawImageOptions
	op.GeoM.Scale(2.0, 2.0)
	op.GeoM.Translate(-bounds.X, bounds.Y)
	op.GeoM.Translate(center.X, center.Y)
	op.ColorScale.ScaleWithColor(rgbaOf(0xa05e5eff))
	text.DrawWithOptions(target, village.Name, bitmapfont.Gothic12r, &op)
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
