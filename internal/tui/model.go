package tui

import (
	"fmt"
	"sort"
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
	stateSearch
	stateHelp
)

type searchSubState int

const (
	searchLoading searchSubState = iota
	searchReady
)

// searchItem represents a unified search result (project or task).
type searchItem struct {
	Kind      string // "project" or "task"
	ID        int64
	Name      string
	Extra     string // company for projects, project name for tasks
	ProjectID int64  // for tasks: the parent project ID
	TaskID    int64  // 0 for projects, task ID for tasks
}

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

// searchDataLoadedMsg is sent when search data finishes loading.
type searchDataLoadedMsg struct {
	projects []odoo.ProjectInfo
	tasks    []odoo.TaskInfo
	err      error
}

// clockToggleMsg is sent when a clock in or out operation completes.
type clockToggleMsg struct {
	status *odoo.AttendanceStatus
	err    error
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

	helpPrevState uiState // state to return to when exiting help

	companyColors map[string]string // company name → lipgloss color

	pendingRows []GridRow // rows added via search that have no server entries yet

	searchSub       searchSubState
	searchInput     textinput.Model
	searchItems     []searchItem // full combined list
	searchFiltered  []searchItem // after text filter
	searchCursor    int
	searchUseFilter bool // true = config filters active (default)
	searchErr       error
}

// MondayTime wraps time.Time for the Monday of the displayed week.
type MondayTime struct {
	time.Time
}

