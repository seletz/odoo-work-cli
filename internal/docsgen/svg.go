package docsgen

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"image/color"

	uv "github.com/charmbracelet/ultraviolet"
)

// Geometry of the SVG screenshots in user units. Every text run carries
// its width (textLength), so columns line up whatever monospace font the
// browser picks.
const (
	cellW    = 10
	cellH    = 20
	fontSize = 16
	padX     = 18
	padY     = 14
	titleBar = 34 // room above the first row for the window buttons
	radius   = 10
)

// Window colours: a dark terminal with the TUI's default text colour.
const (
	bgColour     = "#161b22"
	fgColour     = "#d0d0d0"
	borderColour = "#30363d"
	fontFamily   = "ui-monospace, SFMono-Regular, Menlo, Consolas, 'DejaVu Sans Mono', monospace"
)

var buttonColours = []string{"#ff5f57", "#febc2e", "#28c840"}

// svgSize returns the size of the image of a cols × rows terminal.
func svgSize(cols, rows int) (int, int) {
	return cols*cellW + 2*padX, rows*cellH + titleBar + padY
}

// SVG draws a frame as a terminal window: a rounded dark panel with three
// window buttons and one cell per terminal cell. Background colours become
// rectangles, text runs of one style become <text> elements, and the
// box-drawing characters of the grid and overlays become paths, so lines
// join without gaps. The output depends only on the frame.
//
// :param f: the frame
// :returns: the SVG document
func SVG(f Frame) []byte {
	cols, rows := f.Shot.Width, f.Shot.Height
	buf := uv.NewScreenBuffer(cols, rows)
	uv.NewStyledString(f.Content).Draw(buf, buf.Bounds())
	w, h := svgSize(cols, rows)

	var b bytes.Buffer
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d" role="img">`+"\n", w, h, w, h)
	fmt.Fprintf(&b, `<rect x="0.5" y="0.5" width="%d" height="%d" rx="%d" fill="%s" stroke="%s"/>`+"\n", w-1, h-1, radius, bgColour, borderColour)
	for i, c := range buttonColours {
		fmt.Fprintf(&b, `<circle cx="%d" cy="%d" r="6" fill="%s"/>`+"\n", padX+6+i*20, titleBar/2, c)
	}

	cells := make([][]cellStyle, rows)
	for y := range rows {
		cells[y] = make([]cellStyle, cols)
		for x := range cols {
			cells[y][x] = styleAt(buf, x, y)
		}
	}
	writeBackgrounds(&b, cells)
	lines := &linePaths{}
	fmt.Fprintf(&b, `<g font-family="%s" font-size="%d" xml:space="preserve">`+"\n", fontFamily, fontSize)
	for y := range rows {
		writeRow(&b, lines, cells[y], y)
	}
	b.WriteString("</g>\n")
	lines.write(&b)
	b.WriteString("</svg>\n")
	return b.Bytes()
}

// cellStyle is the resolved look of one terminal cell.
type cellStyle struct {
	text      string
	width     int // cells covered; 0 for the tail of a wide character
	fg, bg    string
	bold      bool
	italic    bool
	underline bool
}

func (c cellStyle) sameText(o cellStyle) bool {
	return c.fg == o.fg && c.bold == o.bold && c.italic == o.italic && c.underline == o.underline
}

// styleAt resolves the colours of a cell: reverse swaps them, faint
// blends the foreground towards the background.
func styleAt(buf uv.ScreenBuffer, x, y int) cellStyle {
	cell := buf.CellAt(x, y)
	if cell == nil {
		return cellStyle{text: " ", width: 1, fg: fgColour}
	}
	s := cell.Style
	fg, bg := s.Fg, s.Bg
	if s.Attrs&uv.AttrReverse != 0 {
		fg, bg = orDefault(bg, bgColour), orDefault(fg, fgColour)
	}
	out := cellStyle{
		text: cell.Content, width: cell.Width,
		fg: hex(fg, fgColour), bg: hex(bg, ""),
		bold:      s.Attrs&uv.AttrBold != 0,
		italic:    s.Attrs&uv.AttrItalic != 0,
		underline: s.Underline != uv.UnderlineNone,
	}
	if out.text == "" && out.width > 0 {
		out.text = " "
	}
	if s.Attrs&uv.AttrFaint != 0 {
		out.fg = blend(out.fg, orString(out.bg, bgColour), 0.55)
	}
	return out
}

