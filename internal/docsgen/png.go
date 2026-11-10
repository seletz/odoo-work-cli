package docsgen

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"sync"

	uv "github.com/charmbracelet/ultraviolet"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
)

// Geometry of the screenshots in pixels. The font is drawn at twice a
// typical terminal size, so the images stay sharp on high-resolution
// displays.
const (
	fontSize    = 26
	cellW       = 16 // advance of Go Mono at fontSize, rounded
	cellH       = 34
	padX        = 36
	padY        = 32
	titleBar    = 60 // room above the first row, holds the window buttons
	radius      = 22
	stroke      = 3 // thickness of light box-drawing lines
	heavyStroke = 6 // thickness of heavy box-drawing lines
	doubleGap   = 3 // distance of the two strokes of a double line from the centre line
)

// Window colours: a dark terminal with the TUI's default text colour.
var (
	background = color.RGBA{0x16, 0x1b, 0x22, 0xff}
	foreground = color.RGBA{0xd0, 0xd0, 0xd0, 0xff}
	border     = color.RGBA{0x30, 0x36, 0x3d, 0xff}
	buttons    = []color.RGBA{{0xff, 0x5f, 0x57, 0xff}, {0xfe, 0xbc, 0x2e, 0xff}, {0x28, 0xc8, 0x40, 0xff}}
)

// Indexes into faces.
const (
	faceRegular = iota
	faceBold
	faceItalic
	faceBoldItalic
)

// faces are the Go Mono faces, parsed once.
var faces = sync.OnceValues(func() ([4]*glyphFace, error) {
	var out [4]*glyphFace
	for i, ttf := range [][]byte{gomono.TTF, gomonobold.TTF, gomonoitalic.TTF, gomonobolditalic.TTF} {
		f, err := newGlyphFace(ttf)
		if err != nil {
			return out, fmt.Errorf("png: font: %w", err)
		}
		out[i] = f
	}
	return out, nil
})

// imageSize returns the size in pixels of the image of a cols × rows
// terminal.
func imageSize(cols, rows int) (int, int) {
	return cols*cellW + 2*padX, rows*cellH + titleBar + padY
}

// PNG writes the image of a frame as PNG; see Image.
//
// :param w: the destination
// :param f: the frame
// :returns: an error if drawing or encoding fails
func PNG(w io.Writer, f Frame) error {
	img, err := Image(f)
	if err != nil {
		return err
	}
	return png.Encode(w, img)
}

// Image draws a frame as a terminal window: a rounded dark panel with three
// window buttons and one cell per terminal cell. Text is set in the
// embedded Go Mono, so the image looks the same everywhere: bold and
// italic cells use those faces, faint ones blend the foreground towards
// the background, underlined ones get a line, reverse swaps the colours.
// The box-drawing glyphs of the grid and the overlays are drawn as
// shapes, so rules join without gaps. The output depends only on the
// frame.
//
// :param f: the frame
// :returns: the image
func Image(f Frame) (*image.RGBA, error) {
	fs, err := faces()
	if err != nil {
		return nil, err
	}
	cols, rows := f.Shot.Width, f.Shot.Height
	buf := uv.NewScreenBuffer(cols, rows)
	uv.NewStyledString(f.Content).Draw(buf, buf.Bounds())

	w, h := imageSize(cols, rows)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	fillRounded(img, img.Bounds(), radius, border)
	fillRounded(img, image.Rect(2, 2, w-2, h-2), radius-2, background)
	for i, c := range buttons {
		fillCircle(img, padX+8+i*26, titleBar/2, 8, c)
	}

	metrics := fs[faceRegular].metrics
	ascent := (cellH-(metrics.Ascent+metrics.Descent).Ceil())/2 + metrics.Ascent.Ceil()
	for y := range rows {
		for x := range cols {
			cell := buf.CellAt(x, y)
			if cell == nil || cell.Width == 0 || cell.Content == "" {
				continue
			}
			s := cell.Style
			fg, bg := colours(s)
			r := image.Rect(padX+x*cellW, titleBar+y*cellH, padX+(x+cell.Width)*cellW, titleBar+(y+1)*cellH)
			if bg != nil {
				fill(img, r, bg)
			}
			if fg == nil {
				fg = foreground
			}
			if s.Attrs&uv.AttrFaint != 0 {
				fg = blend(fg, bgOr(bg), 0.55)
			}
			if !boxShape(img, r, cell.Content, fg) && !shape(img, r, cell.Content, fg) {
				face := fs[faceRegular]
				switch bold, italic := s.Attrs&uv.AttrBold != 0, s.Attrs&uv.AttrItalic != 0; {
				case bold && italic:
					face = fs[faceBoldItalic]
				case bold:
					face = fs[faceBold]
				case italic:
					face = fs[faceItalic]
				}
				if err := face.draw(img, r.Min.X, r.Min.Y+ascent, cell.Content, fg); err != nil {
					return nil, fmt.Errorf("png: %q: %w", cell.Content, err)
				}
			}
			if s.Underline != uv.UnderlineNone {
				fill(img, image.Rect(r.Min.X, r.Min.Y+ascent+4, r.Max.X, r.Min.Y+ascent+6), fg)
			}
		}
	}
	return img, nil
}

