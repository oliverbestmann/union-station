package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/oliverbestmann/union-station/assets"
	. "github.com/quasilyte/gmath"
	"image/color"
	"math"
	"sync"
)

func (g *Game) drawHUD(screen *ebiten.Image) {
	// hud position we start to draw at
	pos := Vec{X: imageSizeOf(screen).X - 16, Y: 16}

	g.btnSettings.Draw(screen)
	pos.X -= g.btnSettings.Size.X
	pos.X -= 16

	// buttons for the settings menu. calculate
	// the size to put a layer below
	menuContentSize := Rect{Min: vecSplat(math.Inf(1))}
	for _, button := range g.menu {
		menuContentSize.Min.X = min(menuContentSize.Min.X, button.Position.X)
		menuContentSize.Min.Y = min(menuContentSize.Min.Y, button.Position.Y)

		menuContentSize.Max.X = max(menuContentSize.Max.X, button.Position.X+button.Size.X)
		menuContentSize.Max.Y = max(menuContentSize.Max.Y, button.Position.Y+button.Size.Y)
	}

	DrawWindow(screen, menuContentSize.Min.Sub(vecSplat(16)), menuContentSize.Size().Add(vecSplat(32)))

	for _, button := range g.menu {
		button.Draw(screen)
	}

	if g.stats.CoinsTotal > 0 {
		msg := fmt.Sprintf("Budget: %d", g.stats.CoinsAvailable())
		g.hudRectangleWithIcon(screen, &pos, -1, msg, HudRectangleColor, assets.Coin())

		if g.stats.CoinsPlanned > 0 {
			// add some space between the rectangles
			pos.X -= 16

			msg := fmt.Sprintf("Planned: %d", g.stats.CoinsPlanned)
			g.hudRectangleWithIcon(screen, &pos, -1, msg, HudPlannedRectangleColor, assets.PlannedCoin())
		}

		if g.stats.StationsConnected > 0 {
			// add some space between the rectangles
			pos.X -= 16

			msg := fmt.Sprintf("Connected %d of %d", g.stats.StationsConnected, g.stats.StationsTotal)
			g.hudRectangleWithIcon(screen, &pos, -1, msg, HudRectangleColor, nil)
		}

	}

	if g.stats.Score > 0 {
		pos := Vec{X: 16, Y: 16}

		msg := fmt.Sprintf("Score: %d", g.stats.Score)
		g.hudRectangleWithIcon(screen, &pos, 1, msg, HudRectangleColor, nil)
	}
}

func (g *Game) hudRectangleWithIcon(target *ebiten.Image, pos *Vec, dir float64, msg string, rectangleColor color.Color, icon *ebiten.Image) {
	textWidth := MeasureText(Font24, msg).X

	var iconSize Vec
	if icon != nil {
		iconSize = imageSizeOf(icon)
	}

	// 16px padding, 8px gap, icon size
	rSize := Vec{X: textWidth + 8 + iconSize.X + 16*2, Y: 48}
	rPos := Vec{X: pos.X, Y: pos.Y - 8}

	if dir < 0 {
		rPos.X -= rSize.X
	}

	// draw a small shadow
	shadow := rPos.Add(vecSplat(2))
	DrawRoundRect(target, shadow, rSize, ShadowColor)

	// draw the rectangle
	DrawRoundRect(target, rPos, rSize, rectangleColor)

	// padding within the rectangle
	pos.X += 16 * dir

	if icon != nil {
		// draw the coin icon
		if dir < 0 {
			pos.X -= iconSize.X
		}

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(pos.X, pos.Y)
		target.DrawImage(icon, op)

		if dir > 0 {
			pos.X += iconSize.X
		}

		// spacing to the text
		pos.X += 8 * dir
	}

	if dir < 0 {
		pos.X -= textWidth
	}

	DrawTextLeft(target, msg, Font24, *pos, BackgroundColor)

	if dir > 0 {
		pos.X += textWidth
	}

	// add some padding within the rectangle
	pos.X += 16 * dir
}

func DrawRoundRect(target *ebiten.Image, rectanglePos Vec, rectangleSize Vec, color color.Color) {
	rrVertices, rrIndices := RoundedRectangle(rectanglePos, rectangleSize, 8)

	ApplyColorToVertices(rrVertices, color)

	target.DrawTriangles(rrVertices, rrIndices, whiteImage, &ebiten.DrawTrianglesOptions{
		AntiAlias: true,
	})
}

var rrVertices []ebiten.Vertex
var rrIndices []uint16

func RoundedRectangle(pos Vec, size Vec, radius float64) ([]ebiten.Vertex, []uint16) {
	r := float32(radius)
	p := pos.AsVec32()
	s := size.AsVec32()

	var path vector.Path

	c0 := p
	c1 := p.Add(Vec32{X: s.X})
	c2 := p.Add(Vec32{Y: s.Y})
	c3 := p.Add(s)

	path.MoveTo(c0.X+r, c0.Y)
	path.ArcTo(c1.X, c1.Y, c3.X, c3.Y, r)
	path.ArcTo(c3.X, c3.Y, c2.X, c2.Y, r)
	path.ArcTo(c2.X, c2.Y, c0.X, c0.Y, r)
	path.ArcTo(c0.X, c0.Y, c1.X, c1.Y, r)

	rrVertices, rrIndices = path.AppendVerticesAndIndicesForFilling(rrVertices[:0], rrIndices[:0])
	return rrVertices, rrIndices
}

var vertexCache = sync.Pool{
	New: func() any {
		value := &VertexCache{}
		value.Self = value
		return value.Self
	},
}

type VertexCache struct {
	Self     any
	Vertices []ebiten.Vertex
	Indices  []uint16
}

func AcquireVertexCache() *VertexCache {
	return vertexCache.Get().(*VertexCache)
}

func (r *VertexCache) Release() {
	vertexCache.Put(r.Self)
}
