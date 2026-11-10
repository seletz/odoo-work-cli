// Package changelog reads and edits a Keep a Changelog file (CHANGELOG.md).
//
// A section starts with a level-2 heading "## [Unreleased]" or
// "## [X.Y.Z] - DATE" and runs to the next level-2 heading. The release
// tooling (tools/changelog, scripts/release.sh, the release workflow) uses it
// to refuse an empty Unreleased section, to cut a release into a dated
// section and to extract a release's notes for GoReleaser.
package changelog

import (
	"fmt"
	"regexp"
	"strings"
)

// Unreleased is the title of the section that collects the changes of the
// next release.
const Unreleased = "Unreleased"

var headingRe = regexp.MustCompile(`^## \[([^\]]+)\]`)

// span locates a section: the heading line and the half-open range of body
// lines after it.
type span struct{ heading, start, end int }

func find(lines []string, title string) (span, bool) {
	for i, l := range lines {
		m := headingRe.FindStringSubmatch(l)
		if m == nil || m[1] != title {
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if headingRe.MatchString(lines[j]) || strings.HasPrefix(lines[j], "## ") {
				end = j
				break
			}
		}
		return span{i, i + 1, end}, true
	}
	return span{}, false
}

// hasEntries reports whether lines hold anything but blank lines and
// level-3 headings.
func hasEntries(lines []string) bool {
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t != "" && !strings.HasPrefix(t, "### ") {
			return true
		}
	}
	return false
}

// Section returns the body of the section for version (a leading "v" is
// ignored; "Unreleased" selects the unreleased section), trimmed of
// surrounding blank lines. It fails when the section is missing or has no
// entries.
//
// :param doc: the content of CHANGELOG.md
// :param version: "X.Y.Z", "vX.Y.Z" or "Unreleased"
func Section(doc, version string) (string, error) {
	version = strings.TrimPrefix(version, "v")
	lines := strings.Split(doc, "\n")
	s, ok := find(lines, version)
	if !ok {
		return "", fmt.Errorf("no section for %s in the changelog", version)
	}
	body := lines[s.start:s.end]
	if !hasEntries(body) {
		return "", fmt.Errorf("the section for %s in the changelog is empty", version)
	}
	return strings.Trim(strings.Join(body, "\n"), "\n"), nil
}

// UnreleasedHasEntries reports whether the Unreleased section exists and
// lists at least one change.
func UnreleasedHasEntries(doc string) bool {
	lines := strings.Split(doc, "\n")
	s, ok := find(lines, Unreleased)
	return ok && hasEntries(lines[s.start:s.end])
}

// Release turns the Unreleased section into the dated section of version
// and leaves a new empty Unreleased section above it. It refuses an empty
// or missing Unreleased section and a version that already has a section.
//
// :param doc: the content of CHANGELOG.md
// :param version: "X.Y.Z" or "vX.Y.Z"
// :param date: release date, YYYY-MM-DD
func Release(doc, version, date string) (string, error) {
	version = strings.TrimPrefix(version, "v")
	lines := strings.Split(doc, "\n")
	s, ok := find(lines, Unreleased)
	if !ok {
		return "", fmt.Errorf("the changelog has no [%s] section", Unreleased)
	}
	if !hasEntries(lines[s.start:s.end]) {
		return "", fmt.Errorf("the [%s] section of the changelog is empty", Unreleased)
	}
	if _, exists := find(lines, version); exists {
		return "", fmt.Errorf("the changelog already has a section for %s", version)
	}
	out := make([]string, 0, len(lines)+2)
	out = append(out, lines[:s.heading+1]...)
	out = append(out, "", fmt.Sprintf("## [%s] - %s", version, date))
	out = append(out, lines[s.heading+1:]...)
	return strings.Join(out, "\n"), nil
}

// MissingEntry reports whether a change set needs a changelog entry it does
// not have: it touches code under cmd/ or internal/ (not tests, not test
// data) but not CHANGELOG.md. CI uses it for pull requests; the label
// no-changelog is the opt-out.
//
// :param files: repository-relative paths changed by the pull request
func MissingEntry(files []string) bool {
	code := false
	for _, f := range files {
		if f == "CHANGELOG.md" {
			return false
		}
		if (strings.HasPrefix(f, "cmd/") || strings.HasPrefix(f, "internal/")) &&
			!strings.HasSuffix(f, "_test.go") && !strings.Contains(f, "/testdata/") {
			code = true
		}
	}
	return code
}
