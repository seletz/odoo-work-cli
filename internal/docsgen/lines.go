package docsgen

import (
	"bytes"
	"fmt"
	"strings"
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

func isLine(s string) bool {
	_, ok := boxGlyphs[s]
	return ok
}

// linePaths collects the path segments of box-drawing cells, one path per
// colour and stroke width, in first-seen order.
type linePaths struct {
	order []string
	paths map[string]*strings.Builder
}

func (p *linePaths) builder(fg string, width float64) *strings.Builder {
	key := fmt.Sprintf(`stroke="%s" stroke-width="%g"`, fg, width)
	if p.paths == nil {
		p.paths = map[string]*strings.Builder{}
	}
	sb, ok := p.paths[key]
	if !ok {
		sb = &strings.Builder{}
		p.paths[key] = sb
		p.order = append(p.order, key)
	}
	return sb
}

// add draws c at cell x, y if it is a box-drawing glyph and reports
// whether it was one.
func (p *linePaths) add(c cellStyle, x, y int) bool {
	a, ok := boxGlyphs[c.text]
	if !ok {
		return false
	}
	x0, y0 := float64(padX+x*cellW), float64(titleBar+y*cellH)
	cx, cy := x0+cellW/2, y0+cellH/2
	x1, y1 := x0+cellW, y0+cellH
	weight := max(a.l, a.r, a.u, a.d)

	switch {
	case weight == double:
		sb := p.builder(c.fg, 1)
		// Two parallel strokes, 2 units either side of the centre line.
		const o = 2.0
		hs, vs := a.l != none || a.r != none, a.u != none || a.d != none
		switch {
		case hs && !vs:
			fmt.Fprintf(sb, "M%g %gH%g M%g %gH%g ", x0, cy-o, x1, x0, cy+o, x1)
		case vs && !hs:
			fmt.Fprintf(sb, "M%g %gV%g M%g %gV%g ", cx-o, y0, y1, cx+o, y0, y1)
		default:
			// A corner: h and v point to the arms; outer and inner L.
			h, hx := 1.0, x1
			if a.l != none {
				h, hx = -1, x0
			}
			v, vy := 1.0, y1
			if a.u != none {
				v, vy = -1, y0
			}
			for _, s := range []float64{-o, o} {
				fmt.Fprintf(sb, "M%g %gL%g %gL%g %g ", hx, cy+s*v, cx+s*h, cy+s*v, cx+s*h, vy)
			}
		}
	case a.round:
		sb := p.builder(c.fg, 1)
		hx, vy := x1, y1
		if a.l != none {
			hx = x0
		}
		if a.u != none {
			vy = y0
		}
		// Quadratic curve through the centre gives the rounded corner.
		fmt.Fprintf(sb, "M%g %gL%g %gQ%g %g %g %gL%g %g ",
			hx, cy, (hx+cx)/2, cy, cx, cy, cx, (vy+cy)/2, cx, vy)
	default:
		sb := p.builder(c.fg, map[int]float64{light: 1, heavy: 2.5}[weight])
		if a.l != none {
			fmt.Fprintf(sb, "M%g %gH%g ", x0, cy, cx)
		}
		if a.r != none {
			fmt.Fprintf(sb, "M%g %gH%g ", cx, cy, x1)
		}
		if a.u != none {
			fmt.Fprintf(sb, "M%g %gV%g ", cx, y0, cy)
		}
		if a.d != none {
			fmt.Fprintf(sb, "M%g %gV%g ", cx, cy, y1)
		}
	}
	return true
}

func (p *linePaths) write(b *bytes.Buffer) {
	for _, key := range p.order {
		d := strings.TrimSpace(p.paths[key].String())
		fmt.Fprintf(b, `<path d="%s" fill="none" %s stroke-linecap="square"/>`+"\n", d, key)
	}
}
