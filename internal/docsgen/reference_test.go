package docsgen

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/seletz/odoo-work-cli/internal/cli"
)

func TestReference_OnePagePerCommand(t *testing.T) {
	files, err := Reference(cli.NewRootCmd())
	if err != nil {
		t.Fatal(err)
	}
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if !c.IsAvailableCommand() && c.HasParent() {
			return
		}
		name := strings.ReplaceAll(c.CommandPath(), " ", "_") + ".md"
		if _, ok := files[name]; !ok {
			t.Errorf("no reference page %s for %q", name, c.CommandPath())
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(cli.NewRootCmd())
	if _, ok := files["odoo-work-cli_clock_in.md"]; !ok {
		t.Fatal("clock in missing")
	}
	if !strings.Contains(string(files["odoo-work-cli_clock_in.md"]), "--at") {
		t.Error("clock in page lacks its --at flag")
	}
}

func TestReference_NoDateStamp(t *testing.T) {
	files, err := Reference(cli.NewRootCmd())
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range files {
		if strings.Contains(string(data), "Auto generated") {
			t.Errorf("%s carries cobra's dated auto-gen tag; the check would fail every day", name)
		}
	}
}

func TestReference_IndexListsCommands(t *testing.T) {
	files, err := Reference(cli.NewRootCmd())
	if err != nil {
		t.Fatal(err)
	}
	index := string(files["index.md"])
	for _, want := range []string{"(odoo-work-cli_entries_add.md)", "(odoo-work-cli_clock_edit.md)", "(odoo-work-cli_llm.md)"} {
		if !strings.Contains(index, want) {
			t.Errorf("index lacks link %s:\n%s", want, index)
		}
	}
}
