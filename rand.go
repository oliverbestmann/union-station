package main

import "math/rand/v2"

func randf[T ~float64](rng *rand.Rand, min, max T) T {
	return T(rng.Float64())*(max-min) + min
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
