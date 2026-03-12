package filter

import (
	"testing"

	"github.com/seletz/odoo-work-cli/internal/odoo"
)

func TestEntries(t *testing.T) {
	entries := []odoo.TimesheetEntry{
		{ID: 1, Date: "2026-03-02", Project: "Acme Corp", Task: "Backend Dev", Name: "Auth endpoint", Hours: 2.0, ValidatedStatus: "draft"},
		{ID: 2, Date: "2026-03-02", Project: "Acme Corp", Task: "QA Testing", Name: "Review PR", Hours: 1.5, ValidatedStatus: "validated"},
		{ID: 3, Date: "2026-03-03", Project: "Beta Project", Task: "Frontend Dev", Name: "Dashboard", Hours: 4.0, ValidatedStatus: "draft"},
	}

	tests := []struct {
		name    string
		project string
		task    string
		status  string
		wantIDs []int64
	}{
		{
			name:    "no filter",
			wantIDs: []int64{1, 2, 3},
		},
		{
			name:    "filter by project",
			project: "acme",
			wantIDs: []int64{1, 2},
		},
		{
			name:    "filter by task",
			task:    "dev",
			wantIDs: []int64{1, 3},
		},
		{
			name:    "filter by both",
			project: "acme",
			task:    "qa",
			wantIDs: []int64{2},
		},
		{
			name:    "no match",
			project: "nonexistent",
			wantIDs: nil,
		},
		{
			name:    "filter by status draft",
			status:  "draft",
			wantIDs: []int64{1, 3},
		},
		{
			name:    "filter by status validated",
			status:  "validated",
			wantIDs: []int64{2},
		},
		{
			name:    "filter by status and project",
			project: "acme",
			status:  "draft",
			wantIDs: []int64{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Entries(entries, tt.project, tt.task, tt.status)
			if len(got) != len(tt.wantIDs) {
				t.Fatalf("got %d entries, want %d", len(got), len(tt.wantIDs))
			}
			for i, entry := range got {
				if entry.ID != tt.wantIDs[i] {
					t.Errorf("entry[%d].ID = %d, want %d", i, entry.ID, tt.wantIDs[i])
				}
			}
		})
	}
}
