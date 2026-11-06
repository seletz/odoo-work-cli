package llm

import (
	"regexp"
	"strings"
	"testing"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/output"
)

func TestGuide_ListsEveryErrorKind(t *testing.T) {
	g, err := Guide()
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range apperr.Kinds() {
		if !strings.Contains(g, "`"+string(k)+"`") {
			t.Errorf("guide does not mention error kind %q", k)
		}
	}
}

func TestKindDescriptions_CoverEveryKind(t *testing.T) {
	for _, k := range apperr.Kinds() {
		if kindHelp[k] == "" {
			t.Errorf("no agent-oriented description for error kind %q", k)
		}
	}
}

func TestGuide_ListsFieldsOfEverySchema(t *testing.T) {
	g, err := Guide()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		output.SchemaProject, output.SchemaTask, output.SchemaEntry, output.SchemaUser,
		output.SchemaAttendance, output.SchemaClockStatus, output.SchemaAttendanceEdit,
	} {
		if !strings.Contains(g, "`"+name+"`") {
			t.Errorf("guide does not describe schema %q", name)
		}
	}
	if !strings.Contains(g, "before") {
		t.Error("guide must mention the `before` field of clock edit")
	}
}

func TestGuide_NoUnresolvedPlaceholders(t *testing.T) {
	g, err := Guide()
	if err != nil {
		t.Fatal(err)
	}
	if m := regexp.MustCompile(`\{\{[^}]*\}\}`).FindString(g); m != "" {
		t.Errorf("unresolved placeholder %s", m)
	}
}

func TestGuide_CoversPitfalls(t *testing.T) {
	g, err := Guide()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"secrets_locked", "ODOO_OP_TIMEOUT", "--create-only", "YYYY-MM-DD HH:MM",
		"attendance_manager_id", "Attendance Officer",
	} {
		if !strings.Contains(g, want) {
			t.Errorf("guide does not mention %q", want)
		}
	}
}
