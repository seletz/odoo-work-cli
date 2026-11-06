package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// walk calls fn for every command below c (c included), skipping hidden,
// help and completion commands.
func walk(c *cobra.Command, fn func(*cobra.Command)) {
	fn(c)
	for _, sub := range c.Commands() {
		switch {
		case sub.Hidden, sub.Name() == "help", sub.Name() == "completion":
			continue
		}
		walk(sub, fn)
	}
}

// TestLLMSkill_DocumentsEveryCommandAndFlag fails when a command or flag is
// added without documenting it in the guide.
func TestLLMSkill_DocumentsEveryCommandAndFlag(t *testing.T) {
	code, out, stderr := execute(t, failingSetup(errors.New("no setup")), "llm", "--skill")
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr)
	}
	guide := string(out)

	root, _ := newRootCmd(failingSetup(nil))
	walk(root, func(c *cobra.Command) {
		if c.Parent() != nil {
			path := strings.TrimPrefix(c.CommandPath(), root.Name()+" ")
			if !strings.Contains(guide, "odoo-work-cli "+path) {
				t.Errorf("command %q is not documented in the llm guide", path)
			}
		}
		check := func(f *pflag.Flag) {
			if f.Name == "help" {
				return
			}
			if !strings.Contains(guide, "--"+f.Name) {
				t.Errorf("flag --%s of %q is not documented in the llm guide", f.Name, c.CommandPath())
			}
		}
		c.NonInheritedFlags().VisitAll(check)
		c.InheritedFlags().VisitAll(check)
	})
}

func TestLLMSkill_NeedsNoConfigOrConnection(t *testing.T) {
	code, out, _ := execute(t, failingSetup(errors.New("must not be called")), "llm", "--skill")
	if code != 0 || len(out) == 0 {
		t.Fatalf("llm --skill must work without setup; exit %d", code)
	}
}

func TestLLMWithoutSkillFlag_IsValidationError(t *testing.T) {
	code, out, stderr := execute(t, failingSetup(errors.New("must not be called")), "llm")
	if code != 1 || len(out) != 0 || !strings.Contains(stderr, "--skill") {
		t.Fatalf("exit %d, stdout %s, stderr %s", code, out, stderr)
	}
}
