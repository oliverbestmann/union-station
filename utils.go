package main

import (
	"github.com/hajimehoshi/bitmapfont"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/vector"
	. "github.com/quasilyte/gmath"
	"golang.org/x/image/font"
	"image/color"
	"iter"
	"math"
	"sync/atomic"
)

var Font = bitmapfont.Gothic12r

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
	started  bool
}

func AsyncTask[T any, P any](task func(yield func(P)) T) Promise[T, P] {
	ptr := &atomic.Pointer[T]{}
	progress := &atomic.Pointer[P]{}

	// spawn go-routine with task
	go func() {
		result := task(func(p P) {
			progress.Store(&p)
		})

		ptr.Store(&result)
	}()

	return Promise[T, P]{started: true, result: ptr, progress: progress}
}

func (p Promise[T, P]) Get() *T {
	if p.result == nil {
		return nil
	}

	return p.result.Load()
}

func (p Promise[T, P]) Progress() *P {
	if p.progress == nil || p.Get() != nil {
		return nil
	}

	return p.progress.Load()
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

var DebugColor = color.RGBA{R: 0xff, B: 0xff, A: 0xff}

func MeasureText(face font.Face, text string) Vec {
	bounds, _ := font.BoundString(face, text)

	size := bounds.Max.Sub(bounds.Min)
	width := size.X.Ceil()
	height := size.Y.Ceil()

	return Vec{X: float64(width), Y: float64(height)}
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

func vecSplat(val float64) Vec {
	return Vec{X: val, Y: val}
}

func imageSizeOf(image *ebiten.Image) Vec {
	return Vec{
		X: float64(image.Bounds().Dx()),
		Y: float64(image.Bounds().Dy()),
	}
}
