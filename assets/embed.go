package assets

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/oliverbestmann/union-station/qoa"
	"image/png"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"os"
	"runtime"
	"sync"
)

//go:embed button_hover.ogg
var button_hover_ogg []byte

//go:embed button_press.ogg
var button_press_ogg []byte

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

func Song1() io.ReadSeeker {
	return loadStreamOf("assets/song1.qoa")
}

func Song2() io.ReadSeeker {
	return loadStreamOf("assets/song2.qoa")
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

func loadStreamOf(name string) io.ReadSeeker {
	if runtime.GOOS == "js" {
		resp, err := http.Get(name)
		if err != nil {
			fmt.Printf("[assets] request failed %q: %s\n", name, err)
			return qoaOf(dummy_qoa)
		}

		defer resp.Body.Close()

		buf, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("[assets] failed to read %q: %s\n", name, err)
			return qoaOf(dummy_qoa)
		}

		return qoaOf(buf)
	}

	buf, err := os.ReadFile(name)
	if err != nil {
		fmt.Printf("[assets] failed to load %q: %s\n", name, err)
		return qoaOf(dummy_qoa)
	}

	return qoaOf(buf)
}

func qoaOf(buf []byte) io.ReadSeeker {
	hdr, samples, err := qoa.Decode(buf)
	if err != nil {
		panic(err)
	}

	return &float32BytesReader{
		r: qoa.NewReader(samples, int(hdr.Channels)),
	}
}

type float32BytesReader struct {
	r      io.Reader
	eof    bool
	i16Buf []byte
}

func (r *float32BytesReader) Read(buf []byte) (int, error) {
	if r.eof && len(r.i16Buf) == 0 {
		return 0, io.EOF
	}

	if i16LenToFill := len(buf) / 4 * 2; len(r.i16Buf) < i16LenToFill && !r.eof {
		origLen := len(r.i16Buf)
		if cap(r.i16Buf) < i16LenToFill {
			r.i16Buf = append(r.i16Buf, make([]byte, i16LenToFill-origLen)...)
		}

		// Read int16 bytes.
		n, err := r.r.Read(r.i16Buf[origLen:i16LenToFill])
		if err != nil && err != io.EOF {
			return 0, err
		}
		if err == io.EOF {
			r.eof = true
		}
		r.i16Buf = r.i16Buf[:origLen+n]
	}

	// Convert int16 bytes to float32 bytes and fill buf.
	samplesToFill := min(len(r.i16Buf)/2, len(buf)/4)
	for i := 0; i < samplesToFill; i++ {
		vi16l := r.i16Buf[2*i]
		vi16h := r.i16Buf[2*i+1]
		v := float32(int16(vi16l)|int16(vi16h)<<8) / (1 << 15)
		vf32 := math.Float32bits(v)
		buf[4*i] = byte(vf32)
		buf[4*i+1] = byte(vf32 >> 8)
		buf[4*i+2] = byte(vf32 >> 16)
		buf[4*i+3] = byte(vf32 >> 24)
	}

	// Copy the remaining part for the next read.
	copy(r.i16Buf, r.i16Buf[samplesToFill*2:])
	r.i16Buf = r.i16Buf[:len(r.i16Buf)-samplesToFill*2]

	n := samplesToFill * 4
	if r.eof {
		return n, io.EOF
	}
	return n, nil
}

func (r *float32BytesReader) Seek(offset int64, whence int) (int64, error) {
	s, ok := r.r.(io.Seeker)
	if !ok {
		return 0, fmt.Errorf("float32: the source must be io.Seeker when seeking but not")
	}
	r.i16Buf = r.i16Buf[:0]
	r.eof = false
	n, err := s.Seek(offset/4*2, whence)
	if err != nil {
		return 0, err
	}
	return n / 2 * 4, nil
}
