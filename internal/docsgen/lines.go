package docsgen

import (
	"image"
	"image/color"
	"math"
)

// Line weights of a box-drawing arm.
const (
	none = iota
	light
	heavy
	double
)

// arms describes a box-drawing glyph by the weight of its four arms from
// the cell centre: left, right, up, down. round marks the rounded corners.
type arms struct {
	l, r, u, d int
	round      bool
}

var boxGlyphs = map[string]arms{
	"─": {l: light, r: light}, "│": {u: light, d: light},
	"┌": {r: light, d: light}, "┐": {l: light, d: light},
	"└": {r: light, u: light}, "┘": {l: light, u: light},
	"├": {r: light, u: light, d: light}, "┤": {l: light, u: light, d: light},
	"┬": {l: light, r: light, d: light}, "┴": {l: light, r: light, u: light},
	"┼": {l: light, r: light, u: light, d: light},
	"╭": {r: light, d: light, round: true}, "╮": {l: light, d: light, round: true},
	"╰": {r: light, u: light, round: true}, "╯": {l: light, u: light, round: true},
	"━": {l: heavy, r: heavy}, "┃": {u: heavy, d: heavy},
	"┏": {r: heavy, d: heavy}, "┓": {l: heavy, d: heavy},
	"┗": {r: heavy, u: heavy}, "┛": {l: heavy, u: heavy},
	"═": {l: double, r: double}, "║": {u: double, d: double},
	"╔": {r: double, d: double}, "╗": {l: double, d: double},
	"╚": {r: double, u: double}, "╝": {l: double, u: double},
}

// boxShape draws the box-drawing glyph ch into the cell r and reports
// whether ch was one. The geometry is Ghostty's (src/font/sprite/draw/
// box.zig, linesChar and arc): light strokes are cell.box thick, heavy
// ones twice that, centred in the cell; a double arm is two light
// strokes with a light stroke's gap between them; arms meet in a square
// so corners and tees close; rounded corners are a quarter circle of half
// the cell width.
func boxShape(img *image.RGBA, r image.Rectangle, ch string, fg color.Color) bool {
	a, ok := boxGlyphs[ch]
	if !ok {
		return false
	}
	w, h := r.Dx(), r.Dy()
	lightPx, heavyPx := cell.box, 2*cell.box
	hLightTop := (h - lightPx) / 2
	hLightBottom := hLightTop + lightPx
	hHeavyTop := (h - heavyPx) / 2
	hHeavyBottom := hHeavyTop + heavyPx
	hDoubleTop := hLightTop - lightPx
	hDoubleBottom := hLightBottom + lightPx
	vLightLeft := (w - lightPx) / 2
	vLightRight := vLightLeft + lightPx
	vHeavyLeft := (w - heavyPx) / 2
	vHeavyRight := vHeavyLeft + heavyPx
	vDoubleLeft := vLightLeft - lightPx
	vDoubleRight := vLightRight + lightPx

	box := func(x0, y0, x1, y1 int) {
		fill(img, image.Rect(r.Min.X+x0, r.Min.Y+y0, r.Min.X+x1, r.Min.Y+y1), fg)
	}
	if a.round {
		roundCorner(img, r, a, lightPx, fg)
		return true
	}

	// Where the vertical arms end and the horizontal arms start, so the
	// arms join without a notch (linesChar's up_bottom, down_top,
	// left_right, right_left).
	upBottom := pick(a.l == heavy || a.r == heavy, hHeavyBottom,
		pick(a.l != a.r || a.d == a.u, pick(a.l == double || a.r == double, hDoubleBottom, hLightBottom),
			pick(a.l == none && a.r == none, hLightBottom, hLightTop)))
	downTop := pick(a.l == heavy || a.r == heavy, hHeavyTop,
		pick(a.l != a.r || a.u == a.d, pick(a.l == double || a.r == double, hDoubleTop, hLightTop),
			pick(a.l == none && a.r == none, hLightTop, hLightBottom)))
	leftRight := pick(a.u == heavy || a.d == heavy, vHeavyRight,
		pick(a.u != a.d || a.l == a.r, pick(a.u == double || a.d == double, vDoubleRight, vLightRight),
			pick(a.u == none && a.d == none, vLightRight, vLightLeft)))
	rightLeft := pick(a.u == heavy || a.d == heavy, vHeavyLeft,
		pick(a.u != a.d || a.r == a.l, pick(a.u == double || a.d == double, vDoubleLeft, vLightLeft),
			pick(a.u == none && a.d == none, vLightLeft, vLightRight)))

	switch a.u {
	case light:
		box(vLightLeft, 0, vLightRight, upBottom)
	case heavy:
		box(vHeavyLeft, 0, vHeavyRight, upBottom)
	case double:
		box(vDoubleLeft, 0, vLightLeft, pick(a.l == double, hLightTop, upBottom))
		box(vLightRight, 0, vDoubleRight, pick(a.r == double, hLightTop, upBottom))
	}
	switch a.r {
	case light:
		box(rightLeft, hLightTop, w, hLightBottom)
	case heavy:
		box(rightLeft, hHeavyTop, w, hHeavyBottom)
	case double:
		box(pick(a.u == double, vLightRight, rightLeft), hDoubleTop, w, hLightTop)
		box(pick(a.d == double, vLightRight, rightLeft), hLightBottom, w, hDoubleBottom)
	}
	switch a.d {
	case light:
		box(vLightLeft, downTop, vLightRight, h)
	case heavy:
		box(vHeavyLeft, downTop, vHeavyRight, h)
	case double:
		box(vDoubleLeft, pick(a.l == double, hLightBottom, downTop), vLightLeft, h)
		box(vLightRight, pick(a.r == double, hLightBottom, downTop), vDoubleRight, h)
	}
	switch a.l {
	case light:
		box(0, hLightTop, leftRight, hLightBottom)
	case heavy:
		box(0, hHeavyTop, leftRight, hHeavyBottom)
	case double:
		box(0, hDoubleTop, pick(a.u == double, vLightLeft, leftRight), hLightTop)
		box(0, hLightBottom, pick(a.d == double, vLightLeft, leftRight), hDoubleBottom)
	}
	return true
}