// NewModel creates a new TUI model with the given client and starting Monday.
func NewModel(client odoo.Client, monday MondayTime, limits config.HoursLimits, bundesland string, keys config.KeysConfig, companyColors map[string]string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	km := DefaultKeyMap()
	if keys != nil {
		km = ApplyKeysConfig(km, keys)
	}
	return Model{
		state:         stateLoading,
		client:        client,
		monday:        monday,
		limits:        limits,
		bundesland:    bundesland,
		holidays:      BuildHolidayMap(monday.Year(), bundesland),
		companyColors: companyColors,
		spinner:       s,
		help:          help.New(),
		keys:          km,
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

// toggleClock clocks in or out depending on the current attendance state.
func (m Model) toggleClock() tea.Cmd {
	client := m.client
	clockedIn := m.attendance != nil && m.attendance.ClockedIn
	return func() tea.Msg {
		if clockedIn {
			_, err := client.ClockOut()
			if err != nil {
				return clockToggleMsg{err: err}
			}
		} else {
			_, err := client.ClockIn()
			if err != nil {
				return clockToggleMsg{err: err}
			}
		}
		status, err := client.AttendanceStatus()
		return clockToggleMsg{status: status, err: err}
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
			m.cursor = [2]int{0, TodayColumn(m.monday.Time, time.Now())}
		}
		hints := HintLabelsFromEntries(msg.prevEntries)
		m.grid = BuildWeekGridWithHints(msg.entries, m.monday.Time, hints)
		m.holidays = BuildHolidayMap(m.monday.Year(), m.bundesland)
		m.weekHols = WeekHolidays(m.monday.Time, m.holidays)

		// Re-inject pending rows from search that have no server entries yet,
		// and prune those that now appear in the grid naturally.
		gridLabels := make(map[string]bool, len(m.grid.Rows))
		for _, row := range m.grid.Rows {
			gridLabels[row.Label] = true
		}
		var remaining []GridRow
		for _, pr := range m.pendingRows {
			if gridLabels[pr.Label] {
				continue // server now has entries for this row
			}
			m.grid.Rows = append(m.grid.Rows, pr)
			remaining = append(remaining, pr)
		}
		m.pendingRows = remaining
		if len(remaining) > 0 {
			sort.Slice(m.grid.Rows, func(i, j int) bool {
				return m.grid.Rows[i].Label < m.grid.Rows[j].Label
			})
		}

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

	case clockToggleMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.attendance = msg.status
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

	case searchDataLoadedMsg:
		if m.state != stateSearch {
			return m, nil
		}
		if msg.err != nil {
			m.searchErr = msg.err
			m.searchSub = searchReady
			return m, nil
		}
		m.searchItems = buildSearchItems(msg.projects, msg.tasks)
		m.searchFiltered = filterSearchItems(m.searchItems, m.searchInput.Value())
		m.searchSub = searchReady
		m.searchCursor = 0
		return m, nil

	case tea.KeyPressMsg:
		// In help state, only Esc/q/? dismiss the overlay.
		if m.state == stateHelp {
			switch {
			case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.Help):
				m.state = m.helpPrevState
			}
			return m, nil
		}
		// In search state, forward keys to search handler.
		if m.state == stateSearch {
			return m.updateSearch(msg)
		}
		// In edit state, forward keys to text inputs.
		if m.state == stateEdit {
			return m.updateEdit(msg)
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.helpPrevState = m.state
			m.state = stateHelp
			return m, nil

		case key.Matches(msg, m.keys.Refresh):
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadTimesheets(), m.loadAttendance())

		case key.Matches(msg, m.keys.ClockToggle):
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.toggleClock())

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
			case key.Matches(msg, m.keys.Search):
				return m.enterSearch()
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
	m.editHours.Placeholder = "1.5 or 1:30"
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
	m.editHours.Placeholder = "1.5 or 1:30"
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
	hours, err := ParseHours(m.editHours.Value())
	if err != nil {
		m.editErr = err
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
// If deleting the last entry for a row, the row is preserved as a pending row.
func (m Model) deleteEntry() (tea.Model, tea.Cmd) {
	entries := m.detailEntries()
	if m.detailCursor >= len(entries) {
		return m, nil
	}

	// Check if this is the last entry across all days for the current row.
	if m.cursor[0] < len(m.grid.Rows) {
		row := m.grid.Rows[m.cursor[0]]
		totalEntries := 0
		for d := 0; d < 7; d++ {
			totalEntries += len(row.Entries[d])
		}
		if totalEntries <= 1 {
			projectID, taskID := row.ProjectTaskIDs()
			m.pendingRows = append(m.pendingRows, GridRow{
				Label:         row.Label,
				Company:       row.Company,
				HintProjectID: projectID,
				HintTaskID:    taskID,
			})
		}
	}

	client := m.client
	id := entries[m.detailCursor].ID
	return m, func() tea.Msg {
		err := client.DeleteTimesheet(id)
		return deleteEntryMsg{err: err}
	}
}

// enterSearch transitions from grid to search state.
func (m Model) enterSearch() (tea.Model, tea.Cmd) {
	m.state = stateSearch
	m.searchSub = searchLoading
	m.searchUseFilter = true
	m.searchCursor = 0
	m.searchItems = nil
	m.searchFiltered = nil
	m.searchErr = nil

	m.searchInput = textinput.New()
	m.searchInput.Placeholder = "Type to search..."
	m.searchInput.SetWidth(40)
	cmd := m.searchInput.Focus()

	return m, tea.Batch(cmd, m.spinner.Tick, m.loadSearchData())
}

// loadSearchData fires parallel API calls to load projects and tasks.
func (m Model) loadSearchData() tea.Cmd {
	client := m.client
	filtered := m.searchUseFilter
	return func() tea.Msg {
		var projects []odoo.ProjectInfo
		var tasks []odoo.TaskInfo
		var pErr, tErr error

		done := make(chan struct{}, 2)
		go func() {
			if filtered {
				projects, pErr = client.ListProjects()
			} else {
				projects, pErr = client.ListAllProjects()
			}
			done <- struct{}{}
		}()
		go func() {
			if filtered {
				tasks, tErr = client.ListTasks(0)
			} else {
				tasks, tErr = client.ListAllTasks(0)
			}
			done <- struct{}{}
		}()
		<-done
		<-done

		if pErr != nil {
			return searchDataLoadedMsg{err: pErr}
		}
		if tErr != nil {
			return searchDataLoadedMsg{err: tErr}
		}
		return searchDataLoadedMsg{projects: projects, tasks: tasks}
	}
}

// buildSearchItems converts projects and tasks into a unified search item list.
func buildSearchItems(projects []odoo.ProjectInfo, tasks []odoo.TaskInfo) []searchItem {
	items := make([]searchItem, 0, len(projects)+len(tasks))
	for _, p := range projects {
		items = append(items, searchItem{
			Kind:      "project",
			ID:        p.ID,
			Name:      p.Name,
			Extra:     p.Company,
			ProjectID: p.ID,
			TaskID:    0,
		})
	}
	for _, t := range tasks {
		items = append(items, searchItem{
			Kind:      "task",
			ID:        t.ID,
			Name:      t.Name,
			Extra:     t.Project,
			ProjectID: t.ProjectID,
			TaskID:    t.ID,
		})
	}
	return items
}

// filterSearchItems returns items matching the query (case-insensitive substring).
func filterSearchItems(items []searchItem, query string) []searchItem {
	if query == "" {
		return items
	}
	q := strings.ToLower(query)
	var result []searchItem
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), q) ||
			strings.Contains(strings.ToLower(item.Extra), q) {
			result = append(result, item)
		}
	}
	return result
}