func orDefault(c color.Color, def string) color.Color {
	if c == nil {
		return parseHex(def)
	}
	return c
}

func orString(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// hex formats c as #rrggbb, or def when c is nil.
func hex(c color.Color, def string) string {
	if c == nil {
		return def
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

func parseHex(s string) color.Color {
	var r, g, b uint8
	_, _ = fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
	return color.RGBA{r, g, b, 0xff}
}

// blend mixes a over b: t = 1 is a, t = 0 is b.
func blend(a, b string, t float64) string {
	ca, cb := parseHex(a).(color.RGBA), parseHex(b).(color.RGBA)
	mix := func(x, y uint8) uint8 { return uint8(float64(x)*t + float64(y)*(1-t) + 0.5) }
	return fmt.Sprintf("#%02x%02x%02x", mix(ca.R, cb.R), mix(ca.G, cb.G), mix(ca.B, cb.B))
}

// writeBackgrounds draws one rectangle per run of equal background colour.
func writeBackgrounds(b *bytes.Buffer, cells [][]cellStyle) {
	for y, row := range cells {
		for x := 0; x < len(row); {
			bg := row[x].bg
			end := x + 1
			for end < len(row) && row[end].bg == bg {
				end++
			}
			if bg != "" {
				fmt.Fprintf(b, `<rect x="%d" y="%d" width="%d" height="%d" fill="%s"/>`+"\n",
					padX+x*cellW, titleBar+y*cellH, (end-x)*cellW, cellH, bg)
			}
			x = end
		}
	}
}

// writeRow writes the text runs of row y; box-drawing cells go to lines.
func writeRow(b *bytes.Buffer, lines *linePaths, row []cellStyle, y int) {
	baseline := titleBar + y*cellH + cellH*3/4
	for x := 0; x < len(row); {
		c := row[x]
		if c.width == 0 {
			x++
			continue
		}
		if lines.add(c, x, y) {
			x++
			continue
		}
		var text bytes.Buffer
		end, cellsCovered := x, 0
		for end < len(row) {
			n := row[end]
			if n.width == 0 {
				end++
				continue
			}
			if !n.sameText(c) || isLine(n.text) {
				break
			}
			text.WriteString(n.text)
			cellsCovered += n.width
			end++
		}
		if s := text.String(); !blank(s) || c.underline {
			fmt.Fprintf(b, `<text x="%d" y="%d" textLength="%d" lengthAdjust="spacingAndGlyphs" fill="%s"%s>`,
				padX+x*cellW, baseline, cellsCovered*cellW, c.fg, textAttrs(c))
			_ = xml.EscapeText(b, []byte(s))
			b.WriteString("</text>\n")
		}
		x = end
	}
}

func textAttrs(c cellStyle) string {
	var s string
	if c.bold {
		s += ` font-weight="bold"`
	}
	if c.italic {
		s += ` font-style="italic"`
	}
	if c.underline {
		s += ` text-decoration="underline"`
	}
	return s
}

func blank(s string) bool {
	for _, r := range s {
		if r != ' ' {
			return false
		}
	}
	return true
}

// WriteSVGs writes one <shot>.svg per frame into dir and removes stale ones.
//
// :param dir: the output directory (docs/assets/screenshots)
// :param frames: the rendered frames
// :returns: an error if writing fails
func WriteSVGs(dir string, frames []Frame) error {
	files := map[string][]byte{}
	for _, f := range frames {
		files[f.Shot.Name+".svg"] = SVG(f)
	}
	return writeDir(dir, ".svg", files)
}
