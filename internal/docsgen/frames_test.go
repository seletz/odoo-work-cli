package docsgen

import (
	"flag"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "rewrite the goldens in testdata/frames")

func TestMain(m *testing.M) {
	// Renders format times in time.Local; pin it like tools/docsgen does.
	time.Local = Location
	os.Exit(m.Run())
}

// allFrames renders every shot once per test binary.
var allFrames = sync.OnceValues(RenderAll)

func frames(t *testing.T) []Frame {
	t.Helper()
	fs, err := allFrames()
	if err != nil {
		t.Fatal(err)
	}
	return fs
}

func renderShot(t *testing.T, name string) Frame {
	t.Helper()
	for _, f := range frames(t) {
		if f.Shot.Name == name {
			return f
		}
	}
	t.Fatalf("no shot %q", name)
	return Frame{}
}

func TestRender_GridShowsDemoWeek(t *testing.T) {
	plain := renderShot(t, "grid").Plain()
	for _, want := range []string{"W15", "Lighthouse Portal", "● In 08:41 (6:31)", "Check-in"} {
		if !strings.Contains(plain, want) {
			t.Errorf("grid frame lacks %q:\n%s", want, plain)
		}
	}
	if strings.Contains(plain, "Loading") {
		t.Errorf("grid frame still loading:\n%s", plain)
	}
}

func TestRender_FrameFitsTheTerminal(t *testing.T) {
	for _, f := range frames(t) {
		s := f.Shot
		lines := strings.Split(f.Plain(), "\n")
		if len(lines) > s.Height {
			t.Errorf("%s: %d lines, terminal has %d", s.Name, len(lines), s.Height)
		}
	}
}

func TestRender_IsDeterministic(t *testing.T) {
	for _, a := range frames(t) {
		s := a.Shot
		b, err := Render(s)
		if err != nil {
			t.Fatal(err)
		}
		if a.Content != b.Content {
			t.Errorf("%s renders differently on a second run", s.Name)
		}
	}
}

func TestRender_ShotsDiffer(t *testing.T) {
	seen := map[string]string{}
	for _, f := range frames(t) {
		s := f.Shot
		if other, ok := seen[f.Plain()]; ok {
			t.Errorf("shots %s and %s render the same frame; do the keys work?", other, s.Name)
		}
		seen[f.Plain()] = s.Name
	}
}

// TestFramesGolden compares every shot with its committed golden. A TUI
// change that alters a frame fails here until `mise run docs:screenshots`
// (or go test -update) rewrote the goldens and images.
func TestFramesGolden(t *testing.T) {
	fs := frames(t)
	if *update {
		if err := WriteGoldens(GoldenDir, fs); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range fs {
		want, err := os.ReadFile(GoldenPath(GoldenDir, f.Shot.Name))
		if err != nil {
			t.Fatalf("%v (run `mise run docs:screenshots`)", err)
		}
		if got := f.Plain() + "\n"; got != string(want) {
			t.Errorf("%s differs from its golden; run `mise run docs:screenshots`:\n%s", f.Shot.Name, got)
		}
	}
}
