package tui

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/parsing"
)

// attendanceFutureGrace tolerates small clock skew when rejecting future
// times, mirroring the clock CLI commands.
const attendanceFutureGrace = time.Minute

// attendanceSubState is the step within the attendance overlay.
type attendanceSubState int

const (
	attLoading attendanceSubState = iota // fetching the day's records
	attPick                              // several records: choose one
	attForm                              // edit (or create) check-in/out times
)

// attendanceDayLoadedMsg is sent when the selected day's attendance records
// finish loading.
type attendanceDayLoadedMsg struct {
	records []odoo.AttendanceRecord
	err     error
}

// attendanceSavedMsg is sent when an attendance create/edit completes.
type attendanceSavedMsg struct {
	err error
}

// attendanceState holds the attendance overlay's state (issue #47).
type attendanceState struct {
	sub        attendanceSubState
	day        time.Time               // midnight of the selected day
	records    []odoo.AttendanceRecord // the day's records
	pickCursor int                     // selected row in the pick list
	record     *odoo.AttendanceRecord  // record being edited; nil = creating
	inInput    textinput.Model
	outInput   textinput.Model
	focus      int // 0 = check-in, 1 = check-out
	err        error
	prevState  uiState // state to return to when the overlay closes
}

// isNew reports whether the form creates a record (the day had none).
func (a attendanceState) isNew() bool { return a.record == nil }

// attendanceChange describes the write derived from the form inputs.
type attendanceChange struct {
	create   bool
	id       int64
	checkIn  *time.Time // nil = unchanged
	checkOut *time.Time // nil = unchanged
}

// parseAttendanceTime parses a form value ("HH:MM" on day, or
// "YYYY-MM-DD HH:MM") in day's location and rejects times after now.
func parseAttendanceTime(s string, day, now time.Time) (time.Time, error) {
	t, err := parsing.ParseClockTime(strings.TrimSpace(s), day)
	if err != nil {
		return time.Time{}, err
	}
	if t.After(now.Add(attendanceFutureGrace)) {
		return time.Time{}, fmt.Errorf("time %s is in the future", t.Format("2006-01-02 15:04"))
	}
	return t, nil
}

// buildAttendanceChange validates the form inputs against the record being
// edited (nil when creating) and returns the resulting write. Times equal to
// the record's current minute are reported as unchanged so an untouched
// field is not rewritten (which would drop Odoo's stored seconds).
func buildAttendanceChange(record *odoo.AttendanceRecord, inStr, outStr string, day, now time.Time) (attendanceChange, error) {
	inStr = strings.TrimSpace(inStr)
	outStr = strings.TrimSpace(outStr)

	if record == nil {
		if inStr == "" || outStr == "" {
			return attendanceChange{}, errors.New("check-in and check-out are both required to create a record")
		}
		checkIn, err := parseAttendanceTime(inStr, day, now)
		if err != nil {
			return attendanceChange{}, err
		}
		checkOut, err := parseAttendanceTime(outStr, day, now)
		if err != nil {
			return attendanceChange{}, err
		}
		if err := checkOutAfterCheckIn(checkIn, checkOut); err != nil {
			return attendanceChange{}, err
		}
		return attendanceChange{create: true, checkIn: &checkIn, checkOut: &checkOut}, nil
	}

	if inStr == "" {
		return attendanceChange{}, errors.New("check-in is required")
	}
	if outStr == "" && record.CheckOut != nil {
		return attendanceChange{}, errors.New("check-out is required (the record is closed)")
	}

	change := attendanceChange{id: record.ID}
	effectiveIn := record.CheckIn
	checkIn, err := parseAttendanceTime(inStr, day, now)
	if err != nil {
		return attendanceChange{}, err
	}
	if !checkIn.Equal(record.CheckIn.Truncate(time.Minute)) {
		change.checkIn = &checkIn
		effectiveIn = checkIn
	}

	var effectiveOut *time.Time
	if outStr != "" {
		checkOut, err := parseAttendanceTime(outStr, day, now)
		if err != nil {
			return attendanceChange{}, err
		}
		if record.CheckOut == nil || !checkOut.Equal(record.CheckOut.Truncate(time.Minute)) {
			change.checkOut = &checkOut
		}
		effectiveOut = &checkOut
	} else if record.CheckOut != nil {
		effectiveOut = record.CheckOut
	}

	if change.checkIn == nil && change.checkOut == nil {
		return attendanceChange{}, errors.New("nothing to change")
	}
	if effectiveOut != nil {
		if err := checkOutAfterCheckIn(effectiveIn, *effectiveOut); err != nil {
			return attendanceChange{}, err
		}
	}
	return change, nil
}

