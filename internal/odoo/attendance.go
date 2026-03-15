package odoo

import (
	"errors"
	"fmt"
	"time"

	goOdoo "github.com/skilld-labs/go-odoo"
)

// odooDatetimeFormat is the datetime format used by Odoo XML-RPC.
const odooDatetimeFormat = "2006-01-02 15:04:05"

// findEmployeeID looks up the hr.employee ID for the authenticated user.
func (x *XMLRPCClient) findEmployeeID() (int64, error) {
	criteria := goOdoo.NewCriteria().Add("user_id.login", "=", x.login)
	opts := goOdoo.NewOptions().FetchFields("id")

	records, err := x.searchReadRaw("hr.employee", criteria, opts)
	if err != nil {
		return 0, fmt.Errorf("looking up employee: %w", err)
	}
	if len(records) == 0 {
		return 0, fmt.Errorf("no employee record found for user %q", x.login)
	}

	r := records[0]
	switch id := r["id"].(type) {
	case int64:
		return id, nil
	case float64:
		return int64(id), nil
	default:
		return 0, fmt.Errorf("unexpected employee id type %T", r["id"])
	}
}

// attendanceToggle calls the Odoo systray attendance toggle via JSON-RPC.
// This controller endpoint internally uses sudo() to bypass hr.attendance
// ACLs, so it works for non-admin users. Requires an authenticated web
// session (WebPassword), since Odoo rejects API keys for session auth.
func (x *XMLRPCClient) attendanceToggle() error {
	session, err := x.jsonSession()
	if err != nil {
		return err
	}
	_, err = session.call("/hr_attendance/systray_check_in_out", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("toggling attendance: %w", err)
	}
	return nil
}

// ClockIn toggles attendance state to clocked-in via the Odoo web controller.
// Returns the new attendance record ID. Returns an error if already clocked in.
func (x *XMLRPCClient) ClockIn() (int64, error) {
	status, err := x.AttendanceStatus()
	if err != nil {
		return 0, err
	}
	if status.ClockedIn {
		return 0, errors.New("already clocked in")
	}

	if err := x.attendanceToggle(); err != nil {
		return 0, err
	}

	// Re-read to get the new record ID.
	updated, err := x.AttendanceStatus()
	if err != nil {
		return 0, fmt.Errorf("reading status after clock in: %w", err)
	}
	return updated.CurrentID, nil
}

