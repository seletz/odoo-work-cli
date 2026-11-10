package cli

import (
	"os"
	"strings"
	"testing"
)

// mcpToolsPage is the generated reference page of the MCP tools.
const mcpToolsPage = "../../docs/reference/60-mcp-tools.md"

// TestMCPToolsReferenceUpToDate is the stale check of the generated MCP
// tool reference, like docs:check for the command reference.
func TestMCPToolsReferenceUpToDate(t *testing.T) {
	committed, err := os.ReadFile(mcpToolsPage)
	if err != nil {
		t.Fatalf("%v: run `mise run docs:mcp`", err)
	}
	if string(committed) != MCPToolsReference() {
		t.Fatalf("%s is stale: run `mise run docs:mcp` and commit the result", mcpToolsPage)
	}
}

// TestMCPToolsReferenceListsEveryTool: every tool and every argument is on
// the page, with its annotations.
func TestMCPToolsReferenceListsEveryTool(t *testing.T) {
	page := MCPToolsReference()
	for _, tool := range mcpTools() {
		if !strings.Contains(page, "## `"+tool.Name+"`") {
			t.Errorf("no section for %s", tool.Name)
		}
		for _, p := range tool.Params {
			if !strings.Contains(page, "| `"+p.Name+"` |") {
				t.Errorf("%s: argument %s missing", tool.Name, p.Name)
			}
		}
	}
	for _, want := range []string{"readOnlyHint", "destructiveHint", "idempotentHint", "--read-only"} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %s", want)
		}
	}
}