func checkOutAfterCheckIn(checkIn, checkOut time.Time) error {
	if !checkOut.After(checkIn) {
		return fmt.Errorf("check-out %s is not after check-in %s",
			checkOut.Format("15:04"), checkIn.In(checkOut.Location()).Format("15:04"))
	}
	return nil
}

// attendanceDay returns midnight of the grid cursor's day in the displayed
// week's location.
func (m Model) attendanceDay() time.Time {
	mon := m.monday.Time
	return time.Date(mon.Year(), mon.Month(), mon.Day()+m.cursor[1], 0, 0, 0, 0, mon.Location())
}

// enterAttendance opens the attendance overlay for the cursor's day and
// starts loading that day's records.
func (m Model) enterAttendance() (tea.Model, tea.Cmd) {
	day := m.attendanceDay()
	m.att = attendanceState{sub: attLoading, day: day, prevState: m.state}
	m.state = stateAttendance
	m.loading = true

	client := m.client
	from, to := day.UTC(), day.AddDate(0, 0, 1).UTC()
	return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
		records, err := client.ListAttendance(from, to)
		return attendanceDayLoadedMsg{records: records, err: err}
	})
}

// handleAttendanceDayLoaded routes the loaded records to the pick list or
// straight into the form.
func (m Model) handleAttendanceDayLoaded(msg attendanceDayLoadedMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if m.state != stateAttendance {
		return m, nil
	}
	if msg.err != nil {
		// Keep the overlay open so the error is visible and Esc still works.
		m.att.err = fmt.Errorf("loading attendance: %w", msg.err)
		return m.openAttendanceForm(nil)
	}
	// Odoo returns hr.attendance newest-first; show the day chronologically.
	records := append([]odoo.AttendanceRecord(nil), msg.records...)
	sort.Slice(records, func(i, j int) bool { return records[i].CheckIn.Before(records[j].CheckIn) })
	m.att.records = records
	switch len(records) {
	case 0:
		return m.openAttendanceForm(nil)
	case 1:
		return m.openAttendanceForm(&m.att.records[0])
	default:
		m.att.sub = attPick
		m.att.pickCursor = 0
		return m, nil
	}
}

// openAttendanceForm shows the edit form for record (nil = create).
func (m Model) openAttendanceForm(record *odoo.AttendanceRecord) (tea.Model, tea.Cmd) {
	m.att.sub = attForm
	m.att.record = record
	m.att.focus = 0

	m.att.inInput = textinput.New()
	m.att.inInput.SetWidth(20)
	m.att.inInput.Placeholder = "HH:MM"
	m.att.outInput = textinput.New()
	m.att.outInput.SetWidth(20)
	m.att.outInput.Placeholder = "HH:MM"
	if record != nil {
		m.att.inInput.SetValue(formatClockTime(record.CheckIn, m.att.day))
		if record.CheckOut != nil {
			m.att.outInput.SetValue(formatClockTime(*record.CheckOut, m.att.day))
		}
	}
	m.att.outInput.Blur()
	cmd := m.att.inInput.Focus()
	return m, cmd
}

