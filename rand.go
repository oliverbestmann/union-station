package main

import (
	"github.com/quasilyte/gmath"
	"math/rand/v2"
)

func RandWithSeed(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed))

}

func Randf[T ~float64 | ~float32](rng *rand.Rand, min, max T) T {
	return T(rng.Float64())*(max-min) + min
}

func RandVecIn(rng *rand.Rand, rect gmath.Rect) gmath.Vec {
	return gmath.Vec{
		X: Randf(rng, rect.Min.X, rect.Max.X),
		Y: Randf(rng, rect.Min.Y, rect.Max.Y),
	}
}

func Choose[T any](rng *rand.Rand, values ...T) T {
	idx := rng.IntN(len(values))
	return values[idx]
}

func prob(rng *rand.Rand, prop float64) bool {
	return rng.Float64() < prop
}

func Shuffled[T any](rng *rand.Rand, values []T) []T {
	values = append([]T(nil), values...)

	rng.Shuffle(len(values), func(i, j int) {
		values[i], values[j] = values[j], values[i]
	})

	return values
}
