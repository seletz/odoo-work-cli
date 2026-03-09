package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

type uiState int

const (
	stateLoading uiState = iota
	stateGrid
	stateDetail
	stateError
	stateEdit
)

// timesheetsLoadedMsg is sent when timesheets finish loading.
type timesheetsLoadedMsg struct {
	entries     []odoo.TimesheetEntry
	prevEntries []odoo.TimesheetEntry
	err         error
}

// attendanceLoadedMsg is sent when attendance status finishes loading.
type attendanceLoadedMsg struct {
	status *odoo.AttendanceStatus
	err    error
}

// editSavedMsg is sent when an edit save operation completes.
type editSavedMsg struct {
	err error
}

// deleteEntryMsg is sent when a delete operation completes.
type deleteEntryMsg struct {
	err error
}

// attendanceTickMsg triggers a re-render to update elapsed clock-in time.
type attendanceTickMsg time.Time

// Model is the bubbletea model for the weekly timesheet TUI.
type Model struct {
	state        uiState
	client       odoo.Client
	grid         WeekGrid
	monday       MondayTime
	cursor       [2]int // [row, col]
	spinner      spinner.Model
	help         help.Model
	keys         KeyMap
	limits       config.HoursLimits
	bundesland   string
	holidays     HolidayMap
	weekHols     [7]string
	attendance   *odoo.AttendanceStatus
	loading      bool
	err          error
	width        int
	height       int
	detailCursor int             // selected entry row in detail view
	editIndex    int             // index into current day's entries slice
	editHours    textinput.Model // hours input
	editDesc     textinput.Model // description input
	editFocus    int             // 0=hours, 1=description
	editErr      error           // last edit error
	editIsNew    bool            // true = creating new entry, false = editing existing
}

// MondayTime wraps time.Time for the Monday of the displayed week.
type MondayTime struct {
	time.Time
}

// NewModel creates a new TUI model with the given client and starting Monday.
func NewModel(client odoo.Client, monday MondayTime, limits config.HoursLimits, bundesland string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return Model{
		state:      stateLoading,
		client:     client,
		monday:     monday,
		limits:     limits,
		bundesland: bundesland,
		holidays:   BuildHolidayMap(monday.Year(), bundesland),
		spinner:    s,
		help:       help.New(),
		keys:       DefaultKeyMap(),
	}
}

// Init starts the spinner and triggers the initial data load.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadTimesheets(), m.loadAttendance())
}

func (m Model) loadTimesheets() tea.Cmd {
	monday := m.monday.Time
	prevMonday := monday.AddDate(0, 0, -7)
	sunday := monday.AddDate(0, 0, 6)
	dateFrom := prevMonday.Format("2006-01-02")
	dateTo := sunday.Format("2006-01-02")
	client := m.client
	mondayStr := monday.Format("2006-01-02")
	return func() tea.Msg {
		entries, err := client.ListTimesheets(dateFrom, dateTo)
		if err != nil {
			return timesheetsLoadedMsg{err: err}
		}
		var current, prev []odoo.TimesheetEntry
		for _, e := range entries {
			if e.Date >= mondayStr {
				current = append(current, e)
			} else {
				prev = append(prev, e)
			}
		}
		return timesheetsLoadedMsg{entries: current, prevEntries: prev}
	}
}

func (m Model) loadAttendance() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		status, err := client.AttendanceStatus()
		return attendanceLoadedMsg{status: status, err: err}
	}
}

func attendanceTick() tea.Cmd {
	return tea.Tick(time.Minute, func(t time.Time) tea.Msg {
		return attendanceTickMsg(t)
	})
}

