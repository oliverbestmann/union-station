package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	. "github.com/quasilyte/gmath"
	"image/color"
	"iter"
	"maps"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"unicode"
)

type Village struct {
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

	FunFact string
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
			if idx.Remaining.Has(segment) && query.DistanceToOther(segment.Line) <= distThreshold {
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
	funfacts := Shuffled(rng, funfacts)

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
			villageId := len(villages) + 1

			name := names[villageId%len(names)]
			funfact := strings.ReplaceAll(funfacts[villageId%len(funfacts)], "$NAME", name)

			villages = append(villages, &Village{
				Name:            name,
				Hull:            hull,
				BBox:            bboxOf(hull),
				Segments:        cluster,
				FunFact:         "Did you know: " + funfact,
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
	StrokeColor color.Color
	FillColor   color.Color
}

func DrawVillageBounds(target *ebiten.Image, village *Village, opts DrawVillageBoundsOptions) {
	path := pathOf(village.Hull, true)

	_, _, _, fillAlpha := opts.FillColor.RGBA()
	if fillAlpha > 0 {
		FillPath(target, path, opts.ToScreen, opts.FillColor)
	}

	if opts.StrokeWidth > 0 {
		StrokePath(target, path, opts.ToScreen, opts.StrokeColor, &vector.StrokeOptions{
			Width:    float32(opts.StrokeWidth),
			LineJoin: vector.LineJoinRound,
			LineCap:  vector.LineCapSquare,
		})
	}
}

func (g *Game) drawVillageTooltip(target *ebiten.Image, pos Vec, village *Village) {
	dialog := Dialog{
		Padding: vecSplat(16),
		Texts: []Text{
			{
				Text:  village.Name,
				Face:  Font24,
				Color: DarkTextColor,
			},
			{
				Text:  fmt.Sprintf("Population: %d", village.PopulationCount),
				Face:  Font16,
				Color: DarkTextColor,
			},
		},
	}

	var noSpace bool
	for line := range textwrap(village.FunFact) {
		dialog.Texts = append(dialog.Texts, Text{
			Text:   line,
			Face:   Font16,
			Color:  DarkTextColor,
			Offset: Vec{Y: iff(noSpace, 0.0, 8)},
		})

		noSpace = true
	}

	size, _ := dialog.Measure()

	if int(pos.X) > imageWidth(target)*3/4 {
		// anchor tooltip top right corner of the dialog
		pos = pos.Add(Vec{X: -size.X - 16, Y: 24})
	} else {
		// anchor tooltip at the top left corner
		pos = pos.Add(Vec{X: 16, Y: 24})
	}

	if pos.Y+size.Y > imageSizeOf(target).Y {
		pos.Y -= size.Y
	}

	dialog.DrawAt(target, pos)
}

func textwrap(text string) iter.Seq[string] {
	return func(yield func(string) bool) {
		var chars int
		var lineStart int

		for idx, ch := range text {
			chars += 1

			if chars > 40 && unicode.IsSpace(ch) {
				// break here
				yield(strings.TrimSpace(text[lineStart:idx]))

				chars = 0
				lineStart = idx
			}
		}

		if chars > 0 {
			yield(strings.TrimSpace(text[lineStart:]))
		}
	}
}

func DrawWindow(target *ebiten.Image, pos Vec, size Vec) {
	posShadow := pos.Add(vecSplat(4))
	DrawRoundRect(target, posShadow, size, ShadowColor)

	DrawRoundRect(target, pos, size, TooltipColor)
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

var funfacts = []string{
	"$NAME is known for having one of the oldest postboxes still in use in Britain.",
	"The only shop in $NAME doubles as the post office and community centre.",
	"$NAME's railway station was closed during the Beeching cuts of the 1960s.",
	"You can find a 12th-century church at the heart of $NAME.",
	"$NAME is famous for its annual scarecrow festival.",
	"The local pub in $NAME is said to be haunted by a Victorian railway worker.",
	"$NAME once had a stationmaster who commuted by horse from a neighboring village.",
	"Trains still whistle when passing through the disused station of $NAME.",
	"$NAME has no streetlights, making it perfect for stargazing.",
	"In $NAME, the village green is used for sheep grazing during winter.",
	"The telephone box in $NAME has been turned into a miniature library.",
	"$NAME has a centuries-old well still used during droughts.",
	"A famous British poet once stayed in a cottage in $NAME for inspiration.",
	"$NAME’s name derives from Old English and means 'hill of the wolves'.",
	"The original signal box from $NAME’s station now sits in a railway museum.",
	"Every house in $NAME has a thatched roof, due to heritage protection.",
	"$NAME was once used as a filming location for a BBC period drama.",
	"The station at $NAME had only one platform and a cattle ramp.",
	"$NAME is connected to the national footpath network via the Monarch's Way.",
	"Local legend claims a Roman treasure is buried beneath $NAME’s village green.",
	"$NAME's village church still rings bells using ropes pulled manually.",
	"$NAME hosts a traditional Maypole dance every spring.",
	"The old railway viaduct near $NAME is now a popular walking trail.",
	"In $NAME, residents still hold an annual goose fair on the village green.",
	"The train tunnel near $NAME is said to be the longest hand-dug tunnel in the region.",
	"$NAME’s war memorial lists more names than the current population.",
	"$NAME has a tradition of lighting a hilltop beacon on national holidays.",
	"A historic steam train passes by $NAME on special occasions.",
	"$NAME has never had a supermarket within 10 miles.",
	"A preserved station clock from $NAME keeps time at the National Railway Museum.",
	"$NAME’s churchyard has gravestones dating back to the 1500s.",
	"Only three surnames dominate the residents of $NAME.",
	"$NAME's primary school has fewer than 20 pupils.",
	"The village of $NAME had a blacksmith shop that operated until 1987.",
	"$NAME’s railway halt was once the shortest platform in the county.",
	"Each house in $NAME is required by law to use traditional stone for repairs.",
	"A Victorian railway bridge in $NAME is now used for sheep crossings.",
	"$NAME’s pub brews its own ale named after the village.",
	"The bell tower in $NAME leans by 4 degrees but is structurally sound.",
	"$NAME has been continuously inhabited since Saxon times.",
	"The railway line that passed through $NAME was known as the 'milk run'.",
	"You can walk from $NAME to a neighboring village entirely via public footpaths.",
	"$NAME's local folklore includes a ghost train that appears once a year.",
	"Every building in $NAME is a listed historical structure.",
	"The village of $NAME holds the record for the lowest recorded UK temperature.",
	"$NAME has a heritage railway society that maintains the old station building.",
	"The church in $NAME has a yew tree over 1,000 years old.",
	"$NAME used to export cheese via a dedicated railway siding.",
	"The villagers of $NAME once built their own footbridge over a stream in a weekend.",
	"$NAME’s village pond is home to a species of rare native newt.",
	"The last train to stop at $NAME carried only the stationmaster’s bicycle.",
	"$NAME is part of a conservation area with strict building rules.",
	"The local train station in $NAME was only accessible by footpath.",
	"$NAME once had a windmill, now only the base remains.",
	"$NAME has a tradition of wassailing in its apple orchards each winter.",
	"The railway platform in $NAME was once used as a theatre stage for summer plays.",
	"$NAME’s main road is still made of cobblestones.",
	"The sheep in $NAME are known to block roads during lambing season.",
	"$NAME has its own microclimate due to its valley position.",
	"The village shop in $NAME is run entirely by volunteers.",
	"$NAME has an annual duck race down the local stream.",
	"One of the oldest wooden footbridges in England can be found in $NAME.",
	"$NAME’s railway sidings were once used for royal mail distribution.",
	"The bus to $NAME runs only twice a week.",
	"A disused train carriage in $NAME has been converted into a holiday rental.",
	"$NAME has no traffic lights or roundabouts within a 10-mile radius.",
	"$NAME’s annual flower show includes categories like ‘best marrow’ and ‘oddest vegetable’.",
	"A section of Roman road still runs near $NAME’s village boundary.",
	"$NAME’s village sign was carved from local oak by a resident woodworker.",
	"The name $NAME appears in the Domesday Book.",
	"$NAME’s railway station was used for livestock loading until the 1970s.",
	"$NAME holds an unofficial record for most dogs per household.",
	"Children in $NAME used to be taught in the church vestry before the school was built.",
	"The railway trackbed near $NAME is now part of a national cycle route.",
	"$NAME’s post office was once a stagecoach stop.",
	"During WWII, $NAME’s railway line was used for troop movements.",
	"The phone box in $NAME is now a defibrillator station.",
	"A steam rally is held on the outskirts of $NAME every summer.",
	"$NAME was once famous for its cherry orchards, now all but gone.",
	"The local stream in $NAME used to power a grain mill.",
	"The railway embankment near $NAME is home to a rare orchid species.",
	"A railway accident near $NAME in the 1800s led to changes in safety standards.",
	"$NAME has a tradition of blessing the fields every spring.",
	"The thatched roofs in $NAME must be re-done every 30 years by regulation.",
	"$NAME was once twinned with a French village that no longer exists.",
	"The old goods yard in $NAME has been turned into a community garden.",
	"$NAME’s railway signal still works and is used ceremonially each year.",
	"The grave of a famous inventor lies in $NAME’s churchyard.",
	"$NAME is one of the only villages with an original Tudor barn still in use.",
	"Trains through $NAME used to stop only on market days.",
	"$NAME’s entire street plan hasn’t changed since 1750.",
	"An archaeological dig in $NAME uncovered Bronze Age tools.",
	"$NAME has its own flag, designed by local schoolchildren.",
	"The village of $NAME appeared on a UK postage stamp in the 1990s.",
	"A traveling fair has stopped in $NAME every June for over 100 years.",
	"The old train turntable in $NAME is now a roundabout for pedestrians.",
	"A mystery manuscript was found hidden in the rafters of $NAME’s church.",
	"Every Tuesday, the people of $NAME celebrate 'Moss Appreciation Day' with a parade of wheelbarrows.",
	"$NAME once tried to declare independence from the UK over a dispute about scone recipes.",
	"The train station in $NAME only has one bench, but it’s officially a heritage site.",
	"In $NAME, it’s illegal to own more than three teapots unless you're the mayor.",
	"The local train in $NAME is powered entirely by fermented beetroot juice.",
	"$NAME's annual 'Invisible Dog Show' attracts imaginary pets from all over the country.",
	"$NAME claims to have the world’s quietest bell tower—it’s completely silent.",
	"Every house in $NAME has at least one painting of a sheep, by law.",
	"The $NAME train whistle was once voted ‘most soothing’ in a national poll.",
	"Every third Thursday, $NAME residents wear only tweed to honor 'Tweed Day'.",
	"$NAME has a mysterious postbox that sends letters into the future.",
	"The ducks in $NAME are known for crossing the road in synchronized formations.",
	"There’s a pub in $NAME that only serves drinks named after clouds.",
	"$NAME has a train platform that only appears during leap years.",
	"Once a year, $NAME holds a silent disco for tractors.",
	"$NAME’s village green is shaped like a perfect question mark.",
	"All the roads in $NAME are subtly scented with lavender.",
	"$NAME’s train conductor insists on reciting haikus before each departure.",
	"In $NAME, the local bakery claims their sourdough can tell fortunes.",
	"$NAME is twinned with the Moon. No one knows why.",
	"The church in $NAME rings its bells backward during full moons.",
	"$NAME has a museum dedicated entirely to left socks found on trains.",
	"There’s a local myth that the sheep in $NAME can predict train delays.",
	"The train to $NAME only stops if someone waves with their left hand.",
	"In $NAME, every street is named after a different type of cheese.",
	"The mayor of $NAME was elected after winning a pie-eating contest.",
	"A hedge maze in $NAME has no exit and the locals like it that way.",
	"$NAME’s official flower is a dandelion wearing a top hat (in sculpture form).",
	"$NAME once hosted a chess tournament played entirely on picnic blankets.",
	"$NAME’s annual train-themed opera is performed entirely by owls.",
	"In $NAME, it’s traditional to tap three times on a lamppost before boarding a train.",
	"The train station in $NAME was built upside down and no one has fixed it.",
	"Local folklore claims $NAME was founded by a runaway steam engine.",
	"The people of $NAME hold a monthly meeting to decide the flavor of air.",
	"The $NAME train always runs late—by artistic design.",
	"$NAME has the narrowest alley in Britain, used only for snail racing.",
	"There is a toad in $NAME who has been honorary mayor since 1872.",
	"Train announcements in $NAME are sung by a retired opera singer.",
	"The village sign of $NAME is upside down and no one knows why.",
	"Every resident in $NAME is required to own a rubber duck.",
	"All the sheep in $NAME wear tiny scarves during winter.",
	"A train once stopped in $NAME for five years due to a nap.",
	"Every house in $NAME has a room dedicated to jam.",
	"The $NAME signal box is operated by a well-trained squirrel.",
	"The local river in $NAME flows in reverse on Wednesdays.",
	"Each bench in $NAME’s park plays a different Beatles song when sat on.",
	"$NAME’s primary export is novelty moustaches.",
	"The trains in $NAME are pulled by enthusiastic hobbyists on bicycles.",
	"At night, $NAME’s streetlamps glow a gentle mauve for ‘mood lighting’.",
	"$NAME holds an annual snail marathon with loud cheering crowds.",
	"The village clocktower in $NAME runs on a diet of biscuits.",
	"A tunnel in $NAME echoes compliments instead of sounds.",
	"In $NAME, the stationmaster wears a monocle and cape by tradition.",
	"Every pigeon in $NAME has a registered name and address.",
	"The local legend says the hills around $NAME are actually sleeping giants.",
	"$NAME has a train-themed tea shop where the scones arrive on model trains.",
	"$NAME celebrates the equinox by balancing eggs on the vicar’s head.",
	"There’s a scarecrow in $NAME who receives more mail than the mayor.",
	"The bus shelter in $NAME is a legally protected ancient monument.",
	"In $NAME, the telephone boxes have been turned into mini libraries with biscuits.",
	"The rail line to $NAME has more curves than any track in the country—on purpose.",
	"$NAME’s high street features a shop that only sells socks with pineapples.",
	"The village pond in $NAME is shaped like a badger.",
	"The annual $NAME trainspotter's ball involves dancing with actual train tickets.",
	"The village of $NAME has a law that mandates singing when crossing bridges.",
	"Every cloud over $NAME is tracked and given a friendly name.",
	"Train drivers in $NAME wear special gloves hand-knitted by the council.",
	"The local pub in $NAME has a portrait of every customer—painted weekly.",
	"$NAME’s railway line hums in B-flat during fog.",
	"The cows in $NAME are rumored to moo in regional accents.",
	"$NAME has the only train station with a slide instead of stairs.",
	"There’s a bench in $NAME dedicated to a hedgehog named Charles.",
	"During summer, the train to $NAME is decorated like an ice cream sundae.",
	"$NAME was once renamed briefly to 'Trainville' as a marketing stunt.",
	"The school in $NAME is shaped like a giant open book.",
	"There’s a tradition in $NAME to greet trains with a curtsy, no matter the gender.",
	"The train timetable in $NAME is illustrated entirely with watercolor art.",
	"$NAME once held the record for most simultaneous kettles boiled.",
	"The signal lights at $NAME’s train crossing are replaced with disco balls during festivals.",
	"The train station in $NAME has its own tea sommelier.",
	"$NAME’s residents believe badgers bring good rail fortune.",
	"Each garden in $NAME contains at least one garden gnome in a railway uniform.",
	"The air in $NAME smells of biscuits every third Friday.",
	"$NAME’s town motto is 'We were on time once, and we liked it.'",
	"At $NAME station, the waiting room is filled with bean bags and wind chimes.",
	"$NAME locals use spoons as weather indicators.",
	"Train horns in $NAME must be tuned to play part of 'God Save the Queen.'",
	"$NAME’s annual village play is based on the timetable of the 4:17 service.",
	"The pond in $NAME reflects only happy faces on Sundays.",
	"A goose once delayed every train to $NAME by five hours—it’s now a legend.",
	"Every lamppost in $NAME is named and regularly hugged.",
	"$NAME’s village green hosts competitive cloud staring leagues.",
	"The train to $NAME is sometimes mistaken for a carnival ride.",
	"Local artists in $NAME paint a new mural on the train every week.",
	"$NAME’s water tower whistles when it’s full.",
	"In $NAME, it’s considered lucky to wave at passing trains with a teacup.",
	"Every doorbell in $NAME rings the sound of a passing train.",
	"There is a law in $NAME requiring all announcements to rhyme.",
	"$NAME once declared itself the 'Unofficial Capital of Whistling.'",
	"The train tunnel to $NAME features glowworms as natural lighting.",
	"$NAME’s bus service is just a retired train in disguise.",
	"In $NAME, train tickets come with a complimentary riddle.",
	"The signalman of $NAME writes poetry between shifts and publishes it on train receipts.",
}
