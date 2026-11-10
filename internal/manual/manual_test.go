package manual

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"

	"github.com/seletz/odoo-work-cli/internal/apperr"
)

func TestList_SplitsAndTrims(t *testing.T) {
	cmd := &cobra.Command{Annotations: map[string]string{Errors: " not_found, conflict ,"}}
	got := List(cmd, Errors)
	if want := []string{"not_found", "conflict"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("List = %q, want %q", got, want)
	}
	if got := List(&cobra.Command{}, Errors); got != nil {
		t.Fatalf("List without annotation = %q, want nil", got)
	}
}

func TestIsOffline_InheritedFromParent(t *testing.T) {
	parent := &cobra.Command{Use: "config", Annotations: map[string]string{Offline: "true"}}
	child := &cobra.Command{Use: "install"}
	parent.AddCommand(child)
	if !IsOffline(child) {
		t.Fatal("a subcommand of an offline command must be offline")
	}
	if IsOffline(&cobra.Command{Use: "tasks"}) {
		t.Fatal("a command without the annotation is not offline")
	}
}

func TestGroupOf_InheritedFromTopLevel(t *testing.T) {
	root := &cobra.Command{Use: "odoo-work-cli"}
	clock := &cobra.Command{Use: "clock", Annotations: map[string]string{Group: GroupAttendance}}
	in := &cobra.Command{Use: "in"}
	root.AddCommand(clock)
	clock.AddCommand(in)
	if got := GroupOf(in); got != GroupAttendance {
		t.Fatalf("GroupOf(clock in) = %q, want %q", got, GroupAttendance)
	}
}

func TestFlagAnnotations(t *testing.T) {
	cmd := &cobra.Command{Use: "x"}
	cmd.Flags().String("date", "", "day")
	SetFlagDefault(cmd.Flags(), "date", "today")
	SetFlagEnv(cmd.Flags(), "date", "SOME_VAR")
	f := cmd.Flags().Lookup("date")
	if got := FlagDefault(f); got != "today" {
		t.Errorf("FlagDefault = %q", got)
	}
	if got := FlagEnv(f); got != "SOME_VAR" {
		t.Errorf("FlagEnv = %q", got)
	}
	if FlagRequired(f) {
		t.Error("date is not required")
	}
	SetFlagRequired(cmd.Flags(), "date")
	cmd.Flags().String("hours", "", "h")
	_ = cmd.MarkFlagRequired("hours")
	if !FlagRequired(f) || !FlagRequired(cmd.Flags().Lookup("hours")) {
		t.Error("SetFlagRequired and MarkFlagRequired both make a flag required")
	}
}

func TestEnvAndFiles_AreDescribed(t *testing.T) {
	for _, v := range append(OdooEnv(), "ODOO_WORK_CLI_BACKEND", "ODOO_WORK_CLI_FAKE_STATE", "ODOO_OP_TIMEOUT") {
		if EnvHelp[v] == "" {
			t.Errorf("environment variable %s has no description", v)
		}
	}
	for _, f := range []string{FileGlobalConfig, FileLocalConfig, FileFakeState} {
		if FileHelp[f] == "" {
			t.Errorf("file %s has no description", f)
		}
	}
}

func TestKindsOf_OnlineAddsConnectionKinds(t *testing.T) {
	root := &cobra.Command{Use: "odoo-work-cli"}
	del := &cobra.Command{Use: "delete", Annotations: map[string]string{Errors: "not_found"}}
	llm := &cobra.Command{Use: "llm", Annotations: map[string]string{Offline: "true"}}
	root.AddCommand(del, llm)

	got := KindsOf(del)
	want := []apperr.Kind{apperr.KindAuth, apperr.KindNotFound, apperr.KindValidation,
		apperr.KindNetwork, apperr.KindConfig, apperr.KindSecretsLocked, apperr.KindInternal}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("KindsOf(delete) = %v, want %v", got, want)
	}
	if got, want := KindsOf(llm), []apperr.Kind{apperr.KindValidation, apperr.KindInternal}; !reflect.DeepEqual(got, want) {
		t.Errorf("KindsOf(llm) = %v, want %v", got, want)
	}
}

func TestKindHelp_EveryKind(t *testing.T) {
	for _, k := range apperr.Kinds() {
		if KindHelp[k] == "" {
			t.Errorf("error kind %s has no description in KindHelp", k)
		}
	}
}

func TestSitePages_HaveTitles(t *testing.T) {
	for path, title := range SitePages {
		if title == "" {
			t.Errorf("site page %s has no title", path)
		}
	}
}