// ClockOut toggles attendance state to clocked-out via the Odoo web controller.
// Returns the completed record with worked_hours.
func (x *XMLRPCClient) ClockOut() (*AttendanceRecord, error) {
	status, err := x.AttendanceStatus()
	if err != nil {
		return nil, err
	}
	if !status.ClockedIn {
		return nil, errors.New("not clocked in")
	}

	recordID := status.CurrentID

	if err := x.attendanceToggle(); err != nil {
		return nil, err
	}

	// Re-read the closed record to get computed worked_hours.
	criteria := goOdoo.NewCriteria().Add("id", "=", recordID)
	opts := goOdoo.NewOptions().
		FetchFields("id", "employee_id", "check_in", "check_out", "worked_hours")
	records, err := x.searchReadRaw("hr.attendance", criteria, opts)
	if err != nil {
		return nil, fmt.Errorf("re-reading attendance record: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("attendance record %d not found after update", recordID)
	}

	return parseAttendanceRecord(records[0])
}

// attendanceSearchFunc abstracts over searchReadRaw for testing.
type attendanceSearchFunc func(model string, criteria *goOdoo.Criteria, opts *goOdoo.Options) ([]map[string]interface{}, error)

// AttendanceStatus returns the current clock state and today's attendance periods.
func (x *XMLRPCClient) AttendanceStatus() (*AttendanceStatus, error) {
	empID, err := x.findEmployeeID()
	if err != nil {
		return nil, err
	}
	return fetchAttendanceStatus(x.searchReadRaw, empID, time.Now())
}

// fetchAttendanceStatus queries attendance records and builds the status.
// It fetches both today's records (by check_in date) and any open records
// from previous days (check_out = false), so that overnight attendance
// spanning midnight is correctly detected.
func fetchAttendanceStatus(searchFn attendanceSearchFunc, empID int64, now time.Time) (*AttendanceStatus, error) {
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tomorrowStart := todayStart.AddDate(0, 0, 1)

	opts := goOdoo.NewOptions().
		FetchFields("id", "employee_id", "check_in", "check_out", "worked_hours")

	// Query 1: records checked in today.
	todayCriteria := goOdoo.NewCriteria().
		Add("employee_id", "=", empID).
		Add("check_in", ">=", todayStart.Format(odooDatetimeFormat)).
		Add("check_in", "<", tomorrowStart.Format(odooDatetimeFormat))

	records, err := searchFn("hr.attendance", todayCriteria, opts)
	if IsNotFound(err) {
		records = nil
	} else if err != nil {
		return nil, fmt.Errorf("fetching today's attendance: %w", err)
	}

	// Query 2: open records from previous days (check_in before today,
	// check_out = false). This catches overnight attendance that started
	// before midnight.
	openCriteria := goOdoo.NewCriteria().
		Add("employee_id", "=", empID).
		Add("check_in", "<", todayStart.Format(odooDatetimeFormat)).
		Add("check_out", "=", false)

	openRecords, err := searchFn("hr.attendance", openCriteria, opts)
	if IsNotFound(err) {
		openRecords = nil
	} else if err != nil {
		return nil, fmt.Errorf("fetching open attendance: %w", err)
	}

	// Merge: open records first, then today's records.
	all := append(openRecords, records...)

	return buildAttendanceStatus(all, now)
}

// buildAttendanceStatus processes raw Odoo attendance records into an
// AttendanceStatus summary. The now parameter is used to compute elapsed
// time for open (running) periods.
func buildAttendanceStatus(records []map[string]interface{}, now time.Time) (*AttendanceStatus, error) {
	status := &AttendanceStatus{}
	for _, r := range records {
		rec, err := parseAttendanceRecord(r)
		if err != nil {
			return nil, err
		}
		status.Periods = append(status.Periods, *rec)

		if rec.CheckOut == nil {
			status.ClockedIn = true
			status.CurrentID = rec.ID
			checkIn := rec.CheckIn
			status.CheckIn = &checkIn
			// For running period, compute elapsed time.
			elapsed := now.Sub(rec.CheckIn).Hours()
			status.TotalHours += elapsed
		} else {
			status.TotalHours += rec.WorkedHours
		}
	}

	return status, nil
}

// parseAttendanceRecord converts a raw Odoo record map to an AttendanceRecord.
func parseAttendanceRecord(r map[string]interface{}) (*AttendanceRecord, error) {
	rec := &AttendanceRecord{
		Employee: extractMany2OneName(r["employee_id"]),
	}

	switch id := r["id"].(type) {
	case int64:
		rec.ID = id
	case float64:
		rec.ID = int64(id)
	}

	rec.EmployeeID = extractMany2OneID(r["employee_id"])

	if checkIn, ok := r["check_in"].(string); ok {
		t, err := time.Parse(odooDatetimeFormat, checkIn)
		if err != nil {
			return nil, fmt.Errorf("parsing check_in %q: %w", checkIn, err)
		}
		rec.CheckIn = t
	}

	if co := r["check_out"]; co != nil && co != false {
		if checkOut, ok := co.(string); ok {
			t, err := time.Parse(odooDatetimeFormat, checkOut)
			if err != nil {
				return nil, fmt.Errorf("parsing check_out %q: %w", checkOut, err)
			}
			rec.CheckOut = &t
		}
	}

	if wh, ok := r["worked_hours"].(float64); ok {
		rec.WorkedHours = wh
	}

	return rec, nil
}
