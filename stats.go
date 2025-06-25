package main

import (
	"math"
	"strconv"
)

type Coins int

func (c Coins) String() string {
	return strconv.Itoa(int(c)) + "c"
}

type Stats struct {
	CoinsTotal   Coins
	CoinsSpent   Coins
	CoinsPlanned Coins

	StationsTotal     int
	StationsConnected int

	Score int
}

func (s *Stats) CoinsAvailable() Coins {
	return s.CoinsTotal - s.CoinsSpent
}

func priceOf(one, two *Station) Coins {
	price := one.Position.DistanceTo(two.Position)
	return Coins(math.Ceil(price/100) * 10)
}
