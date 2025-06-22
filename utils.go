package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/oliverbestmann/union-station/assets"
	. "github.com/quasilyte/gmath"
	"image/color"
	"iter"
	"math"
	"sync/atomic"
)

var Font = assets.Font()

var Font12 = &text.GoTextFace{
	Source: Font,
	Size:   12.0,
}

var Font16 = &text.GoTextFace{
	Source: Font,
	Size:   16.0,
}

var Font24 = &text.GoTextFace{
	Source: Font,
	Size:   24.0,
}

var Font64 = &text.GoTextFace{
	Source: Font,
	Size:   64.0,
}

func pop[T any](values *[]T) T {
	n := len(*values)
	if n == 0 {
		panic("slice is empty")
	}

	value := (*values)[n-1]
	*values = (*values)[:n-1]

	return value
}

func StrokePath(target *ebiten.Image, path vector.Path, toScreen ebiten.GeoM, color color.Color, vop *vector.StrokeOptions) {
	toWorld := toScreen
	toWorld.Invert()

	vop.Width = float32(TransformScalar(toWorld, float64(vop.Width)))

	vertices, indices := path.AppendVerticesAndIndicesForStroke(nil, nil, vop)

	for idx := range vertices {
		x, y := toScreen.Apply(float64(vertices[idx].DstX), float64(vertices[idx].DstY))
		vertices[idx].DstX = float32(x)
		vertices[idx].DstY = float32(y)
	}

	top := &colorm.DrawTrianglesOptions{}
	top.AntiAlias = true

	var c colorm.ColorM
	c.ScaleWithColor(color)

	colorm.DrawTriangles(target, vertices, indices, whiteImage, c, top)
}

func FillPath(target *ebiten.Image, path vector.Path, tr ebiten.GeoM, color color.Color) {
	vertices, indices := path.AppendVerticesAndIndicesForFilling(nil, nil)

	for idx := range vertices {
		x, y := tr.Apply(float64(vertices[idx].DstX), float64(vertices[idx].DstY))
		vertices[idx].DstX = float32(x)
		vertices[idx].DstY = float32(y)
	}

	top := &colorm.DrawTrianglesOptions{}
	top.AntiAlias = true

	var c colorm.ColorM
	c.ScaleWithColor(color)

	colorm.DrawTriangles(target, vertices, indices, whiteImage, c, top)
}

type Promise[T any, P any] struct {
	result   *atomic.Pointer[T]
	progress *atomic.Pointer[P]
	seen     *atomic.Bool
	started  bool
}

func AsyncTask[T any, P any](task func(yield func(P)) T) Promise[T, P] {
	result := &atomic.Pointer[T]{}
	progress := &atomic.Pointer[P]{}

	// spawn go-routine with task
	go func() {
		value := task(func(p P) {
			progress.Store(&p)
		})

		result.Store(&value)
	}()

	return Promise[T, P]{
		started:  true,
		result:   result,
		progress: progress,
		seen:     &atomic.Bool{},
	}
}

func (p Promise[T, P]) Get() *T {
	if p.result == nil {
		return nil
	}

	return p.result.Load()
}

func (p Promise[T, P]) GetOnce() *T {
	if p.result == nil || p.seen.Load() {
		return nil
	}

	value := p.result.Load()
	if value != nil && !p.seen.CompareAndSwap(false, true) {
		return nil
	}

	return value
}

func (p Promise[T, P]) Status() *P {
	if p.progress == nil || p.Get() != nil {
		return nil
	}

	return p.progress.Load()
}

func (p Promise[T, P]) Started() bool {
	return p.started
}

func (p Promise[T, P]) Waiting() bool {
	return p.started && p.Get() == nil
}

func TransformScalar(tr ebiten.GeoM, value float64) float64 {
	x, y := tr.Apply(value, 0.0)
	return Vec{X: x, Y: y}.Len()
}

func TransformVec(tr ebiten.GeoM, value Vec) Vec {
	x, y := tr.Apply(value.X, value.Y)
	return Vec{X: x, Y: y}
}

func rgbaOf(rgba uint32) color.NRGBA {
	return color.NRGBA{
		R: uint8((rgba >> 24) & 0xff),
		G: uint8((rgba >> 16) & 0xff),
		B: uint8((rgba >> 8) & 0xff),
		A: uint8((rgba >> 0) & 0xff),
	}
}

func MeasureText(face text.Face, t string) Vec {
	width, height := text.Measure(t, face, 0)
	return Vec{X: width, Y: height}
}

func MaxOf[T any](values iter.Seq[T], scoreOf func(value T) float64) T {
	var bestScore = math.Inf(-1)
	var bestValue T

	for value := range values {
		score := scoreOf(value)
		if score > bestScore {
			bestScore = score
			bestValue = value
		}
	}

	return bestValue
}

func Repeat[T any](n int, fn func() T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for range n {
			if !yield(fn()) {
				return
			}
		}
	}
}

func splatVec(val float64) Vec {
	return Vec{X: val, Y: val}
}

func imageSizeOf(image *ebiten.Image) Vec {
	return Vec{
		X: float64(image.Bounds().Dx()),
		Y: float64(image.Bounds().Dy()),
	}
}

func imageHeight(img *ebiten.Image) int {
	return img.Bounds().Dy()
}

func imageWidth(img *ebiten.Image) int {
	return img.Bounds().Dx()
}

func DrawText(target *ebiten.Image, msg string, face text.Face, pos Vec, color color.Color, primaryAlign, secondaryAlign text.Align) {
	if color == nil {
		color = DebugColor
	}

	op := &text.DrawOptions{}
	op.GeoM.Translate(pos.X, pos.Y)
	op.PrimaryAlign = primaryAlign
	op.SecondaryAlign = secondaryAlign
	op.ColorScale.ScaleWithColor(color)
	op.LineSpacing = face.Metrics().XHeight * 2.0
	text.Draw(target, msg, face, op)
}

func DrawTextCenter(target *ebiten.Image, msg string, face text.Face, pos Vec, color color.Color) {
	DrawText(target, msg, face, pos, color, text.AlignCenter, text.AlignCenter)
}

func DrawTextLeft(target *ebiten.Image, msg string, face text.Face, pos Vec, color color.Color) {
	DrawText(target, msg, face, pos, color, text.AlignStart, text.AlignStart)
}

func DrawTextRight(target *ebiten.Image, msg string, face text.Face, pos Vec, color color.Color) {
	DrawText(target, msg, face, pos, color, text.AlignEnd, text.AlignStart)
}

func TransformVertices(tr ebiten.GeoM, vertices []ebiten.Vertex, reuse *[]ebiten.Vertex) []ebiten.Vertex {
	var trVertices []ebiten.Vertex

	if reuse != nil {
		// transform vertices to screen
		trVertices = (*reuse)[:0]
	}

	for _, vertex := range vertices {
		x, y := tr.Apply(float64(vertex.DstX), float64(vertex.DstY))
		vertex.DstX, vertex.DstY = float32(x), float32(y)
		trVertices = append(trVertices, vertex)
	}

	if reuse != nil {
		*reuse = trVertices[:0]
	}

	return trVertices
}

func directionTo(a, b Vec) Vec {
	return b.Sub(a).Normalized()
}
