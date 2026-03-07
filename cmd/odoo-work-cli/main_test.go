package main

import (
	"testing"
)

func TestWeekDateRange(t *testing.T) {
	tests := []struct {
		name     string
		week     string
		wantFrom string
		wantTo   string
		wantErr  bool
	}{
		{
			name:     "2026-W10",
			week:     "2026-W10",
			wantFrom: "2026-03-02",
			wantTo:   "2026-03-08",
		},
		{
			name:     "2026-W01",
			week:     "2026-W01",
			wantFrom: "2025-12-29",
			wantTo:   "2026-01-04",
		},
		{
			name:     "2025-W52",
			week:     "2025-W52",
			wantFrom: "2025-12-22",
			wantTo:   "2025-12-28",
		},
		{
			name:    "invalid format",
			week:    "not-a-week",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to, err := weekDateRange(tt.week)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if from != tt.wantFrom {
				t.Errorf("from = %q, want %q", from, tt.wantFrom)
			}
			if to != tt.wantTo {
				t.Errorf("to = %q, want %q", to, tt.wantTo)
			}
		})
	}
}
