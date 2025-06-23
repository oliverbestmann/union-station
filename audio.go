package main

import (
	"bytes"
	"errors"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"io"
	"slices"
	"sync"
	"time"
)

// bytesPerSample is the byte size for one sample (8 [bytes] = 2 [channels] * 4 [bytes] (32bit float)).
const bytesPerSample = 8

var AudioContext = sync.OnceValue(func() *audio.Context {
	return audio.NewContext(48000)
})

type Audio struct {
	Songs       []Samples
	ButtonPress Samples
	ButtonHover Samples

	players []*audio.Player

	mute bool
}

func (a *Audio) PlayMusic() {
	var readers []io.Reader

	for _, samples := range a.Songs {
		readers = append(readers, samples.ToStream())
	}

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

				current = a.playerOf(song.ToStream())
				current.Play()
			}
		}
	}()
}

func (a *Audio) Play(samples Samples) {
	if a.mute {
		return
	}

	a.playerOf(samples.ToStream()).Play()
}

func (a *Audio) ToggleMute() {
	a.Cleanup()

	// toggle mute flag
	a.mute = !a.mute

	// calculate mute volume
	volume := 1.0
	if a.mute {
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

type Stream interface {
	io.Reader
	SampleRate() int
	Length() int64
}

func DecodeAudio(idle *IdleSuspend, stream Stream) Samples {
	samples := make([]byte, 0, max(1024, stream.Length()))

	// ~50ms worth of audio data
	buf := make([]byte, stream.SampleRate()/20*bytesPerSample)

	for {
		n, err := io.ReadFull(stream, buf)
		if n > 0 {
			samples = append(samples, buf[:n]...)
		}

		switch {
		case errors.Is(err, io.ErrUnexpectedEOF):
			return Samples{buf: samples}

		case err != nil:
			panic(err)
		}

		idle.MaybeSuspend()
	}
}

type Samples struct {
	buf []byte
}

func (m Samples) ToStream() io.ReadSeeker {
	return bytes.NewReader(m.buf)
}

func (m Samples) Len() int {
	return len(m.buf)
}
