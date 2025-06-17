package main

import (
	"github.com/hajimehoshi/bitmapfont"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/quasilyte/gmath"
	"golang.org/x/image/font"
	"image/color"
	"runtime"
	"sync/atomic"
	"time"
)

func posOf(vec gmath.Vec) gmath.Pos {
	return gmath.MakePos(vec)
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

func StrokePath(target *ebiten.Image, path vector.Path, tr ebiten.GeoM, color color.Color, vop *vector.StrokeOptions) {
	vertices, indices := path.AppendVerticesAndIndicesForStroke(nil, nil, vop)

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

type Promise[T any] struct {
	result  *atomic.Pointer[T]
	started bool
}

func AsyncTask[T any](task func() T) Promise[T] {
	ptr := &atomic.Pointer[T]{}

	// spawn go-routine with task
	go func() {
		result := task()
		ptr.Store(&result)
	}()

	return Promise[T]{started: true, result: ptr}
}

func (p Promise[T]) Get() *T {
	if p.result == nil {
		return nil
	}

	return p.result.Load()
}

func (p Promise[T]) Waiting() bool {
	return p.started && p.Get() == nil
}

func TransformScalar(tr ebiten.GeoM, value float64) float64 {
	x, y := tr.Apply(value, 0.0)
	return gmath.Vec{X: x, Y: y}.Len()
}

func TransformVec(tr ebiten.GeoM, value gmath.Vec) gmath.Vec {
	x, y := tr.Apply(value.X, value.Y)
	return gmath.Vec{X: x, Y: y}
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

func MeasureText(face font.Face, text string) gmath.Vec {
	bounds, _ := font.BoundString(bitmapfont.Gothic12r, text)

	size := bounds.Max.Sub(bounds.Min)
	width := size.X.Ceil()
	height := size.Y.Ceil()

	return gmath.Vec{X: float64(width), Y: float64(height)}
}

func wasmWait() {
	if runtime.GOARCH == "wasm" {
		time.Sleep(16 * time.Millisecond)
	}
}