// pick is a conditional expression.
func pick(cond bool, a, b int) int {
	if cond {
		return a
	}
	return b
}

// roundCorner draws a rounded corner (╭╮╰╯) with soft edges: the strokes
// are centred as in Ghostty's arc; the quarter circle's radius is the
// distance from the vertical stroke to the cell edge on the side of the
// horizontal arm, so the arc ends exactly on that edge and joins the ─ of
// the next cell; the vertical arm runs on straight to its edge.
func roundCorner(img *image.RGBA, r image.Rectangle, a arms, thick int, fg color.Color) {
	w, h := r.Dx(), r.Dy()
	t := float64(thick)
	// Centre of the strokes, as in Ghostty's arc.
	cx := float64(r.Min.X) + float64((w-thick)/2) + t/2
	cy := float64(r.Min.Y) + float64((h-thick)/2) + t/2
	sx, sy := 1.0, 1.0 // direction of the horizontal and vertical arm
	rad := float64(r.Max.X) - cx
	if a.l != none {
		sx = -1
		rad = cx - float64(r.Min.X)
	}
	if a.u != none {
		sy = -1
	}
	// Centre of the arc; the products are rounded on their own (no FMA).
	ox, oy := cx+float64(sx*rad), cy+float64(sy*rad)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			var d float64
			switch {
			case (py-oy)*sy >= 0: // past the arc: the vertical arm
				d = math.Abs(px - cx)
			case (px-ox)*sx >= 0: // the horizontal arm
				d = math.Abs(py - cy)
			default:
				d = math.Abs(hypot(px-ox, py-oy) - rad)
			}
			if cov := t/2 + 0.5 - d; cov > 0 {
				img.Set(x, y, blend(fg, img.RGBAAt(x, y), math.Min(cov, 1)))
			}
		}
	}
}

// blockShape draws the block elements █ ▀ ▄ ▌ ▐ as filled parts of the
// cell r, as Ghostty does (src/font/sprite/draw/block.zig), and reports
// whether ch was one.
func blockShape(img *image.RGBA, r image.Rectangle, ch string, fg color.Color) bool {
	mx, my := r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2
	var part image.Rectangle
	switch ch {
	case "█":
		part = r
	case "▀":
		part = image.Rect(r.Min.X, r.Min.Y, r.Max.X, my)
	case "▄":
		part = image.Rect(r.Min.X, my, r.Max.X, r.Max.Y)
	case "▌":
		part = image.Rect(r.Min.X, r.Min.Y, mx, r.Max.Y)
	case "▐":
		part = image.Rect(mx, r.Min.Y, r.Max.X, r.Max.Y)
	default:
		return false
	}
	fill(img, part, fg)
	return true
}
