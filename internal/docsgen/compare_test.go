package docsgen

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePNG(t *testing.T, dir, name string, img image.Image, level png.CompressionLevel) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := (&png.Encoder{CompressionLevel: level}).Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func testImage(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// TestComparePNGDirs: same pixels compare equal whatever the compression;
// a changed pixel, another size, a missing or an extra file are reported.
func TestComparePNGDirs(t *testing.T) {
	red := color.RGBA{200, 0, 0, 255}
	tests := []struct {
		name  string
		setup func(want, got string)
		diff  []string // substrings of the report; none means equal
	}{
		{"same pixels, other compression", func(want, got string) {
			writePNG(t, want, "a.png", testImage(4, 3, red), png.BestCompression)
			writePNG(t, got, "a.png", testImage(4, 3, red), png.NoCompression)
		}, nil},
		{"one pixel differs", func(want, got string) {
			writePNG(t, want, "a.png", testImage(4, 3, red), png.DefaultCompression)
			img := testImage(4, 3, red)
			img.SetRGBA(2, 1, color.RGBA{201, 0, 0, 255})
			writePNG(t, got, "a.png", img, png.DefaultCompression)
		}, []string{"a.png", "pixel (2,1)"}},
		{"other size", func(want, got string) {
			writePNG(t, want, "a.png", testImage(4, 3, red), png.DefaultCompression)
			writePNG(t, got, "a.png", testImage(5, 3, red), png.DefaultCompression)
		}, []string{"a.png", "4x3", "5x3"}},
		{"missing and extra", func(want, got string) {
			writePNG(t, want, "old.png", testImage(1, 1, red), png.DefaultCompression)
			writePNG(t, got, "new.png", testImage(1, 1, red), png.DefaultCompression)
		}, []string{"old.png", "new.png"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, got := t.TempDir(), t.TempDir()
			tt.setup(want, got)
			diffs, err := ComparePNGDirs(want, got)
			if err != nil {
				t.Fatal(err)
			}
			report := strings.Join(diffs, "\n")
			if tt.diff == nil && len(diffs) != 0 {
				t.Fatalf("want equal, got:\n%s", report)
			}
			if tt.diff != nil && len(diffs) == 0 {
				t.Fatal("want a difference, got none")
			}
			for _, s := range tt.diff {
				if !strings.Contains(report, s) {
					t.Errorf("report lacks %q:\n%s", s, report)
				}
			}
		})
	}
}
