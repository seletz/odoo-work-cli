package odoo

import (
	"errors"
	"testing"
	"time"
)

func TestClockIn_Success(t *testing.T) {
	client := &mockClient{
		clockInID: 99,
	}

	id, err := client.ClockIn()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 99 {
		t.Errorf("ID = %d, want 99", id)
	}
}

func TestClockIn_AlreadyClockedIn(t *testing.T) {
	client := &mockClient{
		clockInErr: errors.New("already clocked in"),
	}

	_, err := client.ClockIn()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "already clocked in" {
		t.Errorf("error = %q, want %q", err.Error(), "already clocked in")
	}
}

func TestClockOut_Success(t *testing.T) {
	checkIn := time.Date(2026, 3, 9, 8, 30, 0, 0, time.UTC)
	checkOut := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	client := &mockClient{
		clockOutRecord: &AttendanceRecord{
			ID:          42,
			EmployeeID:  7,
			Employee:    "Test User",
			CheckIn:     checkIn,
			CheckOut:    &checkOut,
			WorkedHours: 3.5,
		},
	}

	rec, err := client.ClockOut()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.ID != 42 {
		t.Errorf("ID = %d, want 42", rec.ID)
	}
	if rec.WorkedHours != 3.5 {
		t.Errorf("WorkedHours = %f, want 3.5", rec.WorkedHours)
	}
	if rec.CheckOut == nil {
		t.Fatal("expected CheckOut to be set")
	}
}

func TestClockOut_NotClockedIn(t *testing.T) {
	client := &mockClient{
		clockOutErr: errors.New("not clocked in"),
	}

	_, err := client.ClockOut()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "not clocked in" {
		t.Errorf("error = %q, want %q", err.Error(), "not clocked in")
	}
}

func TestAttendanceStatus_ClockedIn(t *testing.T) {
	checkIn := time.Date(2026, 3, 9, 13, 0, 0, 0, time.UTC)
	client := &mockClient{
		attendanceStatus: &AttendanceStatus{
			ClockedIn: true,
			CurrentID: 43,
			CheckIn:   &checkIn,
			Periods: []AttendanceRecord{
				{
					ID:          42,
					CheckIn:     time.Date(2026, 3, 9, 8, 30, 0, 0, time.UTC),
					CheckOut:    ptrTime(time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)),
					WorkedHours: 3.5,
				},
				{
					ID:      43,
					CheckIn: checkIn,
				},
			},
			TotalHours: 5.25,
		},
	}

	status, err := client.AttendanceStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.ClockedIn {
		t.Error("expected ClockedIn = true")
	}
	if status.CurrentID != 43 {
		t.Errorf("CurrentID = %d, want 43", status.CurrentID)
	}
	if len(status.Periods) != 2 {
		t.Fatalf("Periods len = %d, want 2", len(status.Periods))
	}
	if status.TotalHours != 5.25 {
		t.Errorf("TotalHours = %f, want 5.25", status.TotalHours)
	}
}

func TestAttendanceStatus_NotClockedIn(t *testing.T) {
	client := &mockClient{
		attendanceStatus: &AttendanceStatus{
			ClockedIn:  false,
			TotalHours: 8.0,
			Periods: []AttendanceRecord{
				{
					ID:          42,
					CheckIn:     time.Date(2026, 3, 9, 8, 0, 0, 0, time.UTC),
					CheckOut:    ptrTime(time.Date(2026, 3, 9, 16, 0, 0, 0, time.UTC)),
					WorkedHours: 8.0,
				},
			},
		},
	}

	status, err := client.AttendanceStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.ClockedIn {
		t.Error("expected ClockedIn = false")
	}
	if status.CurrentID != 0 {
		t.Errorf("CurrentID = %d, want 0", status.CurrentID)
	}
	if status.TotalHours != 8.0 {
		t.Errorf("TotalHours = %f, want 8.0", status.TotalHours)
	}
}

func TestAttendanceStatus_Error(t *testing.T) {
	client := &mockClient{
		attendanceErr: errors.New("connection failed"),
	}

	_, err := client.AttendanceStatus()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseAttendanceRecord(t *testing.T) {
	tests := []struct {
		name    string
		record  map[string]interface{}
		wantID  int64
		wantOut bool
	}{
		{
			name: "open record",
			record: map[string]interface{}{
				"id":          int64(10),
				"employee_id": []interface{}{int64(7), "Test User"},
				"check_in":    "2026-03-09 08:30:00",
				"check_out":   false,
				"worked_hours": float64(0),
			},
			wantID:  10,
			wantOut: false,
		},
		{
			name: "closed record",
			record: map[string]interface{}{
				"id":           float64(11),
				"employee_id":  []interface{}{float64(7), "Test User"},
				"check_in":     "2026-03-09 08:30:00",
				"check_out":    "2026-03-09 12:00:00",
				"worked_hours": float64(3.5),
			},
			wantID:  11,
			wantOut: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, err := parseAttendanceRecord(tt.record)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rec.ID != tt.wantID {
				t.Errorf("ID = %d, want %d", rec.ID, tt.wantID)
			}
			if tt.wantOut && rec.CheckOut == nil {
				t.Error("expected CheckOut to be set")
			}
			if !tt.wantOut && rec.CheckOut != nil {
				t.Error("expected CheckOut to be nil")
			}
		})
	}
}

// ptrTime returns a pointer to the given time.Time.
func ptrTime(t time.Time) *time.Time {
	return &t
}
