package docsgen

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"
)

// ComparePNGDirs compares the .png files of two directories by their
// decoded pixels and dimensions, not by their bytes: the zlib stream of a
// PNG depends on the Go version's compress/flate, the image does not.
//
// :param want: the directory of the committed images
// :param got: the directory of the fresh render
// :returns: one line per difference (missing, extra, other size, other
//
//	pixels); empty when both hold the same images
func ComparePNGDirs(want, got string) ([]string, error) {
	wantFiles, err := pngNames(want)
	if err != nil {
		return nil, err
	}
	gotFiles, err := pngNames(got)
	if err != nil {
		return nil, err
	}
	var diffs []string
	for _, name := range wantFiles {
		if !contains(gotFiles, name) {
			diffs = append(diffs, fmt.Sprintf("%s: not in the fresh render", name))
		}
	}
	for _, name := range gotFiles {
		if !contains(wantFiles, name) {
			diffs = append(diffs, fmt.Sprintf("%s: new in the fresh render", name))
			continue
		}
		d, err := comparePNG(filepath.Join(want, name), filepath.Join(got, name))
		if err != nil {
			return nil, err
		}
		if d != "" {
			diffs = append(diffs, name+": "+d)
		}
	}
	return diffs, nil
}

// comparePNG returns how two images differ, or "" when they are equal.
func comparePNG(want, got string) (string, error) {
	a, err := decodePNG(want)
	if err != nil {
		return "", err
	}
	b, err := decodePNG(got)
	if err != nil {
		return "", err
	}
	ab, bb := a.Bounds(), b.Bounds()
	if ab.Size() != bb.Size() {
		return fmt.Sprintf("size %dx%d, fresh render %dx%d", ab.Dx(), ab.Dy(), bb.Dx(), bb.Dy()), nil
	}
	for y := range ab.Dy() {
		for x := range ab.Dx() {
			ca, cb := a.At(ab.Min.X+x, ab.Min.Y+y), b.At(bb.Min.X+x, bb.Min.Y+y)
			r1, g1, b1, a1 := ca.RGBA()
			r2, g2, b2, a2 := cb.RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				return fmt.Sprintf("pixel (%d,%d) is %v, fresh render %v", x, y, ca, cb), nil
			}
		}
	}
	return "", nil
}

func decodePNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return img, nil
}

// pngNames lists the .png files of dir, sorted.
func pngNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".png" {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

func contains(list []string, s string) bool {
	i := sort.SearchStrings(list, s)
	return i < len(list) && list[i] == s
}
