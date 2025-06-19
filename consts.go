package main

import "image/color"

type StationColor struct {
	Fill   color.NRGBA
	Stroke color.NRGBA
}

var StationColorSelected = StationColor{
	Fill:   rgbaOf(0x8e6d89ff),
	Stroke: rgbaOf(0x8e6d89ff),
}

var StationColorHover = StationColor{
	Fill:   rgbaOf(0xb089abff),
	Stroke: rgbaOf(0x8e6d89ff),
}

var StationColorIdle = StationColor{
	Fill:   rgbaOf(0x839ca9ff),
	Stroke: rgbaOf(0x6d838eff),
}

var StationColorPlanned = StationColor{
	Fill:   rgbaOf(0xcc9970ff),
	Stroke: rgbaOf(0xa97e5cff),
}

var StationColorConstructed = StationColor{
	Fill:   rgbaOf(0x87a985ff),
	Stroke: rgbaOf(0x6f8b6eff),
}

var BackgroundColor = rgbaOf(0xdbcfb1ff)
