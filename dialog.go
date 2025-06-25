package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	. "github.com/quasilyte/gmath"
	"slices"
)

type DialogStack struct {
	dialogs     []Dialog
	modalAlpha  float64
	initialized bool
}

func (st *DialogStack) Close() {
	if len(st.dialogs) > 0 {
		// pop the last dialog
		st.dialogs = st.dialogs[:len(st.dialogs)-1]
	}
}

func (st *DialogStack) CloseById(id string) {
	st.dialogs = slices.DeleteFunc(st.dialogs, func(dialog Dialog) bool {
		return dialog.Id == id
	})
}

func (st *DialogStack) Push(dialog Dialog) {
	st.dialogs = append(st.dialogs, dialog)
}

func (st *DialogStack) Clear() {
	st.dialogs = nil
}

func (st *DialogStack) Update(dt float64) (modal bool) {
	if len(st.dialogs) > 0 {
		dialog := &st.dialogs[len(st.dialogs)-1]

		if dialog.Button != nil {
			cursor := Cursor()
			dialog.Button.Hover(cursor)
			dialog.Button.IsClicked(cursor)
		}

		if dialog.Modal {
			modal = true
			st.modalAlpha = min(1, st.modalAlpha+8*dt)

			if !st.initialized {
				st.modalAlpha = 1
			}

		} else {
			st.modalAlpha = max(0, st.modalAlpha-4*dt)
		}
	} else {
		// no dialog, decrease alpha
		st.modalAlpha = max(0, st.modalAlpha-4*dt)
	}

	st.initialized = true

	return
}

func (st *DialogStack) Draw(target *ebiten.Image) {
	if st.modalAlpha > 0 {
		screenSize := imageSizeOf(target)
		op := &ebiten.DrawImageOptions{}
		op.ColorScale.ScaleWithColor(rgbaOf(0xada387ff))
		op.ColorScale.ScaleAlpha(float32(0.2 * st.modalAlpha))
		op.GeoM.Scale(screenSize.X, screenSize.Y)
		target.DrawImage(whiteImage, op)
	}

	if len(st.dialogs) > 0 {
		dialog := &st.dialogs[len(st.dialogs)-1]
		dialog.Draw(target)
	}
}

type Dialog struct {
	Id      string
	Texts   []Text
	Modal   bool
	Button  *Button
	Padding Vec

	// the minimum size of the dialog (without padding)
	MinSize Vec
}

func (d *Dialog) Layout(screenSize Vec) {
	if d.Button != nil {
		size, button := d.Measure()
		origin := screenSize.Mulf(0.5).Sub(size.Mulf(0.5))
		d.Button.Position = button.Add(origin)
	}
}

func (d *Dialog) Draw(target *ebiten.Image) {
	size, _ := d.Measure()

	// base position of the dialog so it is centered on the screen
	screenSize := imageSizeOf(target)
	d.Layout(screenSize)
	pos := screenSize.Mulf(0.5).Sub(size.Mulf(0.5))

	d.DrawAt(target, pos)
}

func (d *Dialog) DrawAt(target *ebiten.Image, pos Vec) {
	size, _ := d.Measure()

	// draw the background
	DrawWindow(target, pos, size)

	// draw the text
	textPos := pos.Add(d.paddingWithDefaultValue())
	DrawTexts(target, textPos, d.Texts)

	// draw the button
	if d.Button != nil {
		d.Button.Draw(target)
	}
}

func (d *Dialog) paddingWithDefaultValue() Vec {
	if !d.Padding.IsZero() {
		return d.Padding
	}

	return vecSplat(24)
}

func (d *Dialog) Measure() (size Vec, button Vec) {
	textSize := MeasureTexts(d.Texts)

	size = textSize.Add(d.paddingWithDefaultValue())

	const buttonSpacing = 16

	// if we have a button, add space for the button
	if d.Button != nil {
		size.Y += buttonSpacing

		// account for the width of the button
		size.X = max(size.X, d.Button.Size.X)

		// extract position of button
		button = Vec{X: size.X/2 - d.Button.Size.X/2, Y: size.Y}

		// add the button height to the size
		size.Y += d.Button.Size.Y
	}

	size = size.Add(d.paddingWithDefaultValue())

	size.X = max(size.X, d.MinSize.X)
	size.Y = max(size.Y, d.MinSize.Y)

	return
}

func (d *Dialog) ButtonClicked() {

}
