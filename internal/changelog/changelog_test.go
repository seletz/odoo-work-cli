package changelog

import (
	"strings"
	"testing"
)

const sample = `# Release notes

Intro text.

## [Unreleased]

### Added

- New thing (#1).

## [0.2.0] - 2026-01-02

### Fixed

- A fix (#2).

## [0.1.0] - 2026-01-01

- First.
`

func TestSection(t *testing.T) {
	tests := []struct {
		name    string
		doc     string
		version string
		want    string
		wantErr string
	}{
		{"released with v prefix", sample, "v0.2.0", "### Fixed\n\n- A fix (#2).", ""},
		{"released without prefix", sample, "0.1.0", "- First.", ""},
		{"unreleased", sample, "Unreleased", "### Added\n\n- New thing (#1).", ""},
		{"missing", sample, "0.3.0", "", "no section"},
		{"empty section", "## [0.3.0] - 2026-01-03\n\n## [0.2.0] - x\n- a\n", "0.3.0", "", "empty"},
		{"version is not a prefix match", sample, "0.2", "", "no section"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Section(tt.doc, tt.version)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnreleasedHasEntries(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		want bool
	}{
		{"with entry", sample, true},
		{"empty", "## [Unreleased]\n\n## [0.1.0] - x\n- a\n", false},
		{"only sub-headings", "## [Unreleased]\n\n### Added\n\n### Fixed\n\n## [0.1.0] - x\n", false},
		{"missing section", "## [0.1.0] - x\n- a\n", false},
		{"last section", "## [Unreleased]\n\n- x\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UnreleasedHasEntries(tt.doc); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRelease(t *testing.T) {
	got, err := Release(sample, "v0.3.0", "2026-02-03")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(sample, "## [Unreleased]\n",
		"## [Unreleased]\n\n## [0.3.0] - 2026-02-03\n", 1)
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if _, err := Section(got, "0.3.0"); err != nil {
		t.Errorf("new section not found: %v", err)
	}
	if UnreleasedHasEntries(got) {
		t.Error("Unreleased must be empty after a release")
	}
}

func TestReleaseErrors(t *testing.T) {
	if _, err := Release("## [Unreleased]\n\n## [0.1.0] - x\n- a\n", "0.2.0", "2026-01-01"); err == nil {
		t.Error("empty Unreleased must be refused")
	}
	if _, err := Release("# x\n", "0.2.0", "2026-01-01"); err == nil {
		t.Error("missing Unreleased must be refused")
	}
	if _, err := Release(sample, "0.2.0", "2026-01-01"); err == nil {
		t.Error("an existing version must be refused")
	}
}
