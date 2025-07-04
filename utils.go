package main

import (
	"github.com/hajimehoshi/ebiten/v2"
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

var spVertices []ebiten.Vertex
var spIndices []uint16

func StrokePath(target *ebiten.Image, path vector.Path, toScreen ebiten.GeoM, color color.Color, vop *vector.StrokeOptions) {
	toWorld := toScreen
	toWorld.Invert()

	vop.Width = float32(TransformScalar(toWorld, float64(vop.Width)))

	spVertices, spIndices = path.AppendVerticesAndIndicesForStroke(spVertices[:0], spIndices[:0], vop)

	for idx := range spVertices {
		x, y := toScreen.Apply(float64(spVertices[idx].DstX), float64(spVertices[idx].DstY))
		spVertices[idx].DstX = float32(x)
		spVertices[idx].DstY = float32(y)
	}

	ApplyColorToVertices(spVertices, color)

	top := &ebiten.DrawTrianglesOptions{}
	top.AntiAlias = true

	target.DrawTriangles(spVertices, spIndices, whiteImage, top)
}

var fpVertices []ebiten.Vertex
var fpIndices []uint16

func FillPath(target *ebiten.Image, path vector.Path, tr ebiten.GeoM, color color.Color) {
	fpVertices, fpIndices = path.AppendVerticesAndIndicesForFilling(fpVertices[:0], fpIndices[:0])

	fpVertices = TransformVertices(tr, fpVertices, fpVertices[:0])
	ApplyColorToVertices(fpVertices, color)

	top := &ebiten.DrawTrianglesOptions{}
	top.AntiAlias = true

	target.DrawTriangles(fpVertices, fpIndices, whiteImage, top)
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
	width, height := text.Measure(t, face, 2*face.Metrics().XHeight)
	return Vec{X: width, Y: height}
}

func MaxOf[T any](values iter.Seq[T], scoreOf func(value T) float64) (T, float64, bool) {
	var bestScore = math.Inf(-1)
	var bestValue T
	var ok bool

	for value := range values {
		score := scoreOf(value)
		if score > bestScore {
			bestScore = score
			bestValue = value
			ok = true
		}
	}

	return bestValue, bestScore, ok
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

func vecSplat(val float64) Vec {
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

func TransformVertices(tr ebiten.GeoM, vertices []ebiten.Vertex, target []ebiten.Vertex) []ebiten.Vertex {
	for _, vertex := range vertices {
		x, y := tr.Apply(float64(vertex.DstX), float64(vertex.DstY))
		vertex.DstX, vertex.DstY = float32(x), float32(y)
		target = append(target, vertex)
	}

	return target
}

func directionTo(a, b Vec) Vec {
	return b.Sub(a).Normalized()
}

func iff[T any](cond bool, a, b T) T {
	if cond {
		return a
	} else {
		return b
	}
}
