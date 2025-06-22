package tween

import (
	"github.com/quasilyte/gmath"
	"slices"
	"time"
)

type TweenTarget func(f float64, elapsed, duration time.Duration)

type Tween interface {
	Update(dt time.Duration) (done bool)
}

type Simple struct {
	Duration time.Duration
	Target   TweenTarget
	Ease     func(t float64) float64

	elapsed time.Duration
}

func (t *Simple) Update(dt time.Duration) bool {
	if t.Duration <= 0 {
		return true
	}

	t.elapsed += dt

	f := min(1, float64(t.elapsed)/float64(t.Duration))

	if t.Ease != nil {
		f = t.Ease(f)
	}

	if t.Target != nil {
		t.Target(f, t.elapsed, t.Duration)
	}

	// return if finished
	return t.elapsed >= t.Duration
}

func Sequence(tweens ...Tween) Tween {
	return &tweensSequence{tweens: tweens}
}

type tweensSequence struct {
	tweens []Tween
}

func (s *tweensSequence) Update(dt time.Duration) bool {
	if len(s.tweens) > 0 {
		if done := s.tweens[0].Update(dt); done {
			s.tweens = s.tweens[1:]
		}
	}

	return len(s.tweens) == 0
}

type tweensConcurrent struct {
	tweens []Tween
}

func (t *tweensConcurrent) Update(dt time.Duration) (done bool) {
	t.tweens = slices.DeleteFunc(t.tweens, func(tween Tween) bool {
		return tween.Update(dt)
	})

	return len(t.tweens) == 0
}

func Concurrent(tweens ...Tween) Tween {
	return &tweensConcurrent{tweens: tweens}
}

func LerpValue(target *float64, from, to float64) TweenTarget {
	return func(f float64, _, _ time.Duration) {
		*target = gmath.Lerp(from, to, f)
	}
}

func Delay(delay time.Duration, next Tween) Tween {
	first := &Simple{Duration: delay}
	return Sequence(first, next)
}
