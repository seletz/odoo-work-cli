package clock

import (
	"strings"
	"testing"
	"time"

	"github.com/seletz/odoo-work-cli/internal/odoo"
)

var testZone = time.FixedZone("CEST", 2*3600)

func TestParseAtFlag(t *testing.T) {
	now := time.Date(2026, 7, 16, 14, 0, 0, 0, testZone)

	tests := []struct {
		name    string
		at      string
		want    time.Time
		wantErr string
	}{
		{
			name: "time today",
			at:   "08:30",
			want: time.Date(2026, 7, 16, 8, 30, 0, 0, testZone),
		},
		{
			name: "explicit past date",
			at:   "2026-07-15 17:15",
			want: time.Date(2026, 7, 15, 17, 15, 0, 0, testZone),
		},
		{
			name:    "future time rejected",
			at:      "18:00",
			wantErr: "in the future",
		},
		{
			name:    "invalid format",
			at:      "later",
			wantErr: "invalid time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAtFlag(tt.at, now)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
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

func TestResolveEditTimes(t *testing.T) {
	now := time.Date(2026, 7, 16, 14, 0, 0, 0, testZone)

	tests := []struct {
		name    string
		date    string
		in      string
		out     string
		wantIn  *time.Time
		wantOut *time.Time
		wantDay time.Time
		wantErr string
	}{
		{
			name:    "check-in only, today",
			in:      "08:30",
			wantIn:  timePtr(time.Date(2026, 7, 16, 8, 30, 0, 0, testZone)),
			wantDay: time.Date(2026, 7, 16, 0, 0, 0, 0, testZone),
		},
		{
			name:    "both times on explicit date",
			date:    "2026-07-14",
			in:      "08:00",
			out:     "16:30",
			wantIn:  timePtr(time.Date(2026, 7, 14, 8, 0, 0, 0, testZone)),
			wantOut: timePtr(time.Date(2026, 7, 14, 16, 30, 0, 0, testZone)),
			wantDay: time.Date(2026, 7, 14, 0, 0, 0, 0, testZone),
		},
		{
			name:    "check-out only",
			out:     "12:15",
			wantOut: timePtr(time.Date(2026, 7, 16, 12, 15, 0, 0, testZone)),
			wantDay: time.Date(2026, 7, 16, 0, 0, 0, 0, testZone),
		},
		{
			name:    "neither time given",
			wantErr: "provide --in and/or --out",
		},
		{
			name:    "future check-in rejected",
			in:      "18:00",
			wantErr: "in the future",
		},
		{
			name:    "check-out before check-in",
			in:      "12:00",
			out:     "08:00",
			wantErr: "not after check-in",
		},
		{
			name:    "invalid date",
			date:    "14.07.2026",
			in:      "08:00",
			wantErr: "invalid date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveEditTimes(tt.date, tt.in, tt.out, now)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.day.Equal(tt.wantDay) {
				t.Errorf("day = %v, want %v", got.day, tt.wantDay)
			}
			if !equalTimePtr(got.checkIn, tt.wantIn) {
				t.Errorf("checkIn = %v, want %v", got.checkIn, tt.wantIn)
			}
			if !equalTimePtr(got.checkOut, tt.wantOut) {
				t.Errorf("checkOut = %v, want %v", got.checkOut, tt.wantOut)
			}
		})
	}
}

func TestPickEditTarget(t *testing.T) {
	rec1 := odoo.AttendanceRecord{ID: 10, CheckIn: time.Date(2026, 7, 16, 6, 0, 0, 0, time.UTC)}
	rec2 := odoo.AttendanceRecord{ID: 11, CheckIn: time.Date(2026, 7, 16, 11, 0, 0, 0, time.UTC)}

	tests := []struct {
		name    string
		records []odoo.AttendanceRecord
		id      int64
		wantID  int64
		wantNil bool
		wantErr string
	}{
		{
			name:    "no records yields nil without error",
			records: nil,
			wantNil: true,
		},
		{
			name:    "single record selected",
			records: []odoo.AttendanceRecord{rec1},
			wantID:  10,
		},
		{
			name:    "multiple records require --id",
			records: []odoo.AttendanceRecord{rec1, rec2},
			wantErr: "--id",
		},
		{
			name:    "explicit id selects among multiple",
			records: []odoo.AttendanceRecord{rec1, rec2},
			id:      11,
			wantID:  11,
		},
		{
			name:    "explicit id not found",
			records: []odoo.AttendanceRecord{rec1},
			id:      99,
			wantErr: "no attendance record with ID 99",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pickEditTarget(tt.records, tt.id)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil target, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected a target, got nil")
			}
			if got.ID != tt.wantID {
				t.Errorf("ID = %d, want %d", got.ID, tt.wantID)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time { return &t }

func equalTimePtr(a, b *time.Time) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	return a == nil || a.Equal(*b)
}