// Update handles messages and updates the model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case timesheetsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		// Preserve detail state when reloading after edit save.
		if m.state != stateDetail {
			m.state = stateGrid
			m.cursor = [2]int{0, 0}
		}
		hints := HintLabelsFromEntries(msg.prevEntries)
		m.grid = BuildWeekGridWithHints(msg.entries, m.monday.Time, hints)
		m.holidays = BuildHolidayMap(m.monday.Year(), m.bundesland)
		m.weekHols = WeekHolidays(m.monday.Time, m.holidays)
		return m, nil

	case attendanceLoadedMsg:
		if msg.err == nil {
			m.attendance = msg.status
		}
		if m.attendance != nil && m.attendance.ClockedIn {
			return m, attendanceTick()
		}
		return m, nil

	case attendanceTickMsg:
		if m.attendance != nil && m.attendance.ClockedIn {
			return m, attendanceTick()
		}
		return m, nil

	case editSavedMsg:
		if msg.err != nil {
			m.editErr = msg.err
			return m, nil
		}
		m.state = stateDetail
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.loadTimesheets())

	case deleteEntryMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.state = stateDetail
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.loadTimesheets())

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.SetWidth(msg.Width)
		return m, nil

	case spinner.TickMsg:
		if m.state == stateLoading || m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyPressMsg:
		// In edit state, forward keys to text inputs.
		if m.state == stateEdit {
			return m.updateEdit(msg)
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil

		case key.Matches(msg, m.keys.Refresh):
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadTimesheets(), m.loadAttendance())

		case key.Matches(msg, m.keys.Back):
			if m.state == stateDetail {
				m.state = stateGrid
				return m, nil
			}

		case key.Matches(msg, m.keys.Left):
			if m.state == stateDetail {
				m.state = stateGrid
			}
			m.monday = MondayTime{m.monday.AddDate(0, 0, -7)}
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadTimesheets())

		case key.Matches(msg, m.keys.Right):
			if m.state == stateDetail {
				m.state = stateGrid
			}
			m.monday = MondayTime{m.monday.AddDate(0, 0, 7)}
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadTimesheets())
		}

		if m.state == stateGrid {
			switch {
			case key.Matches(msg, m.keys.Enter):
				if m.cursor[0] < len(m.grid.Rows) {
					m.state = stateDetail
					m.detailCursor = 0
					m.loading = true
					return m, tea.Batch(m.spinner.Tick, m.loadTimesheets())
				}
				return m, nil
			case key.Matches(msg, m.keys.Up):
				if m.cursor[0] > 0 {
					m.cursor[0]--
				}
			case key.Matches(msg, m.keys.Down):
				if m.cursor[0] < len(m.grid.Rows)-1 {
					m.cursor[0]++
				}
			case key.Matches(msg, m.keys.NextCol):
				if m.cursor[1] < 6 {
					m.cursor[1]++
				}
			case key.Matches(msg, m.keys.PrevCol):
				if m.cursor[1] > 0 {
					m.cursor[1]--
				}
			}
		}

		if m.state == stateDetail {
			entries := m.detailEntries()
			switch {
			case key.Matches(msg, m.keys.Up):
				if m.detailCursor > 0 {
					m.detailCursor--
				}
				return m, nil
			case key.Matches(msg, m.keys.Down):
				if m.detailCursor < len(entries)-1 {
					m.detailCursor++
				}
				return m, nil
			case key.Matches(msg, m.keys.Edit):
				if len(entries) > 0 {
					return m.enterEdit()
				}
				return m, nil
			case key.Matches(msg, m.keys.Add):
				return m.enterAdd()
			case key.Matches(msg, m.keys.Delete):
				if len(entries) > 0 {
					return m.deleteEntry()
				}
				return m, nil
			}
		}

		return m, nil
	}

	return m, nil
}

// detailEntries returns the entries for the currently selected grid cell.
func (m Model) detailEntries() []odoo.TimesheetEntry {
	if m.cursor[0] >= len(m.grid.Rows) {
		return nil
	}
	return m.grid.Rows[m.cursor[0]].Entries[m.cursor[1]]
}

// enterEdit transitions from detail to edit state, pre-filling inputs.
func (m Model) enterEdit() (tea.Model, tea.Cmd) {
	entries := m.detailEntries()
	if m.detailCursor >= len(entries) {
		return m, nil
	}
	entry := entries[m.detailCursor]

	m.editIsNew = false
	m.editIndex = m.detailCursor
	m.editFocus = 0
	m.editErr = nil

	m.editHours = textinput.New()
	m.editHours.SetValue(formatDecimalHours(entry.Hours))
	m.editHours.SetWidth(10)
	m.editHours.Placeholder = "0.0"
	cmd := m.editHours.Focus()

	m.editDesc = textinput.New()
	m.editDesc.SetValue(entry.Name)
	m.editDesc.SetWidth(50)
	m.editDesc.Placeholder = "Description"
	m.editDesc.Blur()

	m.state = stateEdit
	return m, cmd
}

// enterAdd transitions from detail to edit state for creating a new entry.
func (m Model) enterAdd() (tea.Model, tea.Cmd) {
	m.editIsNew = true
	m.editIndex = -1
	m.editFocus = 0
	m.editErr = nil

	m.editHours = textinput.New()
	m.editHours.SetWidth(10)
	m.editHours.Placeholder = "0.0"
	cmd := m.editHours.Focus()

	m.editDesc = textinput.New()
	m.editDesc.SetWidth(50)
	m.editDesc.Placeholder = "Description"
	m.editDesc.Blur()

	m.state = stateEdit
	return m, cmd
}

// updateEdit handles key events in the edit state.
func (m Model) updateEdit(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.state = stateDetail
		m.editErr = nil
		return m, nil

	case msg.Code == '\t':
		// Toggle focus between hours and description.
		if m.editFocus == 0 {
			m.editFocus = 1
			m.editHours.Blur()
			cmd := m.editDesc.Focus()
			return m, cmd
		}
		m.editFocus = 0
		m.editDesc.Blur()
		cmd := m.editHours.Focus()
		return m, cmd

	case msg.Code == '\r' || msg.Code == '\n':
		return m.submitEdit()
	}

	// Forward to focused input.
	var cmd tea.Cmd
	if m.editFocus == 0 {
		m.editHours, cmd = m.editHours.Update(msg)
	} else {
		m.editDesc, cmd = m.editDesc.Update(msg)
	}
	return m, cmd
}

