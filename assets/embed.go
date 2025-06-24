package assets

import (
	"bytes"
	_ "embed"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/neilotoole/streamcache"
	"github.com/oliverbestmann/union-station/fetch"
	"github.com/oliverbestmann/union-station/qoa"
	"image/png"
	"io"
	"os"
	"runtime"
	"sync"
)

//go:embed button_hover.qoa
var button_hover_qoa []byte

//go:embed button_press.qoa
var button_press_qoa []byte

//go:embed dummy.qoa
var dummy_qoa []byte

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

func Songs() []MakeStream {
	return []MakeStream{
		loadStreamOf("assets/song2.qoa"),
		loadStreamOf("assets/song1.qoa"),
	}
}

type Int16Samples []byte

func ButtonHover() Int16Samples {
	return samplesOf(button_hover_qoa)
}

func ButtonPress() Int16Samples {
	return samplesOf(button_press_qoa)
}

func loadStreamOf(name string) MakeStream {
	if runtime.GOOS == "js" {
		value := sync.OnceValue(func() *streamcache.Stream {
			return streamcache.New(fetch.Fetch(name))
		})

		return func() io.ReadCloser {
			return value().NewReader(nil)
		}
	}

	buf, err := os.ReadFile(name)
	if err != nil {
		panic(err)
	}

	return func() io.ReadCloser {
		return io.NopCloser(bytes.NewReader(buf))
	}
}

func qoaOf(buf []byte) *qoa.Stream {
	decoder, err := qoa.NewDecoder(bytes.NewReader(buf))
	if err != nil {
		panic(err)
	}

	return qoa.NewStream(decoder)
}

func samplesOf(buf []byte) Int16Samples {
	samples, _ := io.ReadAll(qoaOf(buf))
	return samples
}

type MakeStream func() io.ReadCloser
