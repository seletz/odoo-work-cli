package odoo

import (
	"errors"
	"fmt"
	"strings"
	"time"

	goOdoo "github.com/skilld-labs/go-odoo"
)

// odooAccessErrorPrefix identifies Odoo AccessError faults (ACL / record-rule
// violations, fault code 4) in error messages. The XML-RPC transport under
// go-odoo flattens faults into net/rpc string errors of the form
// "Fault(<code>): <message>", so the code is only available textually. The
// message text is locale-dependent, the code is not.
const odooAccessErrorPrefix = "Fault(4):"

// attendanceAccessHint explains the ACL constraint on hr.attendance writes
// discovered in #47: regular employees may only read their own records;
// create/write/unlink require the officer group, and the officer record rule
// additionally requires being the employee's attendance manager.
const attendanceAccessHint = "editing attendance over the API requires the " +
	"'Attendance Officer' access right with you set as your own attendance " +
	"manager; ask your administrator, or use plain 'clock in'/'clock out'"

// wrapAttendanceAccessErr wraps an hr.attendance write error, appending a
// hint about the required access rights when Odoo raised an AccessError.
func wrapAttendanceAccessErr(err error, action string) error {
	if strings.HasPrefix(err.Error(), odooAccessErrorPrefix) {
		return fmt.Errorf("%s: %w\nhint: %s", action, err, attendanceAccessHint)
	}
	return fmt.Errorf("%s: %w", action, err)
}

// createAttendance creates an hr.attendance record via XML-RPC.
// checkOut may be nil for an open (still clocked in) record.
func (x *XMLRPCClient) createAttendance(checkIn time.Time, checkOut *time.Time) (int64, error) {
	empID, err := x.findEmployeeID()
	if err != nil {
		return 0, err
	}

	vals := map[string]interface{}{
		"employee_id": empID,
		"check_in":    checkIn.UTC().Format(odooDatetimeFormat),
	}
	if checkOut != nil {
		vals["check_out"] = checkOut.UTC().Format(odooDatetimeFormat)
	}

	resp, err := x.client.ExecuteKw("create", "hr.attendance",
		[]interface{}{vals}, goOdoo.NewOptions())
	if err != nil {
		return 0, wrapAttendanceAccessErr(err, "creating attendance record")
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

// readAttendance fetches a single attendance record by ID.
func (x *XMLRPCClient) readAttendance(id int64) (*AttendanceRecord, error) {
	criteria := goOdoo.NewCriteria().Add("id", "=", id)
	opts := goOdoo.NewOptions().
		FetchFields("id", "employee_id", "check_in", "check_out", "worked_hours")
	records, err := x.searchReadRaw("hr.attendance", criteria, opts)
	if err != nil {
		return nil, fmt.Errorf("reading attendance record: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("attendance record %d not found", id)
	}
	return parseAttendanceRecord(records[0])
}

// ClockInAt creates an attendance record with the given check-in time via
// XML-RPC. Unlike ClockIn (which goes through the sudo'd systray controller
// and always uses the server's now()), this requires attendance officer
// rights. Returns an error if already clocked in.
func (x *XMLRPCClient) ClockInAt(checkIn time.Time) (int64, error) {
	status, err := x.AttendanceStatus()
	if err != nil {
		return 0, err
	}
	if status.ClockedIn {
		return 0, errors.New("already clocked in")
	}

	return x.createAttendance(checkIn, nil)
}

// ClockOutAt writes the given check-out time on the open attendance record
// via XML-RPC. Requires attendance officer rights. Returns the completed
// record with computed worked_hours.
func (x *XMLRPCClient) ClockOutAt(checkOut time.Time) (*AttendanceRecord, error) {
	status, err := x.AttendanceStatus()
	if err != nil {
		return nil, err
	}
	if !status.ClockedIn {
		return nil, errors.New("not clocked in")
	}
	if status.CheckIn != nil && !checkOut.After(*status.CheckIn) {
		return nil, fmt.Errorf("check-out %s is not after check-in %s",
			checkOut.Format("15:04"),
			status.CheckIn.In(checkOut.Location()).Format("15:04"))
	}

	fields := map[string]interface{}{
		"check_out": checkOut.UTC().Format(odooDatetimeFormat),
	}
	_, err = x.client.ExecuteKw("write", "hr.attendance",
		[]interface{}{[]int64{status.CurrentID}, fields}, goOdoo.NewOptions())
	if err != nil {
		return nil, wrapAttendanceAccessErr(err, fmt.Sprintf("closing attendance record %d", status.CurrentID))
	}

	return x.readAttendance(status.CurrentID)
}

// CreateAttendance creates a closed attendance record with explicit check-in
// and check-out times via XML-RPC. Requires attendance officer rights.
func (x *XMLRPCClient) CreateAttendance(checkIn, checkOut time.Time) (int64, error) {
	if !checkOut.After(checkIn) {
		return 0, fmt.Errorf("check-out %s is not after check-in %s",
			checkOut.Format("15:04"),
			checkIn.Format("15:04"))
	}
	return x.createAttendance(checkIn, &checkOut)
}

// EditAttendance updates check_in and/or check_out on an attendance record
// via XML-RPC; nil times are left unchanged. Requires attendance officer
// rights. Returns the updated record.
func (x *XMLRPCClient) EditAttendance(id int64, checkIn, checkOut *time.Time) (*AttendanceRecord, error) {
	if id <= 0 {
		return nil, errors.New("attendance ID is required")
	}
	if checkIn == nil && checkOut == nil {
		return nil, errors.New("nothing to update: provide a new check-in and/or check-out time")
	}
	if checkIn != nil && checkOut != nil && !checkOut.After(*checkIn) {
		return nil, fmt.Errorf("check-out %s is not after check-in %s",
			checkOut.Format("15:04"),
			checkIn.Format("15:04"))
	}

	fields := map[string]interface{}{}
	if checkIn != nil {
		fields["check_in"] = checkIn.UTC().Format(odooDatetimeFormat)
	}
	if checkOut != nil {
		fields["check_out"] = checkOut.UTC().Format(odooDatetimeFormat)
	}

	_, err := x.client.ExecuteKw("write", "hr.attendance",
		[]interface{}{[]int64{id}, fields}, goOdoo.NewOptions())
	if err != nil {
		return nil, wrapAttendanceAccessErr(err, fmt.Sprintf("updating attendance record %d", id))
	}

	return x.readAttendance(id)
}
