package assets

import (
	"bytes"
	_ "embed"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	_ "image/png"
)

//go:embed music.ogg
var music_ogg []byte

//go:embed button_hover.ogg
var button_hover_ogg []byte

//go:embed button_press.ogg
var button_press_ogg []byte

func Music() *vorbis.Stream {
	return decoderOf(music_ogg)
}

func ButtonHover() *vorbis.Stream {
	return decoderOf(button_hover_ogg)
}

func ButtonPress() *vorbis.Stream {
	return decoderOf(button_press_ogg)
}

func decoderOf(ogg []byte) *vorbis.Stream {
	s, err := vorbis.DecodeF32(bytes.NewReader(ogg))
	if err != nil {
		panic(err)
	}

	return s
}
