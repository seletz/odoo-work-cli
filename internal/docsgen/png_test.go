package docsgen

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func frameOf(content string, w, h int) Frame {
	return Frame{Shot: Shot{Name: "t", Width: w, Height: h}, Content: content}
}

func mustImage(t *testing.T, f Frame) *image.RGBA {
	t.Helper()
	img, err := Image(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// centre returns the colour at the centre of a cell.
func centre(img *image.RGBA, col, row int) color.RGBA {
	return img.RGBAAt(padX+col*cellW+cellW/2, titleBar+row*cellH+cellH/2)
}

// at returns the colour at a point of a cell, given in eighths of its size.
func pixelAt(img *image.RGBA, col, row, ex, ey int) color.RGBA {
	return img.RGBAAt(padX+col*cellW+ex*cellW/8, titleBar+row*cellH+ey*cellH/8)
}

func near(a, b color.RGBA) bool {
	d := func(x, y uint8) int {
		if x > y {
			return int(x - y)
		}
		return int(y - x)
	}
	return d(a.R, b.R) < 24 && d(a.G, b.G) < 24 && d(a.B, b.B) < 24
}

// TestImage_SizeFollowsTerminalCells: one cell per terminal cell plus the
// window chrome; outside the rounded corner the image is transparent.
func TestImage_SizeFollowsTerminalCells(t *testing.T) {
	img := mustImage(t, frameOf("x", 80, 24))
	w, h := imageSize(80, 24)
	if got := img.Bounds(); got.Dx() != w || got.Dy() != h {
		t.Errorf("image is %dx%d, want %dx%d", got.Dx(), got.Dy(), w, h)
	}
	if w != 80*cellW+2*padX || h != 24*cellH+titleBar+padY {
		t.Errorf("imageSize(80, 24) = %dx%d does not follow the cell grid", w, h)
	}
	if c := img.RGBAAt(0, 0); c.A != 0 {
		t.Errorf("corner pixel = %v, want transparent", c)
	}
}

// TestImage_Colours: a background colour fills its cell up to the corner,
// text is drawn in its foreground, an empty cell shows the window
// background, reverse swaps the colours.
func TestImage_Colours(t *testing.T) {
	img := mustImage(t, frameOf("\x1b[1;38;2;255;95;95;48;2;26;42;26mHi\x1b[m \x1b[7m \x1b[m", 5, 1))
	if c := img.RGBAAt(padX+1, titleBar+1); !near(c, color.RGBA{26, 42, 26, 255}) {
		t.Errorf("corner of the green cell = %v", c)
	}
	red := false
	for y := titleBar; y < titleBar+cellH; y++ {
		for x := padX; x < padX+cellW; x++ {
			if near(img.RGBAAt(x, y), color.RGBA{255, 95, 95, 255}) {
				red = true
			}
		}
	}
	if !red {
		t.Error("no pixel of the H in its foreground colour")
	}
	if c := centre(img, 2, 0); !near(c, background) {
		t.Errorf("empty cell = %v, want the background %v", c, background)
	}
	if c := centre(img, 3, 0); !near(c, foreground) {
		t.Errorf("reverse blank = %v, want the foreground %v", c, foreground)
	}
}

// TestImage_EveryGlyphIsCovered: every character of the shots is either
// drawn as a shape or has a glyph in Go Mono; a missing glyph would show
// as an empty box.
func TestImage_EveryGlyphIsCovered(t *testing.T) {
	fs, err := faces()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[rune]bool{}
	for _, f := range frames(t) {
		for _, r := range f.Plain() {
			if r == '\n' || seen[r] {
				continue
			}
			seen[r] = true
			if _, box := boxGlyphs[string(r)]; box || isShape(r) {
				continue
			}
			for i, face := range fs {
				if !face.has(r) {
					t.Errorf("%q (%U) of the shots has no glyph in face %d and no shape", r, r, i)
				}
			}
		}
	}
}

// TestImage_TextIsDrawn: the glyphs and shapes the TUI uses leave ink in
// their cell.
func TestImage_TextIsDrawn(t *testing.T) {
	glyphs := []rune("Aa0•—…↑↓←→↵●·–")
	img := mustImage(t, frameOf(string(glyphs), len(glyphs), 1))
	for i, g := range glyphs {
		ink := false
		for y := titleBar; y < titleBar+cellH && !ink; y++ {
			for x := padX + i*cellW; x < padX+(i+1)*cellW; x++ {
				if !near(img.RGBAAt(x, y), background) {
					ink = true
					break
				}
			}
		}
		if !ink {
			t.Errorf("%q leaves no ink", g)
		}
	}
}

// TestImage_LinesAreContinuous: a rule of box-drawing glyphs is one
// unbroken line from the first to the last cell, light, heavy and double.
func TestImage_LinesAreContinuous(t *testing.T) {
	for _, tc := range []struct{ name, row string }{
		{"light", "─┬─┼─┴─"}, {"heavy", "━━━━━━━"}, {"double", "═══════"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			img := mustImage(t, frameOf(tc.row, 7, 1))
			rows := 0
			for y := titleBar; y < titleBar+cellH; y++ {
				full := true
				for x := padX; x < padX+7*cellW; x++ {
					if near(img.RGBAAt(x, y), background) {
						full = false
						break
					}
				}
				if full {
					rows++
				}
			}
			want := map[string]int{"light": stroke, "heavy": heavyStroke, "double": 2 * stroke}[tc.name]
			if rows != want {
				t.Errorf("%d pixel rows run through all cells, want %d", rows, want)
			}
		})
	}
	// A vertical rule over three rows meets without gaps.
	img := mustImage(t, frameOf("│\n┃\n║", 1, 3))
	for y := titleBar; y < titleBar+3*cellH; y++ {
		ink := false
		for x := padX; x < padX+cellW; x++ {
			if !near(img.RGBAAt(x, y), background) {
				ink = true
			}
		}
		if !ink {
			t.Fatalf("vertical rule has a gap at pixel row %d", y)
		}
	}
}

// TestImage_Corners: a corner meets its neighbours at the cell midpoints
// and leaves the far corner empty; rounded, heavy and double alike.
func TestImage_Corners(t *testing.T) {
	cases := []struct {
		ch           string
		drawn, empty [][2]int
	}{
		{"┌", [][2]int{{7, 4}, {4, 7}}, [][2]int{{1, 1}, {1, 4}, {4, 1}}},
		{"┘", [][2]int{{1, 4}, {4, 1}}, [][2]int{{7, 7}, {7, 4}, {4, 7}}},
		{"╭", [][2]int{{7, 4}, {4, 7}}, [][2]int{{1, 1}, {4, 4}, {1, 4}, {4, 1}}},
		{"╯", [][2]int{{1, 4}, {4, 1}}, [][2]int{{7, 7}, {4, 4}, {7, 4}, {4, 7}}},
		{"┏", [][2]int{{7, 4}, {4, 7}}, [][2]int{{1, 1}, {1, 4}, {4, 1}}},
		{"╝", [][2]int{{1, 4}, {4, 1}}, [][2]int{{7, 7}, {7, 4}, {4, 7}}},
	}
	for _, c := range cases {
		t.Run(c.ch, func(t *testing.T) {
			img := mustImage(t, frameOf(c.ch, 1, 1))
			for _, p := range c.drawn {
				ink := false
				// Double lines are two strokes beside the midpoint.
				for d := -stroke * 2; d <= stroke*2; d++ {
					x, y := padX+p[0]*cellW/8, titleBar+p[1]*cellH/8
					if p[0] == 4 {
						x += d
					} else {
						y += d
					}
					if near(img.RGBAAt(x, y), foreground) {
						ink = true
					}
				}
				if !ink {
					t.Errorf("%s: no line near %v", c.ch, p)
				}
			}
			for _, p := range c.empty {
				if got := pixelAt(img, 0, 0, p[0], p[1]); !near(got, background) {
					t.Errorf("%s at %v = %v, want the background", c.ch, p, got)
				}
			}
		})
	}
}

// TestPNG_RealFrameIsDeterministic: a real shot encodes as a PNG of the
// expected size, byte for byte the same on every run.
func TestPNG_RealFrameIsDeterministic(t *testing.T) {
	f := renderShot(t, "grid")
	var a, b bytes.Buffer
	if err := PNG(&a, f); err != nil {
		t.Fatal(err)
	}
	if err := PNG(&b, f); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("PNG differs between runs")
	}
	cfg, err := png.DecodeConfig(&a)
	if err != nil {
		t.Fatal(err)
	}
	if w, h := imageSize(Width, Height); cfg.Width != w || cfg.Height != h {
		t.Errorf("png is %dx%d, want %dx%d", cfg.Width, cfg.Height, w, h)
	}
}

// TestImage_ReturnArrow: ↵ is drawn as a shape: a stem down the right of
// the cell and a shaft to the left at mid height, the top left empty.
func TestImage_ReturnArrow(t *testing.T) {
	img := mustImage(t, frameOf("↵", 1, 1))
	if !isShape('↵') {
		t.Fatal("↵ is not a shape")
	}
	if c := pixelAt(img, 0, 0, 1, 1); !near(c, background) {
		t.Errorf("top left = %v, want the background", c)
	}
	if c := pixelAt(img, 0, 0, 4, 4); !near(c, foreground) {
		t.Errorf("shaft at the centre = %v, want the foreground", c)
	}
}