// colours resolves the foreground and background of a cell: nil means the
// default; reverse swaps them.
func colours(s uv.Style) (fg, bg color.Color) {
	fg, bg = s.Fg, s.Bg
	if s.Attrs&uv.AttrReverse != 0 {
		if fg == nil {
			fg = foreground
		}
		if bg == nil {
			bg = background
		}
		fg, bg = bg, fg
	}
	return fg, bg
}

func bgOr(bg color.Color) color.Color {
	if bg == nil {
		return background
	}
	return bg
}

// WritePNGs writes one <shot>.png per frame into dir and removes stale ones.
//
// :param dir: the output directory (docs/assets/screenshots)
// :param frames: the rendered frames
// :returns: an error if drawing or writing fails
func WritePNGs(dir string, frames []Frame) error {
	files := map[string][]byte{}
	for _, f := range frames {
		var b bytes.Buffer
		if err := PNG(&b, f); err != nil {
			return fmt.Errorf("shot %s: %w", f.Shot.Name, err)
		}
		files[f.Shot.Name+".png"] = b.Bytes()
	}
	return writeDir(dir, ".png", files)
}

func fill(img *image.RGBA, r image.Rectangle, c color.Color) {
	draw.Draw(img, r, image.NewUniform(c), image.Point{}, draw.Over)
}

// fillRounded fills r with c, leaving the corners outside a radius empty.
func fillRounded(img *image.RGBA, r image.Rectangle, rad int, c color.Color) {
	mask := image.NewAlpha(r)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			mask.SetAlpha(x, y, color.Alpha{A: uint8(255 * coverage(x, y, r, rad))})
		}
	}
	draw.DrawMask(img, r, image.NewUniform(c), image.Point{}, mask, r.Min, draw.Over)
}

// coverage is how much of the pixel at x, y lies inside the rounded
// rectangle r: 1 away from the corners, falling off over one pixel at the
// corner arcs.
func coverage(x, y int, r image.Rectangle, rad int) float64 {
	cx := math.Min(math.Max(float64(x)+0.5, float64(r.Min.X+rad)), float64(r.Max.X-rad))
	cy := math.Min(math.Max(float64(y)+0.5, float64(r.Min.Y+rad)), float64(r.Max.Y-rad))
	d := hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
	return math.Min(math.Max(float64(rad)+0.5-d, 0), 1)
}

// fillCircle draws a filled circle of radius rad around x, y.
func fillCircle(img *image.RGBA, x, y, rad int, c color.Color) {
	r := image.Rect(x-rad, y-rad, x+rad, y+rad)
	mask := image.NewAlpha(r)
	for py := r.Min.Y; py < r.Max.Y; py++ {
		for px := r.Min.X; px < r.Max.X; px++ {
			d := hypot(float64(px)+0.5-float64(x), float64(py)+0.5-float64(y))
			mask.SetAlpha(px, py, color.Alpha{A: uint8(255 * math.Min(math.Max(float64(rad)+0.5-d, 0), 1))})
		}
	}
	draw.DrawMask(img, r, image.NewUniform(c), image.Point{}, mask, r.Min, draw.Over)
}

// blend mixes a over b: t = 1 is a, t = 0 is b.
func blend(a, b color.Color, t float64) color.Color {
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	mix := func(x, y uint32) uint8 { return uint8(float64(float64(x>>8)*t) + float64(float64(y>>8)*(1-t)) + 0.5) }
	return color.RGBA{mix(ar, br), mix(ag, bg), mix(ab, bb), 0xff}
}

// hypot is sqrt(x² + y²) with every product rounded on its own. The
// explicit conversions keep the compiler from fusing them into FMA
// instructions (arm64 does, amd64 does not), so the images come out byte
// for byte the same on every platform and the stale check can compare
// them.
func hypot(x, y float64) float64 {
	return math.Sqrt(float64(x*x) + float64(y*y))
}
