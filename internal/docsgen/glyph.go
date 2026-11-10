package docsgen

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

// glyphFace draws the glyphs of one font at fontSize.
//
// It rasterizes outlines itself instead of using font.Drawer: the curve
// flattening of x/image/vector computes a + t*(b-a) in float32, which
// arm64 fuses into one FMA instruction and amd64 does not, so the same
// screenshot came out a few pixels different on a Mac and in CI. Here the
// curves are flattened with every product rounded on its own (an
// explicit float32 conversion forbids the fusion), and the rasterizer only
// gets straight lines, whose fixed-point path has no such expression.
type glyphFace struct {
	f       *sfnt.Font
	ppem    fixed.Int26_6
	metrics font.Metrics

	mu    sync.Mutex
	buf   sfnt.Buffer
	cache map[rune]glyphMask
}

// glyphMask is the coverage of one glyph and its offset from the dot.
type glyphMask struct {
	mask    *image.Alpha
	advance int
}

func newGlyphFace(ttf []byte) (*glyphFace, error) {
	f, err := sfnt.Parse(ttf)
	if err != nil {
		return nil, err
	}
	g := &glyphFace{f: f, ppem: fixed.I(fontSize), cache: map[rune]glyphMask{}}
	g.metrics, err = f.Metrics(&g.buf, g.ppem, font.HintingFull)
	if err != nil {
		return nil, err
	}
	return g, nil
}

// has reports whether the font has a glyph for r.
func (g *glyphFace) has(r rune) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	idx, err := g.f.GlyphIndex(&g.buf, r)
	return err == nil && idx != 0
}

// draw draws s with its first glyph's dot at (x, y) in colour c.
func (g *glyphFace) draw(img *image.RGBA, x, y int, s string, c color.Color) error {
	src := image.NewUniform(c)
	for _, r := range s {
		m, err := g.glyph(r)
		if err != nil {
			return err
		}
		if m.mask != nil {
			b := m.mask.Bounds().Add(image.Pt(x, y))
			draw.DrawMask(img, b, src, image.Point{}, m.mask, m.mask.Bounds().Min, draw.Over)
		}
		x += m.advance
	}
	return nil
}

// glyph returns the cached mask of r, rasterizing it on first use. The
// mask's bounds are relative to the dot.
func (g *glyphFace) glyph(r rune) (glyphMask, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if m, ok := g.cache[r]; ok {
		return m, nil
	}
	idx, err := g.f.GlyphIndex(&g.buf, r)
	if err != nil {
		return glyphMask{}, err
	}
	adv, err := g.f.GlyphAdvance(&g.buf, idx, g.ppem, font.HintingFull)
	if err != nil {
		return glyphMask{}, err
	}
	segs, err := g.f.LoadGlyph(&g.buf, idx, g.ppem, nil)
	if err != nil {
		return glyphMask{}, err
	}
	m := glyphMask{advance: adv.Round()}
	fb := segs.Bounds()
	bounds := image.Rect(fb.Min.X.Floor(), fb.Min.Y.Floor(), fb.Max.X.Ceil(), fb.Max.Y.Ceil())
	if !bounds.Empty() {
		m.mask = rasterize(segs, bounds)
	}
	g.cache[r] = m
	return m, nil
}

