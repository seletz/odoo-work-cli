package docsgen

import (
	_ "embed"
	"encoding/binary"
	"fmt"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// The screenshots are set in Fira Code (github.com/tonsky/FiraCode, SIL
// Open Font License 1.1, see fonts/OFL.txt), the font of the reference
// terminal: Ghostty with font-family = "Fira Code". Fira Code has no
// italic; the TUI uses none.
var (
	//go:embed fonts/FiraCode-Regular.ttf
	firaRegular []byte
	//go:embed fonts/FiraCode-Bold.ttf
	firaBold []byte
)

// fontSize is the font size in pixels: Ghostty's default 13 pt on a 2×
// (Retina) display, so the images stay sharp on high-resolution screens.
const fontSize = 26

// faceMetrics are the unrounded metrics of a face at a pixel size, as
// Ghostty's FaceMetrics: vertical values relative to the baseline, +y up.
type faceMetrics struct {
	width   float64 // widest advance of the printable ASCII glyphs
	ascent  float64
	descent float64 // usually negative
	lineGap float64
	ulPos   float64 // top of the underline stroke
	ulThick float64
}

// cellMetrics are the pixel metrics of one terminal cell. Unlike Ghostty,
// baseline and ulPos count from the top of the cell.
type cellMetrics struct {
	w, h     int
	baseline int // top of the cell to the baseline
	ulPos    int // top of the cell to the top of the underline
	ulThick  int
	box      int // thickness of light box-drawing lines
}

// cell is the cell of every screenshot, from the embedded regular face.
var cell = mustCell()

func mustCell() cellMetrics {
	fm, err := readFaceMetrics(firaRegular, fontSize)
	if err != nil {
		panic(fmt.Sprintf("docsgen: embedded font: %v", err))
	}
	return calcMetrics(fm)
}

// calcMetrics turns face metrics into cell metrics the way Ghostty does
// (src/font/Metrics.zig, calc): width and height are rounded; half the
// line gap goes above and half below the text; the baseline is placed so
// the face sits centred in the rounded cell height; the underline is at
// least 1 px thick and box-drawing lines take its thickness.
//
// :param f: the face metrics
// :returns: the cell metrics
func calcMetrics(f faceMetrics) cellMetrics {
	faceH := f.ascent - f.descent + f.lineGap
	w, h := math.Round(f.width), math.Round(faceH)
	faceBaseline := f.lineGap/2 - f.descent // from the bottom
	baseline := math.Round(faceBaseline - (h-faceH)/2)
	top := h - baseline
	ulThick := math.Max(1, math.Ceil(f.ulThick))
	return cellMetrics{
		w:        int(w),
		h:        int(h),
		baseline: int(top),
		ulPos:    int(math.Round(top - f.ulPos)),
		ulThick:  int(ulThick),
		box:      int(ulThick),
	}
}

// readFaceMetrics reads the metrics of a TrueType font at px pixels per
// em, as Ghostty's CoreText face does (src/font/face/coretext.zig,
// getMetrics): the vertical metrics come from OS/2 sTypo* when the font
// sets USE_TYPO_METRICS, otherwise from hhea, then OS/2 sTypo*, then
// usWin*; the underline from post; the width is the widest advance of the
// printable ASCII characters.
//
// :param ttf: the font file
// :param px: pixels per em
// :returns: the face metrics
func readFaceMetrics(ttf []byte, px float64) (faceMetrics, error) {
	f, err := sfnt.Parse(ttf)
	if err != nil {
		return faceMetrics{}, err
	}
	tables, err := tableDir(ttf)
	if err != nil {
		return faceMetrics{}, err
	}
	hhea, os2, post := tables["hhea"], tables["OS/2"], tables["post"]
	if len(hhea) < 10 || len(os2) < 78 || len(post) < 12 {
		return faceMetrics{}, fmt.Errorf("missing or short hhea, OS/2 or post table")
	}
	i16 := func(b []byte, off int) float64 { return float64(int16(binary.BigEndian.Uint16(b[off:]))) }
	u16 := func(b []byte, off int) float64 { return float64(binary.BigEndian.Uint16(b[off:])) }

	upem := float64(f.UnitsPerEm())
	scale := px / upem
	const useTypoMetrics = 1 << 7
	var asc, desc, gap float64
	switch {
	case binary.BigEndian.Uint16(os2[62:])&useTypoMetrics != 0:
		asc, desc, gap = i16(os2, 68), i16(os2, 70), i16(os2, 72)
	case i16(hhea, 4) != 0 || i16(hhea, 6) != 0:
		asc, desc, gap = i16(hhea, 4), i16(hhea, 6), i16(hhea, 8)
	case i16(os2, 68) != 0 || i16(os2, 70) != 0:
		asc, desc, gap = i16(os2, 68), i16(os2, 70), i16(os2, 72)
	default:
		asc, desc, gap = u16(os2, 74), -u16(os2, 76), 0
	}

	// Advances in font units: at ppem = units per em, 26.6 fixed point
	// gives them exactly, unhinted.
	var buf sfnt.Buffer
	var widest fixed.Int26_6
	for r := rune(0x20); r < 0x7f; r++ {
		idx, err := f.GlyphIndex(&buf, r)
		if err != nil || idx == 0 {
			continue
		}
		adv, err := f.GlyphAdvance(&buf, idx, fixed.I(int(upem)), font.HintingNone)
		if err != nil {
			return faceMetrics{}, err
		}
		widest = max(widest, adv)
	}
	return faceMetrics{
		width:   float64(widest) / 64 * scale,
		ascent:  asc * scale,
		descent: desc * scale,
		lineGap: gap * scale,
		ulPos:   i16(post, 8) * scale,
		ulThick: i16(post, 10) * scale,
	}, nil
}

// tableDir maps the tags of a TrueType font's tables to their bytes.
func tableDir(ttf []byte) (map[string][]byte, error) {
	if len(ttf) < 12 {
		return nil, fmt.Errorf("font too short")
	}
	n := int(binary.BigEndian.Uint16(ttf[4:]))
	out := make(map[string][]byte, n)
	for i := range n {
		rec := 12 + 16*i
		if rec+16 > len(ttf) {
			return nil, fmt.Errorf("table directory truncated")
		}
		off := int(binary.BigEndian.Uint32(ttf[rec+8:]))
		size := int(binary.BigEndian.Uint32(ttf[rec+12:]))
		if off+size > len(ttf) {
			return nil, fmt.Errorf("table %q out of bounds", ttf[rec:rec+4])
		}
		out[string(ttf[rec:rec+4])] = ttf[off : off+size]
	}
	return out, nil
}
