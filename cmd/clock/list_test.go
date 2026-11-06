package clock

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/seletz/odoo-work-cli/internal/odoo"
)

func TestResolveListRange(t *testing.T) {
	now := time.Date(2026, 7, 16, 14, 0, 0, 0, time.Local) // Thursday, W29

	tests := []struct {
		name     string
		date     string
		week     string
		wantFrom time.Time
		wantTo   time.Time
		wantErr  string
	}{
		{
			name:     "default is today",
			wantFrom: time.Date(2026, 7, 16, 0, 0, 0, 0, time.Local),
			wantTo:   time.Date(2026, 7, 17, 0, 0, 0, 0, time.Local),
		},
		{
			name:     "explicit date",
			date:     "2026-07-14",
			wantFrom: time.Date(2026, 7, 14, 0, 0, 0, 0, time.Local),
			wantTo:   time.Date(2026, 7, 15, 0, 0, 0, 0, time.Local),
		},
		{
			name:     "week starts at local Monday midnight",
			week:     "2026-W29",
			wantFrom: time.Date(2026, 7, 13, 0, 0, 0, 0, time.Local),
			wantTo:   time.Date(2026, 7, 20, 0, 0, 0, 0, time.Local),
		},
		{
			name:     "week across DST change keeps local midnights",
			week:     "2026-W44",
			wantFrom: time.Date(2026, 10, 26, 0, 0, 0, 0, time.Local),
			wantTo:   time.Date(2026, 11, 2, 0, 0, 0, 0, time.Local),
		},
		{name: "date and week are exclusive", date: "2026-07-14", week: "2026-W29", wantErr: "mutually exclusive"},
		{name: "bad date", date: "14.07.2026", wantErr: "invalid date"},
		{name: "bad week", week: "29", wantErr: "invalid week"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to, err := resolveListRange(tt.date, tt.week, now)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !from.Equal(tt.wantFrom) || !to.Equal(tt.wantTo) {
				t.Errorf("range = [%v, %v), want [%v, %v)", from, to, tt.wantFrom, tt.wantTo)
			}
		})
	}
}

func TestWriteListTable(t *testing.T) {
	now := time.Date(2026, 7, 16, 14, 0, 0, 0, time.Local)
	at := func(d, h, m int) time.Time { return time.Date(2026, 7, d, h, m, 0, 0, time.Local) }
	out := func(d, h, m int) *time.Time { v := at(d, h, m); return &v }

	tests := []struct {
		name    string
		records []odoo.AttendanceRecord
		want    []string
		notWant []string
	}{
		{
			name:    "empty",
			records: nil,
			want:    []string{"No attendance records."},
		},
		{
			name: "same-day record shows id, times and duration",
			records: []odoo.AttendanceRecord{
				{ID: 4667, CheckIn: at(14, 8, 0), CheckOut: out(14, 12, 30), WorkedHours: 4.5},
			},
			want:    []string{"ID", "4667", "2026-07-14 08:00", "12:30", "4:30"},
			notWant: []string{"2026-07-14 12:30"},
		},
		{
			name: "overnight record shows the check-out date",
			records: []odoo.AttendanceRecord{
				{ID: 4680, CheckIn: at(14, 22, 0), CheckOut: out(15, 1, 30), WorkedHours: 3.5},
			},
			want: []string{"4680", "2026-07-14 22:00", "2026-07-15 01:30", "3:30"},
		},
		{
			name: "running record shows elapsed time",
			records: []odoo.AttendanceRecord{
				{ID: 4690, CheckIn: at(16, 12, 0)},
			},
			want: []string{"4690", "2026-07-16 12:00", "--:--", "2:00", "running"},
		},
		{
			name: "total sums worked hours and running time",
			records: []odoo.AttendanceRecord{
				{ID: 1, CheckIn: at(16, 8, 0), CheckOut: out(16, 12, 0), WorkedHours: 4},
				{ID: 2, CheckIn: at(16, 12, 0)},
			},
			want: []string{"Total: 6:00"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := writeListTable(&buf, tt.records, now); err != nil {
				t.Fatal(err)
			}
			got := buf.String()
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("output missing %q:\n%s", w, got)
				}
			}
			for _, w := range tt.notWant {
				if strings.Contains(got, w) {
					t.Errorf("output unexpectedly contains %q:\n%s", w, got)
				}
			}
		})
	}
}

func TestDescribeChange(t *testing.T) {
	at := func(h, m int) time.Time { return time.Date(2026, 7, 14, h, m, 0, 0, time.Local) }
	out := func(h, m int) *time.Time { v := at(h, m); return &v }

	before := odoo.AttendanceRecord{ID: 4667, CheckIn: at(8, 0), CheckOut: out(12, 0), WorkedHours: 4}
	after := odoo.AttendanceRecord{ID: 4667, CheckIn: at(8, 15), CheckOut: out(12, 0), WorkedHours: 3.75}

	got := describeChange(&before, &after)
	for _, w := range []string{"#4667", "was", "2026-07-14 08:00", "now", "2026-07-14 08:15", "3:45"} {
		if !strings.Contains(got, w) {
			t.Errorf("describeChange missing %q:\n%s", w, got)
		}
	}
}

func TestSortByCheckIn(t *testing.T) {
	at := func(d, h int) time.Time { return time.Date(2026, 7, d, h, 0, 0, 0, time.Local) }
	records := []odoo.AttendanceRecord{
		{ID: 3, CheckIn: at(15, 8)},
		{ID: 1, CheckIn: at(13, 22)},
		{ID: 2, CheckIn: at(13, 8)},
	}
	sortByCheckIn(records)
	for i, want := range []int64{2, 1, 3} {
		if records[i].ID != want {
			t.Fatalf("order = %v, want IDs 2,1,3", records)
		}
	}
}
