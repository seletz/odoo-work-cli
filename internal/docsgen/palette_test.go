package docsgen

import (
	"image/color"
	"testing"

	"charm.land/lipgloss/v2"
)

// TestPalette_ANSIMapsToMocha: the 16 ANSI colours, however the TUI asks
// for them, come out in Catppuccin Mocha; the 256-colour cube, the grey
// ramp and true colours stay what they are.
func TestPalette_ANSIMapsToMocha(t *testing.T) {
	for _, tc := range []struct {
		name  string
		style lipgloss.Style
		want  color.RGBA
	}{
		{"lipgloss 5 (company colour)", lipgloss.NewStyle().Foreground(lipgloss.Color("5")), rgb(0xf5c2e7)},
		{"lipgloss 2", lipgloss.NewStyle().Foreground(lipgloss.Color("2")), rgb(0xa6e3a1)},
		{"lipgloss 13 (bright)", lipgloss.NewStyle().Foreground(lipgloss.Color("13")), rgb(0xf398da)},
		{"lipgloss 0", lipgloss.NewStyle().Foreground(lipgloss.Color("0")), rgb(0x45475a)},
		{"cube 63", lipgloss.NewStyle().Foreground(lipgloss.Color("63")), rgb(0x5f5fff)},
		{"grey ramp 236", lipgloss.NewStyle().Foreground(lipgloss.Color("236")), rgb(0x303030)},
		{"true colour", lipgloss.NewStyle().Foreground(lipgloss.Color("#5f87ff")), rgb(0x5f87ff)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// A full block is all ink: its centre shows the foreground.
			img := mustImage(t, frameOf(tc.style.Render("█"), 1, 1))
			if got := centre(img, 0, 0); got != tc.want {
				t.Errorf("centre = %v, want %v", got, tc.want)
			}
		})
	}
	for _, tc := range []struct {
		seq  string
		want color.RGBA
	}{
		{"\x1b[35m█", rgb(0xf5c2e7)},
		{"\x1b[38;5;5m█", rgb(0xf5c2e7)},
		{"\x1b[95m█", rgb(0xf398da)},
		{"\x1b[45m ", rgb(0xf5c2e7)},
	} {
		if got := centre(mustImage(t, frameOf(tc.seq, 1, 1)), 0, 0); got != tc.want {
			t.Errorf("%q: centre = %v, want %v", tc.seq, got, tc.want)
		}
	}
}

// TestPalette_Defaults: unstyled text and the window are Mocha's text and
// base.
func TestPalette_Defaults(t *testing.T) {
	if background != rgb(0x1e1e2e) || foreground != rgb(0xcdd6f4) {
		t.Errorf("background %v, foreground %v; want Mocha base #1e1e2e and text #cdd6f4", background, foreground)
	}
	img := mustImage(t, frameOf("█ ", 2, 1))
	if got := centre(img, 0, 0); got != foreground {
		t.Errorf("default foreground = %v", got)
	}
	if got := centre(img, 1, 0); got != background {
		t.Errorf("default background = %v", got)
	}
}

func rgb(v uint32) color.RGBA {
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xff}
}
