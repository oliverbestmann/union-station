package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	. "github.com/quasilyte/gmath"
	"slices"
)

type DialogStack struct {
	dialogs []Dialog
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

func (st *DialogStack) Draw(screen *ebiten.Image) {
	if len(st.dialogs) > 0 {
		dialog := &st.dialogs[len(st.dialogs)-1]
		dialog.Draw(screen)
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

	if d.Modal {
		screenSize := imageSizeOf(target)
		op := &ebiten.DrawImageOptions{}
		op.ColorScale.ScaleWithColor(rgbaOf(0x00000010))
		op.GeoM.Scale(screenSize.X, screenSize.Y)
		target.DrawImage(whiteImage, op)
	}

	// draw the background
	DrawWindow(target, pos, size)

	// draw the text
	DrawTexts(target, pos.Add(d.paddingWithDefaultValue()), d.Texts)
}

func (d *Dialog) paddingWithDefaultValue() Vec {
	if !d.Padding.IsZero() {
		return d.Padding
	}

	return splatVec(24)
}

func (d *Dialog) Size() Vec {
	textSize := MeasureTexts(d.Texts)

	size := textSize.Add(d.paddingWithDefaultValue().Mulf(2))
	size.X = max(size.X, d.MinSize.X)
	size.Y = max(size.Y, d.MinSize.Y)
	return size
}