// updateSearch handles key events in the search state.
func (m Model) updateSearch(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.state = stateGrid
		return m, nil

	case key.Matches(msg, m.keys.SearchToggle):
		m.searchUseFilter = !m.searchUseFilter
		m.searchSub = searchLoading
		m.searchItems = nil
		m.searchFiltered = nil
		m.searchCursor = 0
		return m, tea.Batch(m.spinner.Tick, m.loadSearchData())

	case msg.Code == '\r' || msg.Code == '\n':
		if m.searchSub == searchReady && len(m.searchFiltered) > 0 &&
			m.searchCursor < len(m.searchFiltered) {
			return m.selectSearchItem(m.searchFiltered[m.searchCursor])
		}
		return m, nil
	}

	if m.searchSub == searchReady {
		switch {
		case key.Matches(msg, m.keys.Up):
			if m.searchCursor > 0 {
				m.searchCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.searchCursor < len(m.searchFiltered)-1 {
				m.searchCursor++
			}
			return m, nil
		}
	}

	// Forward to text input.
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)

	// Re-filter on text change.
	if m.searchSub == searchReady {
		m.searchFiltered = filterSearchItems(m.searchItems, m.searchInput.Value())
		if m.searchCursor >= len(m.searchFiltered) {
			m.searchCursor = 0
		}
	}

	return m, cmd
}

// selectSearchItem adds the selected project/task as a grid row or moves cursor to existing.
func (m Model) selectSearchItem(item searchItem) (tea.Model, tea.Cmd) {
	var label string
	if item.Kind == "project" {
		label = item.Name
	} else {
		label = item.Extra + " / " + item.Name
	}

	// Check for duplicate label.
	for i, row := range m.grid.Rows {
		if row.Label == label {
			m.cursor[0] = i
			m.state = stateGrid
			return m, nil
		}
	}

	// Add new row and track as pending so it survives reloads.
	newRow := GridRow{
		Label:         label,
		HintProjectID: item.ProjectID,
		HintTaskID:    item.TaskID,
	}
	m.grid.Rows = append(m.grid.Rows, newRow)
	m.pendingRows = append(m.pendingRows, newRow)

	// Re-sort and find the new row's index.
	sort.Slice(m.grid.Rows, func(i, j int) bool {
		return m.grid.Rows[i].Label < m.grid.Rows[j].Label
	})
	for i, row := range m.grid.Rows {
		if row.Label == label {
			m.cursor[0] = i
			break
		}
	}

	m.state = stateGrid
	return m, nil
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

	case stateGrid, stateDetail, stateEdit, stateSearch, stateHelp:
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
		grid := RenderGrid(m.grid, m.cursor[0], m.cursor[1], m.width-4, m.limits, m.weekHols, m.companyColors)

		helpView := m.help.View(m.keys)
		s = "\n" + title + grid + "\n  " + helpView + "\n"

		if m.state == stateEdit && m.cursor[0] < len(m.grid.Rows) {
			row := m.grid.Rows[m.cursor[0]]
			day := m.monday.AddDate(0, 0, m.cursor[1])
			edit := renderEditForm(row, day, m.editHours, m.editDesc, m.editFocus, m.editErr, m.width, m.editIsNew)
			s = RenderDetailOverlay(s, edit, m.width, m.height)
		} else if m.state == stateDetail && m.cursor[0] < len(m.grid.Rows) {
			detail := RenderDetail(m.grid.Rows[m.cursor[0]], m.cursor[1], m.monday.Time, m.detailCursor, m.width, m.companyColors)
			s = RenderDetailOverlay(s, detail, m.width, m.height)
		} else if m.state == stateSearch {
			search := renderSearchOverlay(m.searchInput, m.searchFiltered, m.searchCursor, m.searchSub, m.searchUseFilter, m.searchErr, m.spinner, m.width, m.height, m.companyColors)
			s = RenderDetailOverlay(s, search, m.width, m.height)
		} else if m.state == stateHelp {
			helpContent := renderHelpOverlay(m.keys, m.width, m.height)
			s = RenderDetailOverlay(s, helpContent, m.width, m.height)
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
