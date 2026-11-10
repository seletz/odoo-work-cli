// Command changelog is the release tooling for CHANGELOG.md.
//
//	go run ./tools/changelog check                   # fail when Unreleased has no entries
//	go run ./tools/changelog section vX.Y.Z          # print a release's notes
//	go run ./tools/changelog release vX.Y.Z DATE     # turn Unreleased into a dated section
//	git diff --name-only BASE...HEAD | go run ./tools/changelog needed  # fail when a code change has no entry
//
// The file defaults to CHANGELOG.md; -file selects another one. scripts/release.sh
// and the release workflow call it.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/seletz/odoo-work-cli/internal/changelog"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "changelog:", err)
		os.Exit(1)
	}
}

func run(args []string, out *os.File) error {
	fs := flag.NewFlagSet("changelog", flag.ContinueOnError)
	file := fs.String("file", "CHANGELOG.md", "changelog file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return fmt.Errorf("usage: changelog [-file F] check | section VERSION | release VERSION DATE")
	}
	if rest[0] == "needed" {
		return needed(os.Stdin)
	}
	data, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	doc := string(data)
	switch rest[0] {
	case "check":
		if !changelog.UnreleasedHasEntries(doc) {
			return fmt.Errorf("the [Unreleased] section of %s is empty: describe the changes of this release first", *file)
		}
		return nil
	case "section":
		if len(rest) != 2 {
			return fmt.Errorf("usage: changelog section VERSION")
		}
		body, err := changelog.Section(doc, rest[1])
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, body)
		return err
	case "release":
		if len(rest) != 3 {
			return fmt.Errorf("usage: changelog release VERSION DATE")
		}
		next, err := changelog.Release(doc, rest[1], rest[2])
		if err != nil {
			return err
		}
		return os.WriteFile(*file, []byte(next), 0o644)
	default:
		return fmt.Errorf("unknown command %q", rest[0])
	}
}

// needed reads changed file names (one per line) and fails when they change
// code without CHANGELOG.md.
func needed(r io.Reader) error {
	var files []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		if f := sc.Text(); f != "" {
			files = append(files, f)
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if changelog.MissingEntry(files) {
		return fmt.Errorf("this change touches cmd/ or internal/ but not CHANGELOG.md: add an entry under [Unreleased], or label the pull request no-changelog")
	}
	return nil
}
