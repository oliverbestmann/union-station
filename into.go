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

func (st *DialogStack) Update(dt float64) {
	if len(st.dialogs) > 0 {
		dialog := &st.dialogs[len(st.dialogs)-1]

		if dialog.Modal {
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
	Id         string
	Texts      []Text
	Modal      bool
	ButtonText string
	Padding    Vec

	// the minimum size of the dialog (without padding)
	MinSize Vec
}

func (d *Dialog) Draw(target *ebiten.Image) {
	size := d.Size()

	// base position of the dialog so it is centered on the screen
	pos := imageSizeOf(target).Mulf(0.5).Sub(size.Mulf(0.5))

	d.DrawAt(target, pos)
}

func (d *Dialog) DrawAt(target *ebiten.Image, pos Vec) {
	size := d.Size()

	// draw the background
	DrawWindow(target, pos, size)

	// draw the text
	DrawTexts(target, pos.Add(d.paddingWithDefaultValue()), d.Texts)
}

func (d *Dialog) paddingWithDefaultValue() Vec {
	if !d.Padding.IsZero() {
		return d.Padding
	}

	return vecSplat(24)
}

func (d *Dialog) Size() Vec {
	textSize := MeasureTexts(d.Texts)

	size := textSize.Add(d.paddingWithDefaultValue().Mulf(2))
	size.X = max(size.X, d.MinSize.X)
	size.Y = max(size.Y, d.MinSize.Y)
	return size
}
