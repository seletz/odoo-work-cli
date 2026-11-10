package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/seletz/odoo-work-cli/internal/manual"
)

func TestIsCompletionCommand(t *testing.T) {
	root := &cobra.Command{Use: "odoo-work-cli"}
	completion := &cobra.Command{Use: "completion"}
	zsh := &cobra.Command{Use: "zsh"}
	tasks := &cobra.Command{Use: "tasks"}
	internalComplete := &cobra.Command{Use: "__complete"}

	root.AddCommand(completion, tasks, internalComplete)
	completion.AddCommand(zsh)

	if !isNoneSetupCommand(completion) {
		t.Fatal("expected completion command to be detected")
	}

	if !isNoneSetupCommand(zsh) {
		t.Fatal("expected completion subcommand to be detected")
	}

	if !isNoneSetupCommand(internalComplete) {
		t.Fatal("expected internal completion command to be detected")
	}

	if isNoneSetupCommand(tasks) {
		t.Fatal("did not expect regular command to be detected as completion")
	}
}

func TestIsNoneSetupCommand_OfflineAnnotation(t *testing.T) {
	root := NewRootCmd()
	for _, path := range [][]string{{"config"}, {"config", "install"}, {"llm"}, {"completion", "zsh"}} {
		c, _, err := root.Find(path)
		if err != nil {
			t.Fatal(err)
		}
		if !isNoneSetupCommand(c) {
			t.Errorf("%v must not load config or connect", path)
		}
		if !manual.IsOffline(c) {
			t.Errorf("%v must carry the manual.Offline annotation", path)
		}
	}
	tasks, _, _ := root.Find([]string{"tasks"})
	if isNoneSetupCommand(tasks) || manual.IsOffline(tasks) {
		t.Error("tasks connects to Odoo")
	}
}

func TestCompletionCommand_IsDocumented(t *testing.T) {
	root := NewRootCmd()
	c, _, err := root.Find([]string{"completion"})
	if err != nil || c.Name() != "completion" {
		t.Fatalf("completion command missing: %v", err)
	}
	if c.Annotations[manual.Group] != manual.GroupCompletion || c.Example == "" {
		t.Errorf("completion lacks its group or examples: %v", c.Annotations)
	}
}

func TestHelp_ShowsManualDetailsAndReference(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"entries", "add", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	help := stdout.String()
	for _, want := range []string{
		"TEXT describing the work (required)",
		"day of the entry, YYYY-MM-DD (default today)",
		"man odoo-work-cli-entries-add",
		"https://seletz.github.io/odoo-work-cli/reference/commands/odoo-work-cli-entries-add/",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("help lacks %q:\n%s", want, help)
		}
	}
}
