package fake

import (
	"sort"
	"time"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

// odooTime truncates to whole seconds, as Odoo stores datetimes.
func odooTime(t time.Time) time.Time { return t.Truncate(time.Second).Local() }

func (c *Client) attendanceIndex(id int64) int {
	for i := range c.st.Attendances {
		if c.st.Attendances[i].ID == id {
			return i
		}
	}
	return -1
}

func (c *Client) openIndex() int {
	for i := range c.st.Attendances {
		if c.st.Attendances[i].CheckOut == nil {
			return i
		}
	}
	return -1
}

// notAfter is the client-side check of odoo.XMLRPCClient's explicit-time
// methods: check-out must be strictly after check-in.
func notAfter(in, out time.Time) error {
	if !out.After(in) {
		return apperr.New(apperr.KindValidation, "check-out %s is not after check-in %s",
			out.Format("15:04"), in.Format("15:04"))
	}
	return nil
}

// checkValidity mirrors Odoo's hr.attendance constraints: check-out not
// before check-in, at most one open record, no overlap with another record.
func (c *Client) checkValidity(r odoo.AttendanceRecord) error {
	if r.CheckOut != nil && r.CheckOut.Before(r.CheckIn) {
		return apperr.New(apperr.KindValidation, "check-out %s is before check-in %s",
			r.CheckOut.Format("15:04"), r.CheckIn.Format("15:04"))
	}
	for _, o := range c.st.Attendances {
		if o.ID == r.ID {
			continue
		}
		if r.CheckOut == nil && o.CheckOut == nil {
			return apperr.New(apperr.KindValidation,
				"Cannot create new attendance record for %s, the employee hasn't checked out since %s",
				c.st.User.Name, o.CheckIn.Format("2006-01-02 15:04"))
		}
		if overlaps(r, o) {
			return apperr.New(apperr.KindValidation,
				"Cannot create new attendance record for %s, the employee was already checked in on %s",
				c.st.User.Name, o.CheckIn.Format("2006-01-02 15:04"))
		}
	}
	return nil
}

// overlaps reports whether two periods intersect; an open period extends
// to infinity.
func overlaps(a, b odoo.AttendanceRecord) bool {
	aEndsAfterB := a.CheckOut == nil || a.CheckOut.After(b.CheckIn)
	bEndsAfterA := b.CheckOut == nil || b.CheckOut.After(a.CheckIn)
	return aEndsAfterB && bEndsAfterA
}

func withHours(r odoo.AttendanceRecord) odoo.AttendanceRecord {
	if r.CheckOut != nil {
		r.WorkedHours = r.CheckOut.Sub(r.CheckIn).Hours()
	} else {
		r.WorkedHours = 0
	}
	return r
}

// insert validates r, assigns an ID and saves.
func (c *Client) insert(in time.Time, out *time.Time) (int64, error) {
	r := odoo.AttendanceRecord{
		EmployeeID: c.st.EmployeeID, Employee: c.st.User.Name, CheckIn: odooTime(in),
	}
	if out != nil {
		o := odooTime(*out)
		r.CheckOut = &o
	}
	if err := c.checkValidity(r); err != nil {
		return 0, err
	}
	r.ID = c.nextID()
	c.st.Attendances = append(c.st.Attendances, withHours(r))
	return r.ID, c.save()
}

// closeOpen sets the check-out of the open record.
func (c *Client) closeOpen(out time.Time) (*odoo.AttendanceRecord, error) {
	i := c.openIndex()
	if i < 0 {
		return nil, apperr.New(apperr.KindConflict, "not clocked in")
	}
	r := c.st.Attendances[i]
	o := odooTime(out)
	r.CheckOut = &o
	if err := c.checkValidity(r); err != nil {
		return nil, err
	}
	r = withHours(r)
	c.st.Attendances[i] = r
	if err := c.save(); err != nil {
		return nil, err
	}
	return &r, nil
}

// ClockIn implements odoo.Client (the web-session path, at now).
func (c *Client) ClockIn() (int64, error) { return c.ClockInAt(c.now()) }

// ClockOut implements odoo.Client (the web-session path, at now).
func (c *Client) ClockOut() (*odoo.AttendanceRecord, error) { return c.closeOpen(c.now()) }

// ClockInAt implements odoo.Client.
func (c *Client) ClockInAt(in time.Time) (int64, error) {
	if c.openIndex() >= 0 {
		return 0, apperr.New(apperr.KindConflict, "already clocked in")
	}
	return c.insert(in, nil)
}

// ClockOutAt implements odoo.Client.
func (c *Client) ClockOutAt(out time.Time) (*odoo.AttendanceRecord, error) {
	i := c.openIndex()
	if i < 0 {
		return nil, apperr.New(apperr.KindConflict, "not clocked in")
	}
	if err := notAfter(c.st.Attendances[i].CheckIn, out); err != nil {
		return nil, err
	}
	return c.closeOpen(out)
}

// CreateAttendance implements odoo.Client.
func (c *Client) CreateAttendance(in, out time.Time) (int64, error) {
	if err := notAfter(in, out); err != nil {
		return 0, err
	}
	return c.insert(in, &out)
}

// EditAttendance implements odoo.Client.
func (c *Client) EditAttendance(id int64, in, out *time.Time) (*odoo.AttendanceRecord, error) {
	if id <= 0 {
		return nil, apperr.New(apperr.KindValidation, "attendance ID is required")
	}
	if in == nil && out == nil {
		return nil, apperr.New(apperr.KindValidation, "nothing to update: provide a new check-in and/or check-out time")
	}
	if in != nil && out != nil {
		if err := notAfter(*in, *out); err != nil {
			return nil, err
		}
	}
	i := c.attendanceIndex(id)
	if i < 0 {
		return nil, apperr.New(apperr.KindNotFound, "attendance record %d not found", id)
	}
	r := c.st.Attendances[i]
	if in != nil {
		r.CheckIn = odooTime(*in)
	}
	if out != nil {
		o := odooTime(*out)
		r.CheckOut = &o
	}
	if err := c.checkValidity(r); err != nil {
		return nil, err
	}
	r = withHours(r)
	c.st.Attendances[i] = r
	if err := c.save(); err != nil {
		return nil, err
	}
	return &r, nil
}

// GetAttendance implements odoo.Client.
func (c *Client) GetAttendance(id int64) (*odoo.AttendanceRecord, error) {
	if i := c.attendanceIndex(id); i >= 0 {
		r := c.st.Attendances[i]
		return &r, nil
	}
	return nil, apperr.New(apperr.KindNotFound, "attendance record %d not found", id)
}

// AttendanceStatus implements odoo.Client: today's records (UTC day, as
// the real client queries) plus an open record from an earlier day.
func (c *Client) AttendanceStatus() (*odoo.AttendanceStatus, error) {
	now := c.now()
	u := now.UTC()
	day := time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
	recs, _ := c.ListAttendance(day, day.AddDate(0, 0, 1))
	st := &odoo.AttendanceStatus{Periods: recs}
	for _, r := range recs {
		if r.CheckOut == nil {
			st.ClockedIn = true
			st.CurrentID = r.ID
			in := r.CheckIn
			st.CheckIn = &in
			st.TotalHours += now.Sub(r.CheckIn).Hours()
		} else {
			st.TotalHours += r.WorkedHours
		}
	}
	return st, nil
}

// ListAttendance implements odoo.Client: records with check-in in
// [from, to) plus open records that started before from, newest first.
func (c *Client) ListAttendance(from, to time.Time) ([]odoo.AttendanceRecord, error) {
	var open, inRange []odoo.AttendanceRecord
	for _, r := range c.st.Attendances {
		switch {
		case !r.CheckIn.Before(from) && r.CheckIn.Before(to):
			inRange = append(inRange, r)
		case r.CheckIn.Before(from) && r.CheckOut == nil:
			open = append(open, r)
		}
	}
	sort.SliceStable(inRange, func(i, j int) bool { return inRange[i].CheckIn.After(inRange[j].CheckIn) })
	return append(open, inRange...), nil
}
