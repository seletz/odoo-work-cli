package cli

import (
	"strings"
	"testing"

	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/odoo/odootest"
)

// Issue #86: --search is handed to the client (Odoo does the matching) for
// every output format; no match prints [] / a clear message.
func TestSearchFlag(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		fake        *odootest.Fake
		wantProject string
		wantTask    string
		wantTaskPID int64
		wantOut     string // substring of stdout
		wantExact   string // exact stdout, when set
	}{
		{
			name: "projects json", args: []string{"projects", "--search", "ab-1", "--json"},
			fake: populated(), wantProject: "ab-1", wantOut: `"name"`,
		},
		{
			name: "projects jsonl", args: []string{"projects", "--search", "ab-1", "--format", "jsonl"},
			fake: populated(), wantProject: "ab-1", wantOut: `"name"`,
		},
		{
			name: "projects table", args: []string{"projects", "--search", "ab-1"},
			fake: populated(), wantProject: "ab-1", wantOut: "Internal",
		},
		{
			name: "projects json no match", args: []string{"projects", "--search", "zzz", "--json"},
			fake: &odootest.Fake{}, wantProject: "zzz", wantExact: "[]\n",
		},
		{
			name: "projects jsonl no match", args: []string{"projects", "--search", "zzz", "--format", "jsonl"},
			fake: &odootest.Fake{}, wantProject: "zzz", wantExact: "[]\n",
		},
		{
			name: "projects table no match", args: []string{"projects", "--search", "zzz"},
			fake: &odootest.Fake{}, wantProject: "zzz", wantOut: `No projects match "zzz"`,
		},
		{
			name: "tasks search only", args: []string{"tasks", "--search", "rev", "--json"},
			fake: populated(), wantTask: "rev", wantOut: `"name"`,
		},
		{
			name: "tasks search with project", args: []string{"tasks", "3", "--search", "rev", "--json"},
			fake: populated(), wantTask: "rev", wantTaskPID: 3, wantOut: `"name"`,
		},
		{
			name: "tasks json no match", args: []string{"tasks", "--search", "zzz", "--json"},
			fake: &odootest.Fake{}, wantTask: "zzz", wantExact: "[]\n",
		},
		{
			name: "tasks table no match", args: []string{"tasks", "--search", "zzz"},
			fake: &odootest.Fake{}, wantTask: "zzz", wantOut: `No tasks match "zzz"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := execute(t, fakeSetup(tt.fake), tt.args...)
			if code != 0 {
				t.Fatalf("exit %d, stderr: %s, stdout: %s", code, stderr, stdout)
			}
			if tt.fake.ProjectSearch != tt.wantProject {
				t.Errorf("project search = %q, want %q", tt.fake.ProjectSearch, tt.wantProject)
			}
			if tt.fake.TaskSearch != tt.wantTask || tt.fake.TaskSearchProject != tt.wantTaskPID {
				t.Errorf("task search = (%d, %q), want (%d, %q)",
					tt.fake.TaskSearchProject, tt.fake.TaskSearch, tt.wantTaskPID, tt.wantTask)
			}
			if tt.wantExact != "" && string(stdout) != tt.wantExact {
				t.Errorf("stdout = %q, want %q", stdout, tt.wantExact)
			}
			if tt.wantOut != "" && !strings.Contains(string(stdout), tt.wantOut) {
				t.Errorf("stdout %q lacks %q", stdout, tt.wantOut)
			}
		})
	}
}

// Without --search the plain list calls are used, not the search ones.
func TestNoSearchFlagUsesPlainList(t *testing.T) {
	fake := &odootest.Fake{Projects: []odoo.ProjectInfo{{ID: 1, Name: "X"}}}
	if code, _, stderr := execute(t, fakeSetup(fake), "projects", "--json"); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if fake.ProjectSearch != "" {
		t.Errorf("ProjectSearch = %q, want empty", fake.ProjectSearch)
	}
}