// submitEdit validates and saves the edit.
func (m Model) submitEdit() (tea.Model, tea.Cmd) {
	hours, err := strconv.ParseFloat(strings.TrimSpace(m.editHours.Value()), 64)
	if err != nil || hours <= 0 {
		m.editErr = fmt.Errorf("hours must be a positive number")
		return m, nil
	}
	desc := strings.TrimSpace(m.editDesc.Value())
	if desc == "" {
		m.editErr = fmt.Errorf("description cannot be empty")
		return m, nil
	}

	client := m.client

	if m.editIsNew {
		row := m.grid.Rows[m.cursor[0]]
		projectID, taskID := row.ProjectTaskIDs()
		day := m.monday.AddDate(0, 0, m.cursor[1])
		params := odoo.TimesheetWriteParams{
			ProjectID: projectID,
			TaskID:    taskID,
			Date:      day.Format("2006-01-02"),
			Name:      desc,
			Hours:     hours,
		}
		return m, func() tea.Msg {
			_, err := client.CreateTimesheet(params)
			return editSavedMsg{err: err}
		}
	}

	entries := m.detailEntries()
	if m.editIndex >= len(entries) {
		return m, nil
	}
	entry := entries[m.editIndex]

	fields := map[string]interface{}{
		"unit_amount": hours,
		"name":        desc,
	}

	id := entry.ID
	return m, func() tea.Msg {
		err := client.UpdateTimesheet(id, fields)
		return editSavedMsg{err: err}
	}
}

// deleteEntry deletes the currently selected entry in the detail view.
func (m Model) deleteEntry() (tea.Model, tea.Cmd) {
	entries := m.detailEntries()
	if m.detailCursor >= len(entries) {
		return m, nil
	}
	client := m.client
	id := entries[m.detailCursor].ID
	return m, func() tea.Msg {
		err := client.DeleteTimesheet(id)
		return deleteEntryMsg{err: err}
	}
}

// formatDecimalHours formats hours as a decimal string (e.g. "2.5").
func formatDecimalHours(h float64) string {
	return strconv.FormatFloat(h, 'f', -1, 64)
}

// View renders the TUI.
func (m Model) View() tea.View {
	var s string

	switch m.state {
	case stateLoading:
		week := m.monday.Format("2006-01-02")
		s = fmt.Sprintf("\n  %s Loading timesheets for week of %s...\n\n", m.spinner.View(), week)

	case stateError:
		s = fmt.Sprintf("\n  Error: %s\n\n  Press 'r' to retry or 'q' to quit.\n\n", m.err)

	case stateGrid, stateDetail, stateEdit:
		sunday := m.monday.AddDate(0, 0, 6)
		loadingIndicator := ""
		if m.loading {
			loadingIndicator = " " + m.spinner.View()
		}
		_, isoWeek := m.monday.ISOWeek()
		clockStatus := renderClockStatus(m.attendance)
		if clockStatus != "" {
			clockStatus = "  " + clockStatus
		}
		title := fmt.Sprintf("  W%02d: %s — %s%s%s\n\n",
			isoWeek,
			m.monday.Format("Mon 02 Jan 2006"),
			sunday.Format("Mon 02 Jan 2006"),
			loadingIndicator,
			clockStatus)
		grid := RenderGrid(m.grid, m.cursor[0], m.cursor[1], m.width-4, m.limits, m.weekHols)

		helpView := m.help.View(m.keys)
		s = "\n" + title + grid + "\n  " + helpView + "\n"

		if m.state == stateEdit && m.cursor[0] < len(m.grid.Rows) {
			row := m.grid.Rows[m.cursor[0]]
			day := m.monday.AddDate(0, 0, m.cursor[1])
			edit := renderEditForm(row, day, m.editHours, m.editDesc, m.editFocus, m.editErr, m.width, m.editIsNew)
			s = RenderDetailOverlay(s, edit, m.width, m.height)
		} else if m.state == stateDetail && m.cursor[0] < len(m.grid.Rows) {
			detail := RenderDetail(m.grid.Rows[m.cursor[0]], m.cursor[1], m.monday.Time, m.detailCursor, m.width)
			s = RenderDetailOverlay(s, detail, m.width, m.height)
		}
	}

	// Pad to full terminal height so alt screen doesn't show artifacts.
	lines := strings.Count(s, "\n")
	if m.height > 0 && lines < m.height {
		s += strings.Repeat("\n", m.height-lines)
	}

	v := tea.NewView(s)
	v.AltScreen = true
	return v
}
