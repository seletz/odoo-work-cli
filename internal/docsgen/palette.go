package docsgen

import (
	"image/color"

	"github.com/charmbracelet/x/ansi"
)

// The terminal theme of the screenshots: Catppuccin Mocha as Ghostty ships
// it (Ghostty.app/Contents/Resources/ghostty/themes/Catppuccin Mocha, from
// github.com/catppuccin/ghostty). A real terminal shows its own theme for
// the 16 ANSI colours and the defaults; the screenshots show this one.
// The 256-colour cube, the grey ramp and true colours have fixed values
// and are drawn as they are.
var (
	background = color.RGBA{0x1e, 0x1e, 0x2e, 0xff} // base
	foreground = color.RGBA{0xcd, 0xd6, 0xf4, 0xff} // text

	ansiPalette = [16]color.RGBA{
		{0x45, 0x47, 0x5a, 0xff}, // 0 black: surface1
		{0xf3, 0x8b, 0xa8, 0xff}, // 1 red
		{0xa6, 0xe3, 0xa1, 0xff}, // 2 green
		{0xf9, 0xe2, 0xaf, 0xff}, // 3 yellow
		{0x89, 0xb4, 0xfa, 0xff}, // 4 blue
		{0xf5, 0xc2, 0xe7, 0xff}, // 5 magenta: pink
		{0x94, 0xe2, 0xd5, 0xff}, // 6 cyan: teal
		{0xba, 0xc2, 0xde, 0xff}, // 7 white: subtext1
		{0x58, 0x5b, 0x70, 0xff}, // 8 bright black: surface2
		{0xf7, 0xae, 0xc2, 0xff}, // 9
		{0xc2, 0xec, 0xbf, 0xff}, // 10
		{0xfc, 0xd6, 0x82, 0xff}, // 11
		{0xae, 0xcc, 0xfc, 0xff}, // 12
		{0xf3, 0x98, 0xda, 0xff}, // 13
		{0xb1, 0xea, 0xe1, 0xff}, // 14
		{0xa6, 0xad, 0xc8, 0xff}, // 15 bright white: subtext0
	}
)

// themed resolves a cell colour against the theme: the 16 ANSI colours,
// whether set as SGR 30–37/90–97 or as 256-colour indexes 0–15, take the
// theme's values; nil (the default) stays nil; every other colour is
// returned as is.
//
// :param c: the colour of a cell
// :returns: the colour to draw
func themed(c color.Color) color.Color {
	switch v := c.(type) {
	case ansi.BasicColor:
		if v < 16 {
			return ansiPalette[v]
		}
	case ansi.IndexedColor:
		if v < 16 {
			return ansiPalette[v]
		}
	}
	return c
}
