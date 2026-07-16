package parsing

import (
	"testing"
	"time"
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
			from, to, err := WeekDateRange(tt.week)

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

func TestParseDateRange(t *testing.T) {
	tests := []struct {
		name     string
		date     string
		wantFrom string
		wantTo   string
		wantErr  bool
	}{
		{
			name:     "valid date",
			date:     "2026-03-05",
			wantFrom: "2026-03-05",
			wantTo:   "2026-03-05",
		},
		{
			name:    "invalid date",
			date:    "not-a-date",
			wantErr: true,
		},
		{
			name:    "wrong format",
			date:    "05/03/2026",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to, err := ParseDateRange(tt.date)
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

func TestParseClockTime(t *testing.T) {
	zone := time.FixedZone("CEST", 2*3600)
	day := time.Date(2026, 7, 16, 0, 0, 0, 0, zone)

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "bare time HH:MM on the given day",
			input: "08:30",
			want:  time.Date(2026, 7, 16, 8, 30, 0, 0, zone),
		},
		{
			name:  "bare time single-digit hour",
			input: "7:05",
			want:  time.Date(2026, 7, 16, 7, 5, 0, 0, zone),
		},
		{
			name:  "full datetime overrides the day",
			input: "2026-07-14 17:15",
			want:  time.Date(2026, 7, 14, 17, 15, 0, 0, zone),
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "garbage",
			input:   "later",
			wantErr: true,
		},
		{
			name:    "out of range time",
			input:   "25:99",
			wantErr: true,
		},
		{
			name:    "date without time",
			input:   "2026-07-14",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseClockTime(tt.input, day)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if got.Location() != zone {
				t.Errorf("location = %v, want %v", got.Location(), zone)
			}
		})
	}
}

func TestParseDay(t *testing.T) {
	zone := time.FixedZone("CEST", 2*3600)
	now := time.Date(2026, 7, 16, 14, 45, 12, 0, zone)

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "empty input means today at midnight",
			input: "",
			want:  time.Date(2026, 7, 16, 0, 0, 0, 0, zone),
		},
		{
			name:  "explicit date",
			input: "2026-07-14",
			want:  time.Date(2026, 7, 14, 0, 0, 0, 0, zone),
		},
		{
			name:    "invalid date",
			input:   "14.07.2026",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDay(tt.input, now)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