// formatClockTime renders t as the form expects it: "HH:MM" when t falls
// on day, otherwise "YYYY-MM-DD HH:MM" (e.g. an overnight check-out).
func formatClockTime(t, day time.Time) string {
	local := t.In(day.Location())
	if local.Year() == day.Year() && local.YearDay() == day.YearDay() {
		return local.Format("15:04")
	}
	return local.Format("2006-01-02 15:04")
}

// closeAttendance returns to the state the overlay was opened from.
func (m Model) closeAttendance() (tea.Model, tea.Cmd) {
	m.state = m.att.prevState
	m.att.err = nil
	return m, nil
}

// updateAttendance handles key events while the attendance overlay is open.
func (m Model) updateAttendance(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.att.sub {
	case attLoading:
		if key.Matches(msg, m.keys.Back) {
			return m.closeAttendance()
		}
		return m, nil
	case attPick:
		return m.updateAttendancePick(msg)
	default:
		return m.updateAttendanceForm(msg)
	}
}

func (m Model) updateAttendancePick(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		return m.closeAttendance()
	case key.Matches(msg, m.keys.Up):
		if m.att.pickCursor > 0 {
			m.att.pickCursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.att.pickCursor < len(m.att.records)-1 {
			m.att.pickCursor++
		}
	case msg.Code == tea.KeyEnter:
		if m.att.pickCursor < len(m.att.records) {
			return m.openAttendanceForm(&m.att.records[m.att.pickCursor])
		}
	}
	return m, nil
}

func (m Model) updateAttendanceForm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		if len(m.att.records) > 1 {
			m.att.sub = attPick
			m.att.err = nil
			return m, nil
		}
		return m.closeAttendance()

	case msg.Code == tea.KeyTab:
		if m.att.focus == 0 {
			m.att.focus = 1
			m.att.inInput.Blur()
			return m, m.att.outInput.Focus()
		}
		m.att.focus = 0
		m.att.outInput.Blur()
		return m, m.att.inInput.Focus()

	case msg.Code == tea.KeyEnter:
		return m.submitAttendance()
	}

	var cmd tea.Cmd
	if m.att.focus == 0 {
		m.att.inInput, cmd = m.att.inInput.Update(msg)
	} else {
		m.att.outInput, cmd = m.att.outInput.Update(msg)
	}
	return m, cmd
}

// submitAttendance validates the form and fires the create/edit call.
func (m Model) submitAttendance() (tea.Model, tea.Cmd) {
	change, err := buildAttendanceChange(m.att.record, m.att.inInput.Value(), m.att.outInput.Value(), m.att.day, m.now())
	if err != nil {
		m.att.err = err
		return m, nil
	}
	m.att.err = nil

	client := m.client
	return m, func() tea.Msg {
		if change.create {
			_, err := client.CreateAttendance(*change.checkIn, *change.checkOut)
			return attendanceSavedMsg{err: err}
		}
		_, err := client.EditAttendance(change.id, change.checkIn, change.checkOut)
		return attendanceSavedMsg{err: err}
	}
}

// handleAttendanceSaved closes the overlay and refreshes the header totals
// on success; on failure the form stays open showing the error.
func (m Model) handleAttendanceSaved(msg attendanceSavedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.att.err = msg.err
		return m, nil
	}
	m.state = m.att.prevState
	return m, m.loadAttendance()
}

// attendanceRecordLine renders one record for the pick list and form header.
func attendanceRecordLine(rec odoo.AttendanceRecord, day time.Time) string {
	in := formatClockTime(rec.CheckIn, day)
	if rec.CheckOut == nil {
		return fmt.Sprintf("#%d  %s – --:-- (running)", rec.ID, in)
	}
	return fmt.Sprintf("#%d  %s – %s (%s)", rec.ID, in,
		formatClockTime(*rec.CheckOut, day), formatHoursOrZero(rec.WorkedHours))
}

