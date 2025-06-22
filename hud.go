package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/oliverbestmann/union-station/assets"
	. "github.com/quasilyte/gmath"
	"image/color"
)

func (g *Game) drawHUD(screen *ebiten.Image) {
	pos := Vec{X: float64(imageWidth(screen) - 16), Y: 16}
	screenSize := imageSizeOf(screen)

	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleWithColor(rgbaOf(0xffffff40))
	op.GeoM.Scale(screenSize.X, 64)
	screen.DrawImage(whiteImage, op)

	if g.stats.CoinsTotal > 0 {
		msg := fmt.Sprintf("Budget: %d", g.stats.CoinsAvailable())
		g.drawRectangleWithCoin(screen, &pos, msg, HudTextColor, assets.Coin())

		if g.stats.CoinsPlanned > 0 {
			// add some space between the rectangles
			pos.X -= 16

			msg := fmt.Sprintf("Planned: %d", g.stats.CoinsPlanned)
			g.drawRectangleWithCoin(screen, &pos, msg, HudPlannedRectangleColor, assets.PlannedCoin())
		}

		if g.stats.StationsConnected > 0 {
			// add some space between the rectangles
			pos.X -= 16

			msg := fmt.Sprintf("Connected %d of %d", g.stats.StationsConnected, g.stats.StationsTotal)
			g.drawRectangleWithCoin(screen, &pos, msg, HudTextColor, nil)
		}
	}
}

func (g *Game) drawRectangleWithCoin(target *ebiten.Image, pos *Vec, msg string, rectangleColor color.Color, icon *ebiten.Image) {
	textWidth := MeasureText(Font24, msg).X

	var iconSize Vec
	if icon != nil {
		iconSize = imageSizeOf(icon)
	}

	// 16px padding, 8px gap, icon size
	rectangleSize := Vec{X: textWidth + 8 + iconSize.X + 16*2, Y: 48}
	rectanglePos := Vec{X: pos.X - rectangleSize.X, Y: pos.Y - 8}
	DrawRoundRect(target, rectanglePos, rectangleSize, rectangleColor)

	// padding within the rectangle
	pos.X -= 16

	if icon != nil {
		// draw the coin icon
		pos.X -= iconSize.X
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(pos.X, pos.Y)
		target.DrawImage(icon, op)

		// spacing to the text
		pos.X -= 8
	}

	pos.X -= textWidth
	DrawTextLeft(target, msg, Font24, *pos, BackgroundColor)

	// add some padding within the rectangle
	pos.X -= 16
}

func DrawRoundRect(screen *ebiten.Image, rectanglePos Vec, rectangleSize Vec, color color.Color) {
	rrVertices, rrIndices := RoundedRectangle(rectanglePos, rectangleSize, 8)

	c := colorm.ColorM{}
	c.ScaleWithColor(color)

	colorm.DrawTriangles(screen, rrVertices, rrIndices, whiteImage, c, &colorm.DrawTrianglesOptions{
		AntiAlias: true,
	})
}

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

	return path.AppendVerticesAndIndicesForFilling(nil, nil)
}