// rasterize fills the outline segs into a mask of the given bounds (in
// pixels, relative to the dot).
func rasterize(segs sfnt.Segments, bounds image.Rectangle) *image.Alpha {
	var z vector.Rasterizer
	z.Reset(bounds.Dx(), bounds.Dy())
	// Fixed 26.6 to float32 relative to the mask is exact: small integers
	// divided by a power of two.
	ox, oy := fixed.I(bounds.Min.X), fixed.I(bounds.Min.Y)
	pt := func(p fixed.Point26_6) (float32, float32) {
		return float32(p.X-ox) / 64, float32(p.Y-oy) / 64
	}
	var px, py float32
	for _, s := range segs {
		switch s.Op {
		case sfnt.SegmentOpMoveTo:
			px, py = pt(s.Args[0])
			z.MoveTo(px, py)
		case sfnt.SegmentOpLineTo:
			px, py = pt(s.Args[0])
			z.LineTo(px, py)
		case sfnt.SegmentOpQuadTo:
			bx, by := pt(s.Args[0])
			cx, cy := pt(s.Args[1])
			n := steps(devSquared(px, py, bx, by, cx, cy))
			for i := 1; i < n; i++ {
				t := float32(i) / float32(n)
				abx, aby := lerp(t, px, py, bx, by)
				bcx, bcy := lerp(t, bx, by, cx, cy)
				z.LineTo(lerp(t, abx, aby, bcx, bcy))
			}
			px, py = cx, cy
			z.LineTo(px, py)
		case sfnt.SegmentOpCubeTo:
			bx, by := pt(s.Args[0])
			cx, cy := pt(s.Args[1])
			dx, dy := pt(s.Args[2])
			n := steps(max(devSquared(px, py, bx, by, dx, dy), devSquared(px, py, cx, cy, dx, dy)))
			for i := 1; i < n; i++ {
				t := float32(i) / float32(n)
				abx, aby := lerp(t, px, py, bx, by)
				bcx, bcy := lerp(t, bx, by, cx, cy)
				cdx, cdy := lerp(t, cx, cy, dx, dy)
				abcx, abcy := lerp(t, abx, aby, bcx, bcy)
				bcdx, bcdy := lerp(t, bcx, bcy, cdx, cdy)
				z.LineTo(lerp(t, abcx, abcy, bcdx, bcdy))
			}
			px, py = dx, dy
			z.LineTo(px, py)
		}
	}
	z.ClosePath()
	mask := image.NewAlpha(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	z.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})
	mask.Rect = bounds // same pixels, addressed relative to the dot
	return mask
}

// lerp interpolates between p and q; the product is rounded on its own.
func lerp(t, px, py, qx, qy float32) (float32, float32) {
	return px + float32(t*(qx-px)), py + float32(t*(qy-py))
}

// devSquared measures how far the control point b pulls the curve from
// the chord a–c; the products are rounded on their own.
func devSquared(ax, ay, bx, by, cx, cy float32) float32 {
	dx := ax - 2*bx + cx
	dy := ay - 2*by + cy
	return float32(dx*dx) + float32(dy*dy)
}

// steps is the number of line segments for a curve of the given
// deviation, as in x/image/vector.
func steps(devsq float32) int {
	if devsq < 0.333 {
		return 1
	}
	const tol = 3
	return 1 + int(math.Sqrt(math.Sqrt(tol*float64(devsq))))
}

// isShape reports whether r is drawn by shape: characters the TUI uses
// that Fira Code has no glyph for.
func isShape(r rune) bool { return r == '↵' }

// shape draws the characters of isShape into the cell r and reports
// whether ch was one of them.
func shape(img *image.RGBA, r image.Rectangle, ch string, fg color.Color) bool {
	if ch != "↵" {
		return false
	}
	w, h := r.Dx(), r.Dy()
	cy := r.Min.Y + h/2
	stem := r.Min.X + w*3/4
	tip := r.Min.X + w/8
	head := w / 4 // length and half height of the arrow head
	// Stem down the right, shaft to the left at mid height.
	fill(img, image.Rect(stem-cell.box/2, r.Min.Y+h/4, stem-cell.box/2+cell.box, cy+cell.box-cell.box/2), fg)
	fill(img, image.Rect(tip+head/2, cy-cell.box/2, stem-cell.box/2+cell.box, cy-cell.box/2+cell.box), fg)
	// Arrow head: a triangle pointing left, its tip at tip.
	for x := 0; x <= head; x++ {
		fill(img, image.Rect(tip+x, cy-x/2-1, tip+x+1, cy+x/2+1), fg)
	}
	return true
}