// renderAttendanceOverlay renders the attendance overlay content for the
// current sub-state.
func renderAttendanceOverlay(a attendanceState, spin spinner.Model) string {
	dayStr := a.day.Format("Mon 02 Jan 2006")
	var b strings.Builder

	switch a.sub {
	case attLoading:
		b.WriteString(detailHeaderStyle.Render("Attendance — " + dayStr))
		b.WriteString("\n\n")
		fmt.Fprintf(&b, "  %s Loading attendance...", spin.View())
		b.WriteString("\n\n")
		b.WriteString(detailHintStyle.Render("Esc: cancel"))

	case attPick:
		b.WriteString(detailHeaderStyle.Render("Attendance — " + dayStr))
		b.WriteString("\n\n")
		b.WriteString("  Several records on this day, select one to edit:\n\n")
		for i, rec := range a.records {
			line := "  " + attendanceRecordLine(rec, a.day) + "  "
			if i == a.pickCursor {
				line = cursorStyle.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(detailHintStyle.Render("↑/↓: select  Enter: edit  Esc: cancel"))

	default:
		verb := "Edit attendance"
		if a.isNew() {
			verb = "Add attendance"
		}
		b.WriteString(detailHeaderStyle.Render(verb + " — " + dayStr))
		b.WriteString("\n")
		if a.isNew() {
			b.WriteString(detailHintStyle.Render("  This day has no attendance record yet; enter both times to create one."))
		} else {
			b.WriteString(detailHintStyle.Render("  Record " + attendanceRecordLine(*a.record, a.day)))
		}
		b.WriteString("\n\n")

		inLabel, outLabel := "  Check-in:   ", "  Check-out:  "
		if a.focus == 0 {
			inLabel = editActiveLabelStyle.Render(inLabel)
			outLabel = editLabelStyle.Render(outLabel)
		} else {
			inLabel = editLabelStyle.Render(inLabel)
			outLabel = editActiveLabelStyle.Render(outLabel)
		}
		b.WriteString(inLabel + a.inInput.View() + "\n")
		b.WriteString(outLabel + a.outInput.View() + "\n")

		if a.err != nil {
			b.WriteString("\n")
			b.WriteString(renderAttendanceError(a.err))
		}

		b.WriteString("\n")
		b.WriteString(detailHintStyle.Render("Enter: save  Esc: cancel  Tab: next field   HH:MM or YYYY-MM-DD HH:MM"))
	}

	return b.String()
}

// attendanceErrorWidth wraps long Odoo error messages inside the overlay.
const attendanceErrorWidth = 72

// renderAttendanceError renders the error message in the error style and,
// for Odoo AccessErrors, the officer-rights hint in the hint style. A hint
// the client already appended ("\nhint: ...") is split off so it appears
// exactly once.
func renderAttendanceError(err error) string {
	msg := err.Error()
	hint := ""
	if i := strings.Index(msg, "\nhint: "); i >= 0 {
		hint = msg[i+len("\nhint: "):]
		msg = msg[:i]
	} else if odoo.IsAccessError(err) {
		hint = odoo.AttendanceAccessHint
	}

	var b strings.Builder
	for i, line := range wrapText("Error: "+strings.TrimSpace(msg), attendanceErrorWidth) {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(editErrorStyle.Render("  " + line))
	}
	b.WriteString("\n")
	if hint != "" {
		for _, line := range wrapText("Hint: "+hint, attendanceErrorWidth) {
			b.WriteString(editHintStyle.Render("  " + line))
			b.WriteString("\n")
		}
	}
	return b.String()
}

// wrapText word-wraps s to width, preserving explicit line breaks and
// dropping blank lines.
func wrapText(s string, width int) []string {
	var out []string
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			continue
		}
		line := ""
		for _, w := range words {
			switch {
			case line == "":
				line = w
			case len([]rune(line))+1+len([]rune(w)) > width:
				out = append(out, line)
				line = w
			default:
				line += " " + w
			}
		}
		out = append(out, line)
	}
	return out
}
