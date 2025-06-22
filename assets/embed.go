package assets

import (
	"bytes"
	_ "embed"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"image/png"
	_ "image/png"
	"sync"
)

//go:embed music.ogg
var music_ogg []byte

//go:embed button_hover.ogg
var button_hover_ogg []byte

//go:embed button_press.ogg
var button_press_ogg []byte

//go:embed coin.png
var coin_png []byte

//go:embed coin-planned.png
var coin_planned_png []byte

//go:embed CoinageCapsKrugerGray.ttf
var font_ttf []byte

var Coin = sync.OnceValue(func() *ebiten.Image {
	image, _ := png.Decode(bytes.NewReader(coin_png))
	return ebiten.NewImageFromImage(image)
})

var PlannedCoin = sync.OnceValue(func() *ebiten.Image {
	image, _ := png.Decode(bytes.NewReader(coin_planned_png))
	return ebiten.NewImageFromImage(image)
})

var Font = sync.OnceValue(func() *text.GoTextFaceSource {
	f, _ := text.NewGoTextFaceSource(bytes.NewReader(font_ttf))
	return f
})

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
