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

func (st *DialogStack) ById(id string) *Dialog {
	for idx := range st.dialogs {
		dialog := &st.dialogs[idx]
		if dialog.Id == id {
			return dialog
		}
	}

	return nil
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

		cursor := Cursor()
		for _, button := range dialog.Buttons {
			button.Hover(cursor)
			button.Clicked(cursor)
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
	Buttons []*Button
	Padding Vec

	// the minimum size of the dialog (without padding)
	MinSize Vec
}

func (d *Dialog) Layout(screenSize Vec) {
	if len(d.Buttons) > 0 {
		// position is relative to the dialogs origin
		size, pos := d.Measure()

		// origin of the dialog
		origin := screenSize.Mulf(0.5).Sub(size.Mulf(0.5))

		for _, button := range d.Buttons {
			button.Position = pos.Add(origin)
			pos = pos.Add(Vec{X: button.Size.X}).Add(Vec{X: 16})
		}
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
	for _, button := range d.Buttons {
		button.Draw(target)
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
	if len(d.Buttons) > 0 {
		size.Y += buttonSpacing

		var buttonsWidth float64
		var buttonsHeight float64
		for idx, button := range d.Buttons {
			spacing := iff(idx == 0, 0.0, 16.0)
			buttonsWidth += button.Size.X + spacing

			buttonsHeight = max(buttonsHeight, button.Size.Y)
		}

		// account for the width of the button
		size.X = max(size.X, buttonsWidth)

		// extract position of button
		button = Vec{X: size.X/2 - buttonsWidth/2, Y: size.Y}

		// add the button height to the size
		size.Y += buttonsHeight
	}

	size = size.Add(d.paddingWithDefaultValue())

	size.X = max(size.X, d.MinSize.X)
	size.Y = max(size.Y, d.MinSize.Y)

	return
}

func (d *Dialog) ButtonClicked() {

}
