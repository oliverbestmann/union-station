package main

import (
	"bytes"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/oliverbestmann/union-station/assets"
	"github.com/oliverbestmann/union-station/qoa"
	"io"
	"math"
	"slices"
	"sync"
	"time"
)

// bytesPerSample is the byte size for one sample (8 [bytes] = 2 [channels] * 4 [bytes] (32bit float)).
const bytesPerSample = 8

var AudioContext = sync.OnceValue(func() *audio.Context {
	return audio.NewContext(44100)
})

type Audio struct {
	Songs       []assets.MakeStream
	ButtonPress Samples
	ButtonHover Samples

	players []*audio.Player

	Mute bool
}

func (a *Audio) PlayMusic() {
	go func() {
		var current *audio.Player
		var idx int
		for {
			// check if we're still playing
			time.Sleep(500 * time.Millisecond)

			if !AudioContext().IsReady() {
				continue
			}

			if current == nil || !current.IsPlaying() {
				song := a.Songs[idx%len(a.Songs)]
				idx += 1

				stream, err := qoaStreamOf(song())
				if err != nil {
					fmt.Printf("Failed to open song: %s\n", err)
					continue
				}

				current = a.playerOf(stream)
				current.Play()
			}
		}
	}()
}

func (a *Audio) Play(samples Samples) {
	if a.Mute {
		return
	}

	a.playerOf(samples.ToStream()).Play()
}

func (a *Audio) ToggleMute() {
	a.Cleanup()

	// toggle mute flag
	a.Mute = !a.Mute

	// calculate mute volume
	volume := 1.0
	if a.Mute {
		volume = 0.0
	}

	// set volume on all players
	for _, p := range a.players {
		p.SetVolume(volume)
	}
}

func (a *Audio) playerOf(stream io.Reader) *audio.Player {
	// whenever we start a new player, we remove all references to
	// now dead players
	a.Cleanup()

	// create the new player
	player, _ := AudioContext().NewPlayerF32(stream)

	// and record it to handle volume updates later
	a.players = append(a.players, player)

	return player
}

func (a *Audio) Cleanup() {
	// remove players that are not playing
	a.players = slices.DeleteFunc(a.players, func(player *audio.Player) bool {
		return !player.IsPlaying()
	})
}

type Samples assets.Int16Samples

func (m Samples) ToStream() io.Reader {
	return &Int16ToFloat32Reader{r: bytes.NewReader(m)}
}

func qoaStreamOf(r io.Reader) (io.Reader, error) {
	dec, err := qoa.NewDecoder(r)
	if err != nil {
		return nil, fmt.Errorf("open qoa stream: %w", err)
	}

	stream := &Int16ToFloat32Reader{
		r: qoa.NewStream(dec),
	}

	return stream, nil
}

type Int16ToFloat32Reader struct {
	r        io.Reader
	eof      bool
	i16Buf   []byte
	byteSize int64
}

func (r *Int16ToFloat32Reader) Read(buf []byte) (int, error) {
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

/*
func (r *Int16ToFloat32Reader) Seek(offset int64, whence int) (int64, error) {
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
*/
