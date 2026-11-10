package docsgen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// Reference renders the command reference of root with cobra/doc: one
// Markdown page per command (odoo-work-cli_clock_in.md, …) plus index.md
// with a table of all commands. Cobra's dated auto-gen footer is turned
// off, so the pages only change when the commands do.
//
// :param root: the root command (cli.NewRootCmd())
// :returns: file name → content
func Reference(root *cobra.Command) (map[string][]byte, error) {
	root.InitDefaultCompletionCmd()
	disableAutoGenTag(root)

	dir, err := os.MkdirTemp("", "odoo-work-cli-reference-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	if err := doc.GenMarkdownTree(root, dir); err != nil {
		return nil, fmt.Errorf("reference: %w", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		// cobra starts each page with "## <command>"; make it the page title.
		if bytes.HasPrefix(data, []byte("## ")) {
			data = data[1:]
		}
		files[e.Name()] = data
	}
	files["index.md"] = referenceIndex(root)
	return files, nil
}

func disableAutoGenTag(c *cobra.Command) {
	c.DisableAutoGenTag = true
	for _, sub := range c.Commands() {
		disableAutoGenTag(sub)
	}
}

// referenceIndex lists every available command with its short help.
func referenceIndex(root *cobra.Command) []byte {
	var b bytes.Buffer
	b.WriteString("---\nweight: 10\n---\n\n# Commands\n\n")
	b.WriteString("Generated from the CLI's own help text by `mise run docs:reference`.\n")
	b.WriteString("Do not edit these pages by hand; `mise run docs:reference:check` fails when\nthey differ from the code.\n\n")
	b.WriteString("Every command also accepts the global flags `--config` and `--backend`.\n\n")
	b.WriteString("| Command | Description |\n| --- | --- |\n")
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if c.HasParent() && (!c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand()) {
			return
		}
		name := strings.ReplaceAll(c.CommandPath(), " ", "_") + ".md"
		fmt.Fprintf(&b, "| [`%s`](%s) | %s |\n", c.CommandPath(), name, c.Short)
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
	return b.Bytes()
}

// WriteReference writes the reference pages into dir (docs/reference/
// commands) and removes pages of commands that no longer exist.
//
// :param dir: the output directory
// :param files: the pages from Reference
// :returns: an error if writing fails
func WriteReference(dir string, files map[string][]byte) error {
	return writeDir(dir, ".md", files)
}
