package main

import "image/color"

type StationColor struct {
	Fill   color.Color
	Stroke color.Color
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

var DebugColor color.Color = color.RGBA{R: 0xff, B: 0xff, A: 0xff}
var BackgroundColor color.Color = rgbaOf(0xdbcfb1ff)
var DarkTextColor color.Color = rgbaOf(0x937b6aff)
var WaterColor color.Color = rgbaOf(0x6d838eff)
var TooltipColor color.Color = rgbaOf(0xeee1c4ff)
var ShadowColor color.Color = rgbaOf(0xada38780)

var LightTextColor color.Color = rgbaOf(0xdbcfb1ff)

var HudRectangleColor color.Color = rgbaOf(0x937b6aff)
var HudPlannedRectangleColor = StationColorPlanned.Stroke

var StartGameButtonColors = ButtonColors{
	Normal: color.Transparent,
	Hover:  scaleColorWithAlpha(rgbaOf(0x6f8b6eff), 0.25),
	Text:   LightTextColor,
}

var BuildButtonColors = ButtonColors{
	Normal:   rgbaOf(0x6f8b6eff),
	Hover:    rgbaOf(0x87a985ff),
	Disabled: rgbaOf(0xa05e5eff),
	Text:     LightTextColor,
	Shadow:   ShadowColor,
}

var PlanButtonColors = ButtonColors{
	Normal: StationColorPlanned.Stroke,
	Hover:  StationColorPlanned.Fill,
	Text:   LightTextColor,
	Shadow: ShadowColor,
}

var HudButtonColors = ButtonColors{
	Normal: HudRectangleColor,
	Hover:  HudPlannedRectangleColor,
	Text:   LightTextColor,
	Shadow: ShadowColor,
}

var AcceptButtonColors = ButtonColors{
	Normal:   rgbaOf(0x6f8b6eff),
	Hover:    rgbaOf(0x87a985ff),
	Disabled: rgbaOf(0xa05e5eff),
	Text:     LightTextColor,
	Shadow:   ShadowColor,
}
