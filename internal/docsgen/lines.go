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

// boxShape draws the box-drawing glyph ch into the cell r as lines that
// meet the neighbouring cells at their edge midpoints, and reports whether
// ch was one. Light and heavy arms are strokes of their thickness, double
// arms two light strokes either side of the centre line, rounded corners
// a quarter circle.
func boxShape(img *image.RGBA, r image.Rectangle, ch string, fg color.Color) bool {
	a, ok := boxGlyphs[ch]
	if !ok {
		return false
	}
	cx, cy := r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2
	// hband and vband fill a horizontal or vertical stroke of width sw
	// centred off pixels below/right of the centre line.
	hband := func(x0, x1, off, sw int) {
		fill(img, image.Rect(min(x0, x1), cy+off-sw/2, max(x0, x1), cy+off-sw/2+sw), fg)
	}
	vband := func(y0, y1, off, sw int) {
		fill(img, image.Rect(cx+off-sw/2, min(y0, y1), cx+off-sw/2+sw, max(y0, y1)), fg)
	}
	weight := max(a.l, a.r, a.u, a.d)
	switch {
	case a.round:
		rad := min(r.Dx(), r.Dy()) / 2
		h, hx := 1, r.Max.X
		if a.l != none {
			h, hx = -1, r.Min.X
		}
		v, vy := 1, r.Max.Y
		if a.u != none {
			v, vy = -1, r.Min.Y
		}
		ox, oy := cx+h*rad, cy+v*rad
		arc(img, ox, oy, rad, -h, -v, fg)
		hband(ox, hx, 0, stroke)
		vband(oy, vy, 0, stroke)
	case weight == double:
		h, v := 0, 0 // direction of the horizontal and vertical arm, 0 for none
		if a.r != none {
			h = 1
		} else if a.l != none {
			h = -1
		}
		if a.d != none {
			v = 1
		} else if a.u != none {
			v = -1
		}
		switch {
		case v == 0:
			hband(r.Min.X, r.Max.X, -doubleGap, stroke)
			hband(r.Min.X, r.Max.X, doubleGap, stroke)
		case h == 0:
			vband(r.Min.Y, r.Max.Y, -doubleGap, stroke)
			vband(r.Min.Y, r.Max.Y, doubleGap, stroke)
		default:
			// A corner: an outer and an inner L, joined at their bends.
			hx, vy := r.Max.X, r.Max.Y
			if h < 0 {
				hx = r.Min.X
			}
			if v < 0 {
				vy = r.Min.Y
			}
			for _, s := range []int{-1, 1} {
				ex, ey := cx+s*doubleGap*h, cy+s*doubleGap*v
				hband(ex-h*(stroke/2), hx, s*doubleGap*v, stroke)
				vband(ey-v*(stroke/2), vy, s*doubleGap*h, stroke)
			}
		}
	default:
		sw := stroke
		if weight == heavy {
			sw = heavyStroke
		}
		// Each arm runs from the far side of the centre stroke to the edge,
		// so arms join in a square.
		if a.l != none {
			hband(r.Min.X, cx-sw/2+sw, 0, sw)
		}
		if a.r != none {
			hband(cx-sw/2, r.Max.X, 0, sw)
		}
		if a.u != none {
			vband(r.Min.Y, cy-sw/2+sw, 0, sw)
		}
		if a.d != none {
			vband(cy-sw/2, r.Max.Y, 0, sw)
		}
	}
	return true
}

// arc draws the quarter of a circle of radius rad around (ox, oy) that
// lies towards sx, sy (each -1 or +1), as a light stroke with soft edges.
func arc(img *image.RGBA, ox, oy, rad, sx, sy int, c color.Color) {
	for y := oy - rad - stroke; y <= oy+rad+stroke; y++ {
		for x := ox - rad - stroke; x <= ox+rad+stroke; x++ {
			dx, dy := float64(x)+0.5-float64(ox), float64(y)+0.5-float64(oy)
			if dx*float64(sx) < 0 || dy*float64(sy) < 0 {
				continue
			}
			d := math.Abs(hypot(dx, dy) - float64(rad))
			if t := float64(stroke)/2 + 0.5 - d; t > 0 {
				img.Set(x, y, blend(c, img.RGBAAt(x, y), math.Min(t, 1)))
			}
		}
	}
}
