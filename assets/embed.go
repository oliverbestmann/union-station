package assets

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"image/png"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"runtime"
	"sync"
)

//go:embed button_hover.ogg
var button_hover_ogg []byte

//go:embed button_press.ogg
var button_press_ogg []byte

//go:embed dummy.ogg
var dummy_ogg []byte

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

func Song1() *vorbis.Stream {
	return loadStreamOf("assets/song1.ogg")
}

func Song2() *vorbis.Stream {
	return loadStreamOf("assets/song2.ogg")
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

func loadStreamOf(name string) *vorbis.Stream {
	if runtime.GOOS == "js" {
		resp, err := http.Get(name)
		if err != nil {
			fmt.Printf("[assets] request failed %q: %s\n", name, err)
			return decoderOf(dummy_ogg)
		}

		defer resp.Body.Close()

		buf, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("[assets] failed to read %q: %s\n", name, err)
			return decoderOf(dummy_ogg)
		}

		return decoderOf(buf)
	}

	buf, err := os.ReadFile(name)
	if err != nil {
		fmt.Printf("[assets] failed to load %q: %s\n", name, err)
		return decoderOf(dummy_ogg)
	}

	return decoderOf(buf)
}
