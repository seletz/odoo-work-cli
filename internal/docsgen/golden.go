package docsgen

import (
	"fmt"
	"os"
	"path/filepath"
)

// GoldenDir is where the text frames of the shots are checked in, relative
// to this package: one <shot>.txt per shot, written by tools/docsgen
// together with the images and compared by TestFramesGolden.
const GoldenDir = "testdata/frames"

// GoldenPath returns the golden file of a shot below dir.
//
// :param dir: the golden directory
// :param name: the shot name
// :returns: the file path
func GoldenPath(dir, name string) string { return filepath.Join(dir, name+".txt") }

// WriteGoldens writes the plain text frames below dir, one file per shot,
// and removes the files of shots that no longer exist.
//
// :param dir: the golden directory
// :param frames: the rendered frames
// :returns: an error if writing fails
func WriteGoldens(dir string, frames []Frame) error {
	files := map[string][]byte{}
	for _, f := range frames {
		files[f.Shot.Name+".txt"] = []byte(f.Plain() + "\n")
	}
	return writeDir(dir, ".txt", files)
}

// writeDir writes files (name → content) into dir and removes every other
// file with extension ext there, so renamed or dropped outputs disappear.
func writeDir(dir, ext string, files map[string][]byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("%s: %w", dir, err)
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return fmt.Errorf("%s: %w", dir, err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("%s: %w", dir, err)
	}
	for _, e := range entries {
		if _, keep := files[e.Name()]; !e.IsDir() && filepath.Ext(e.Name()) == ext && !keep {
			if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
				return fmt.Errorf("%s: %w", dir, err)
			}
		}
	}
	return nil
}
