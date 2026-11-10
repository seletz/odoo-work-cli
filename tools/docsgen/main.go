// Command docsgen writes the generated parts of the documentation site.
//
//	go run ./tools/docsgen screenshots   # TUI screenshots, goldens, CLI samples, demo state
//	go run ./tools/docsgen reference     # command reference from cobra
//	go run ./tools/docsgen mcp           # MCP tool reference
//
// The output directories are flags, so scripts/docs-check.sh can render
// into a temporary directory and compare with the committed files.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/seletz/odoo-work-cli/internal/cli"
	"github.com/seletz/odoo-work-cli/internal/docsgen"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "docsgen:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: docsgen screenshots|reference|mcp [flags]")
	}
	// Every rendering happens in the demo's time zone, whatever TZ says.
	time.Local = docsgen.Location
	switch args[0] {
	case "screenshots":
		return screenshots(args[1:])
	case "reference":
		return reference(args[1:])
	case "mcp":
		return mcpTools(args[1:])
	}
	return fmt.Errorf("unknown subcommand %q: use screenshots, reference or mcp", args[0])
}

func screenshots(args []string) error {
	fs := flag.NewFlagSet("screenshots", flag.ExitOnError)
	svgDir := fs.String("svg", "docs/assets/screenshots", "directory of the SVG screenshots")
	goldenDir := fs.String("golden", filepath.Join("internal/docsgen", docsgen.GoldenDir), "directory of the text goldens")
	samplesDir := fs.String("samples", "docs/assets/cli", "directory of the CLI output samples")
	state := fs.String("state", "docs/assets/demo-state.json", "demo state file of the fake backend")
	if err := fs.Parse(args); err != nil {
		return err
	}
	frames, err := docsgen.RenderAll()
	if err != nil {
		return err
	}
	if err := docsgen.WriteGoldens(*goldenDir, frames); err != nil {
		return err
	}
	if err := docsgen.WriteSVGs(*svgDir, frames); err != nil {
		return err
	}
	outs, err := docsgen.RunSamples()
	if err != nil {
		return err
	}
	if err := docsgen.WriteSamples(*samplesDir, outs); err != nil {
		return err
	}
	if err := docsgen.WriteDemoState(*state); err != nil {
		return err
	}
	fmt.Printf("wrote %d screenshots to %s, goldens to %s, %d samples to %s, demo state %s\n",
		len(frames), *svgDir, *goldenDir, len(outs), *samplesDir, *state)
	return nil
}

func reference(args []string) error {
	fs := flag.NewFlagSet("reference", flag.ExitOnError)
	out := fs.String("out", "docs/reference/commands", "directory of the command reference")
	if err := fs.Parse(args); err != nil {
		return err
	}
	files, err := docsgen.Reference(cli.NewRootCmd())
	if err != nil {
		return err
	}
	if err := docsgen.WriteReference(*out, files); err != nil {
		return err
	}
	fmt.Printf("wrote %d reference pages to %s\n", len(files), *out)
	return nil
}

// mcpTools writes the MCP tool reference page. Its stale check is
// TestMCPToolsReferenceUpToDate in internal/cli.
func mcpTools(args []string) error {
	fs := flag.NewFlagSet("mcp", flag.ExitOnError)
	out := fs.String("out", "docs/reference/60-mcp-tools.md", "file of the MCP tool reference")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := os.WriteFile(*out, []byte(cli.MCPToolsReference()), 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote the MCP tool reference to %s\n", *out)
	return nil
}
