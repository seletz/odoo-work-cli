package main

import (
	"testing"

	"github.com/spf13/cobra"
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
