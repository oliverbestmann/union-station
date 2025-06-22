package main

import (
	"bytes"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"io"
	"slices"
	"sync"
)

// bytesPerSample is the byte size for one sample (8 [bytes] = 2 [channels] * 4 [bytes] (32bit float)).
const bytesPerSample = 8

var AudioContext = sync.OnceValue(func() *audio.Context {
	return audio.NewContext(48000)
})

type Audio struct {
	Music       MonoSamples
	ButtonPress MonoSamples
	ButtonHover MonoSamples

	players []*audio.Player

	mute bool
}

func (a *Audio) PlayMusic() {
	if a.mute {
		return
	}

	infiniteStream := audio.NewInfiniteLoopF32(a.Music.ToStream(), int64(a.Music.Len()-4_800*bytesPerSample))
	a.playerOf(infiniteStream).Play()
}

func (a *Audio) Play(samples MonoSamples) {
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

func DecodeAudio(idle *IdleSuspend, stream *vorbis.Stream) MonoSamples {
	monoSamples := make([]byte, 0, stream.Length()/2)

	// ~100ms worth of audio data
	buf := make([]byte, stream.SampleRate()/10*bytesPerSample)

	remaining := stream.Length()
	for remaining > 0 {
		count := min(remaining, int64(len(buf)))

		n, err := io.ReadFull(stream, buf[:count])
		if err != nil {
			panic(err)
		}

		remaining -= int64(n)

		// copy every second sample to the samples buffer
		for idx := 0; idx < int(count); idx += bytesPerSample {
			monoSamples = append(monoSamples, buf[idx], buf[idx+1], buf[idx+2], buf[idx+3])
		}

		idle.MaybeSuspend()
	}

	return MonoSamples{buf: monoSamples}
}

type MonoSamples struct {
	buf []byte
}

func (m MonoSamples) ToStream() io.ReadSeeker {
	return NewStereoStream(bytes.NewReader(m.buf))
}

func (m MonoSamples) Len() int {
	return len(m.buf) * 2
}
