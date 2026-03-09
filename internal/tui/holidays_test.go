package tui

import (
	"testing"
	"time"
)

func TestBuildHolidayMap_Bayern(t *testing.T) {
	m := BuildHolidayMap(2026, "Bayern")

	tests := []struct {
		date string
		name string
	}{
		{"2026-01-01", "Neujahr"},
		{"2026-01-06", "Epiphanias"}, // Bayern-specific
		{"2026-05-01", "Tag der Arbeit"},
		{"2026-10-03", "Tag der deutschen Einheit"},
		{"2026-12-25", "Weihnachten"},
		{"2026-12-26", "Zweiter Weihnachtsfeiertag"},
	}
	for _, tt := range tests {
		got, ok := m[tt.date]
		if !ok {
			t.Errorf("missing holiday %s (%s)", tt.date, tt.name)
			continue
		}
		if got != tt.name {
			t.Errorf("holiday %s = %q, want %q", tt.date, got, tt.name)
		}
	}
}

func TestBuildHolidayMap_EmptyBundesland(t *testing.T) {
	m := BuildHolidayMap(2026, "")
	// Should fall back to Deutschland and still have Neujahr.
	if _, ok := m["2026-01-01"]; !ok {
		t.Error("empty bundesland should fall back to Deutschland, missing Neujahr")
	}
}

func TestBuildHolidayMap_UnknownBundesland(t *testing.T) {
	m := BuildHolidayMap(2026, "Narnia")
	if _, ok := m["2026-01-01"]; !ok {
		t.Error("unknown bundesland should fall back to Deutschland, missing Neujahr")
	}
}

func TestBuildHolidayMap_HeiligeDreiKoenige_NotInBerlin(t *testing.T) {
	m := BuildHolidayMap(2026, "Berlin")
	if _, ok := m["2026-01-06"]; ok {
		t.Error("Heilige Drei Könige should not be in Berlin holidays")
	}
}

func TestWeekHolidays(t *testing.T) {
	holidays := BuildHolidayMap(2026, "Bayern")
	// Week of Dec 21 contains Christmas.
	mon := time.Date(2026, 12, 21, 0, 0, 0, 0, time.UTC)
	wh := WeekHolidays(mon, holidays)

	// Thu Dec 25 = index 3 (but 25 is Fri, let me recalc: Mon=21, Tue=22, Wed=23, Thu=24, Fri=25)
	if wh[4] != "Weihnachten" {
		t.Errorf("Fri Dec 25 = %q, want 'Weihnachten'", wh[4])
	}
	if wh[5] != "Zweiter Weihnachtsfeiertag" {
		t.Errorf("Sat Dec 26 = %q, want 'Zweiter Weihnachtsfeiertag'", wh[5])
	}
	// Monday Dec 21 is not a holiday.
	if wh[0] != "" {
		t.Errorf("Mon Dec 21 = %q, want empty", wh[0])
	}
}

func TestWeekHolidays_NoHolidays(t *testing.T) {
	holidays := BuildHolidayMap(2026, "Bayern")
	// A random week with no holidays.
	mon := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	wh := WeekHolidays(mon, holidays)
	for d, name := range wh {
		if name != "" {
			t.Errorf("day %d = %q, want empty", d, name)
		}
	}
}
