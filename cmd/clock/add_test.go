package clock

import (
	"strings"
	"testing"
	"time"

	"github.com/seletz/odoo-work-cli/internal/odoo"
)

func TestResolveAddTimes(t *testing.T) {
	now := time.Date(2026, 7, 16, 14, 0, 0, 0, testZone)

	tests := []struct {
		name    string
		date    string
		in      string
		out     string
		wantErr string
	}{
		{name: "both times", date: "2026-07-14", in: "08:00", out: "12:00"},
		{name: "missing --out", date: "2026-07-14", in: "08:00", wantErr: "needs both --in and --out"},
		{name: "missing --in", date: "2026-07-14", out: "12:00", wantErr: "needs both --in and --out"},
		{name: "neither", wantErr: "needs both --in and --out"},
		{name: "future rejected", in: "13:00", out: "18:00", wantErr: "in the future"},
		{name: "out not after in", date: "2026-07-14", in: "12:00", out: "12:00", wantErr: "not after check-in"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveAddTimes(tt.date, tt.in, tt.out, now)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.checkIn == nil || got.checkOut == nil {
				t.Fatalf("times not set: %+v", got)
			}
		})
	}
}

func TestFindOverlaps(t *testing.T) {
	day := time.Date(2026, 7, 14, 0, 0, 0, 0, testZone)
	at := func(h, m int) time.Time { return day.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute) }
	closed := func(id int64, in, out time.Time) odoo.AttendanceRecord {
		return odoo.AttendanceRecord{ID: id, CheckIn: in, CheckOut: &out}
	}
	morning := closed(1, at(8, 0), at(12, 0))
	overnight := closed(2, at(-2, 0), at(1, 30)) // previous day 22:00 → 01:30
	running := odoo.AttendanceRecord{ID: 3, CheckIn: at(13, 0)}

	tests := []struct {
		name    string
		records []odoo.AttendanceRecord
		in, out time.Time
		wantIDs []int64
	}{
		{name: "free day", in: at(8, 0), out: at(12, 0)},
		{name: "second record after the first", records: []odoo.AttendanceRecord{morning}, in: at(13, 0), out: at(17, 30)},
		{name: "overlap at the start", records: []odoo.AttendanceRecord{morning}, in: at(11, 0), out: at(14, 0), wantIDs: []int64{1}},
		{name: "overlap at the end", records: []odoo.AttendanceRecord{morning}, in: at(7, 0), out: at(9, 0), wantIDs: []int64{1}},
		{name: "fully inside", records: []odoo.AttendanceRecord{morning}, in: at(9, 0), out: at(10, 0), wantIDs: []int64{1}},
		{name: "fully covering", records: []odoo.AttendanceRecord{morning}, in: at(7, 0), out: at(13, 0), wantIDs: []int64{1}},
		{name: "adjacent after", records: []odoo.AttendanceRecord{morning}, in: at(12, 0), out: at(13, 0)},
		{name: "adjacent before", records: []odoo.AttendanceRecord{morning}, in: at(6, 0), out: at(8, 0)},
		{name: "previous-day record past midnight", records: []odoo.AttendanceRecord{overnight}, in: at(1, 0), out: at(4, 0), wantIDs: []int64{2}},
		{name: "previous-day record ends before", records: []odoo.AttendanceRecord{overnight}, in: at(1, 30), out: at(4, 0)},
		{name: "running record started before", records: []odoo.AttendanceRecord{running}, in: at(14, 0), out: at(15, 0), wantIDs: []int64{3}},
		{name: "running record started after", records: []odoo.AttendanceRecord{running}, in: at(8, 0), out: at(12, 0)},
		{name: "running record started at out", records: []odoo.AttendanceRecord{running}, in: at(12, 0), out: at(13, 0)},
		{name: "several overlaps", records: []odoo.AttendanceRecord{morning, running}, in: at(11, 0), out: at(14, 0), wantIDs: []int64{1, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findOverlaps(tt.records, tt.in, tt.out)
			var ids []int64
			for _, r := range got {
				ids = append(ids, r.ID)
			}
			if len(ids) != len(tt.wantIDs) {
				t.Fatalf("overlaps = %v, want %v", ids, tt.wantIDs)
			}
			for i := range ids {
				if ids[i] != tt.wantIDs[i] {
					t.Fatalf("overlaps = %v, want %v", ids, tt.wantIDs)
				}
			}
		})
	}
}

func TestFormatRecord_OvernightCarriesDate(t *testing.T) {
	in := time.Date(2026, 7, 14, 22, 0, 0, 0, time.Local)
	out := in.Add(3*time.Hour + 30*time.Minute)
	got := formatRecord(&odoo.AttendanceRecord{ID: 7, CheckIn: in, CheckOut: &out, WorkedHours: 3.5})
	if want := "#7 2026-07-14 22:00 - 2026-07-15 01:30 (3:30)"; got != want {
		t.Errorf("formatRecord = %q, want %q", got, want)
	}
}
