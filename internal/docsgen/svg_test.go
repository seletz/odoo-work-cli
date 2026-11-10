package docsgen

import (
	"encoding/xml"
	"fmt"
	"strings"
	"testing"
)

func frameOf(content string, w, h int) Frame {
	return Frame{Shot: Shot{Name: "t", Width: w, Height: h}, Content: content}
}

func TestSVG_IsWellFormedXML(t *testing.T) {
	out := string(SVG(frameOf("a < b & c > \"d\" 'e'", 30, 2)))
	d := xml.NewDecoder(strings.NewReader(out))
	for {
		_, err := d.Token()
		if err != nil {
			if err.Error() != "EOF" {
				t.Fatalf("not well-formed: %v\n%s", err, out)
			}
			break
		}
	}
	if !strings.Contains(out, "a &lt; b &amp; c &gt;") {
		t.Errorf("text not escaped:\n%s", out)
	}
}

func TestSVG_SizeFollowsTerminalCells(t *testing.T) {
	out := string(SVG(frameOf("x", 80, 24)))
	w, h := svgSize(80, 24)
	if !strings.Contains(out, fmt.Sprintf(`viewBox="0 0 %d %d"`, w, h)) {
		t.Errorf("viewBox of 80x24 should be %dx%d:\n%s", w, h, out[:200])
	}
}

func TestSVG_ColoursAndBold(t *testing.T) {
	// Bold red on a dark green background.
	out := string(SVG(frameOf("\x1b[1;38;2;255;95;95;48;2;26;42;26mHi\x1b[m", 10, 1)))
	if !strings.Contains(out, `fill="#ff5f5f"`) {
		t.Errorf("foreground colour missing:\n%s", out)
	}
	if !strings.Contains(out, `font-weight="bold"`) {
		t.Errorf("bold missing:\n%s", out)
	}
	if !strings.Contains(out, `fill="#1a2a1a"`) {
		t.Errorf("background rect missing:\n%s", out)
	}
	if !strings.Contains(out, `textLength="`) {
		t.Errorf("text runs must pin their width to the cell grid:\n%s", out)
	}
}

func TestSVG_BoxDrawingAsLines(t *testing.T) {
	out := string(SVG(frameOf("┌─┐\n│ │\n└─┘", 3, 3)))
	if strings.ContainsAny(out, "┌─┐│└┘") {
		t.Errorf("box-drawing glyphs should be drawn as paths, not text:\n%s", out)
	}
	if !strings.Contains(out, "<path") {
		t.Errorf("no path for the box:\n%s", out)
	}
}

func TestSVG_IsDeterministic(t *testing.T) {
	f := frameOf("\x1b[38;5;5mA\x1b[m ─ \x1b[2mfaint\x1b[m", 20, 1)
	a, b := SVG(f), SVG(f)
	if string(a) != string(b) {
		t.Fatal("SVG differs between runs")
	}
}
