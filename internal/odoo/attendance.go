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

// findOpenAttendance searches for an open attendance record (check_out = False)
// for the given employee. Returns nil if none found.
func (x *XMLRPCClient) findOpenAttendance(employeeID int64) (map[string]interface{}, error) {
	criteria := goOdoo.NewCriteria().
		Add("employee_id", "=", employeeID).
		Add("check_out", "=", false)
	opts := goOdoo.NewOptions().
		FetchFields("id", "employee_id", "check_in", "check_out", "worked_hours").
		Limit(1)

	records, err := x.searchReadRaw("hr.attendance", criteria, opts)
	if IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("searching open attendance: %w", err)
	}
	if len(records) == 0 {
		return nil, nil
	}
	return records[0], nil
}

// ClockIn creates an attendance record with check_in = now.
// Returns an error if already clocked in.
func (x *XMLRPCClient) ClockIn() (int64, error) {
	empID, err := x.findEmployeeID()
	if err != nil {
		return 0, err
	}

	open, err := x.findOpenAttendance(empID)
	if err != nil {
		return 0, err
	}
	if open != nil {
		return 0, errors.New("already clocked in")
	}

	now := time.Now().UTC().Format(odooDatetimeFormat)
	vals := map[string]interface{}{
		"employee_id": empID,
		"check_in":    now,
	}

	resp, err := x.client.ExecuteKw("create", "hr.attendance",
		[]interface{}{vals}, goOdoo.NewOptions())
	if err != nil {
		return 0, fmt.Errorf("creating attendance record: %w", err)
	}

	switch id := resp.(type) {
	case int64:
		return id, nil
	case float64:
		return int64(id), nil
	default:
		return 0, fmt.Errorf("unexpected create response type %T", resp)
	}
}

// ClockOut writes check_out = now on the open attendance record.
// Returns the completed record with worked_hours.
func (x *XMLRPCClient) ClockOut() (*AttendanceRecord, error) {
	empID, err := x.findEmployeeID()
	if err != nil {
		return nil, err
	}

	open, err := x.findOpenAttendance(empID)
	if err != nil {
		return nil, err
	}
	if open == nil {
		return nil, errors.New("not clocked in")
	}

	var recordID int64
	switch id := open["id"].(type) {
	case int64:
		recordID = id
	case float64:
		recordID = int64(id)
	}

	now := time.Now().UTC().Format(odooDatetimeFormat)
	_, err = x.client.ExecuteKw("write", "hr.attendance",
		[]interface{}{[]int64{recordID}, map[string]interface{}{
			"check_out": now,
		}}, goOdoo.NewOptions())
	if err != nil {
		return nil, fmt.Errorf("writing check_out: %w", err)
	}

	// Re-read the record to get computed worked_hours.
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

// AttendanceStatus returns the current clock state and today's attendance periods.
func (x *XMLRPCClient) AttendanceStatus() (*AttendanceStatus, error) {
	empID, err := x.findEmployeeID()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tomorrowStart := todayStart.AddDate(0, 0, 1)

	criteria := goOdoo.NewCriteria().
		Add("employee_id", "=", empID).
		Add("check_in", ">=", todayStart.Format(odooDatetimeFormat)).
		Add("check_in", "<", tomorrowStart.Format(odooDatetimeFormat))
	opts := goOdoo.NewOptions().
		FetchFields("id", "employee_id", "check_in", "check_out", "worked_hours")

	records, err := x.searchReadRaw("hr.attendance", criteria, opts)
	if IsNotFound(err) {
		records = nil
	} else if err != nil {
		return nil, fmt.Errorf("fetching today's attendance: %w", err)
	}

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
			elapsed := time.Since(rec.CheckIn).Hours()
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
