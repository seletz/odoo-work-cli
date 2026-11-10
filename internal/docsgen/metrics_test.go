package docsgen

import (
	"math"
	"testing"
)

// TestCalcMetrics follows Ghostty's Metrics.calc: width and height are
// rounded, the baseline centres the face in the rounded cell, the
// underline sits below the baseline by its position, at least 1 px thick,
// and box lines take the underline thickness.
func TestCalcMetrics(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   faceMetrics
		want cellMetrics
	}{
		{
			name: "exact",
			in:   faceMetrics{width: 16, ascent: 24, descent: -8, lineGap: 0, ulPos: -4.0 / 3, ulThick: 2.0 / 3},
			want: cellMetrics{w: 16, h: 32, baseline: 24, ulPos: 25, ulThick: 1, box: 1},
		},
		{
			name: "rounded, line gap split",
			in:   faceMetrics{width: 7.6, ascent: 13.2, descent: -3.4, lineGap: 1.0, ulPos: -1.5, ulThick: 1.2},
			// face height 17.6 → 18; face baseline from bottom 0.5+3.4 = 3.9,
			// minus (18-17.6)/2 = 3.7 → 4; top to baseline 18-4 = 14.
			want: cellMetrics{w: 8, h: 18, baseline: 14, ulPos: 16, ulThick: 2, box: 2},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := calcMetrics(tc.in); got != tc.want {
				t.Errorf("calcMetrics(%+v) = %+v, want %+v", tc.in, got, tc.want)
			}
		})
	}
}

// TestFaceMetrics_FiraCode reads the vertical metrics of the embedded Fira
// Code from its hhea and post tables and the cell width from the widest
// ASCII advance, at Ghostty's default 13 pt on a 2× display (26 px).
func TestFaceMetrics_FiraCode(t *testing.T) {
	for _, ttf := range [][]byte{firaRegular, firaBold} {
		fm, err := readFaceMetrics(ttf, fontSize)
		if err != nil {
			t.Fatal(err)
		}
		// Fira Code: 1950 units per em, advance 1200, hhea 1800/-600/0,
		// post underline -100/50.
		want := faceMetrics{width: 16, ascent: 24, descent: -8, lineGap: 0, ulPos: -100 * 26.0 / 1950, ulThick: 50 * 26.0 / 1950}
		if !sameMetrics(fm, want) {
			t.Errorf("face metrics = %+v, want %+v", fm, want)
		}
	}
	if fontSize != 26 {
		t.Errorf("fontSize = %d, want 26 (13 pt at 2×)", fontSize)
	}
	want := cellMetrics{w: 16, h: 32, baseline: 24, ulPos: 25, ulThick: 1, box: 1}
	if cell != want {
		t.Errorf("cell = %+v, want %+v", cell, want)
	}
}

// sameMetrics compares face metrics up to float rounding.
func sameMetrics(a, b faceMetrics) bool {
	eq := func(x, y float64) bool { return math.Abs(x-y) < 1e-9 }
	return eq(a.width, b.width) && eq(a.ascent, b.ascent) && eq(a.descent, b.descent) &&
		eq(a.lineGap, b.lineGap) && eq(a.ulPos, b.ulPos) && eq(a.ulThick, b.ulThick)
}
