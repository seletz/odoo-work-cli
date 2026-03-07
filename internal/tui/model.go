package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

type uiState int

const (
	stateLoading uiState = iota
	stateGrid
	stateError
)

// timesheetsLoadedMsg is sent when timesheets finish loading.
type timesheetsLoadedMsg struct {
	entries []odoo.TimesheetEntry
	err     error
}

// Model is the bubbletea model for the weekly timesheet TUI.
type Model struct {
	state   uiState
	client  odoo.Client
	grid    WeekGrid
	monday  MondayTime
	cursor  [2]int // [row, col]
	spinner spinner.Model
	help    help.Model
	keys    KeyMap
	limits     config.HoursLimits
	bundesland string
	holidays   HolidayMap
	weekHols   [7]string
	err        error
	width      int
	height     int
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
		state:   stateLoading,
		client:  client,
		monday:     monday,
		limits:     limits,
		bundesland: bundesland,
		holidays:   BuildHolidayMap(monday.Year(), bundesland),
		spinner:    s,
		help:    help.New(),
		keys:    DefaultKeyMap(),
	}
}

// Init starts the spinner and triggers the initial data load.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadTimesheets())
}

func (m Model) loadTimesheets() tea.Cmd {
	monday := m.monday.Time
	sunday := monday.AddDate(0, 0, 6)
	dateFrom := monday.Format("2006-01-02")
	dateTo := sunday.Format("2006-01-02")
	client := m.client
	return func() tea.Msg {
		entries, err := client.ListTimesheets(dateFrom, dateTo)
		return timesheetsLoadedMsg{entries: entries, err: err}
	}
}

// Update handles messages and updates the model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case timesheetsLoadedMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.state = stateGrid
		m.grid = BuildWeekGrid(msg.entries, m.monday.Time)
		m.holidays = BuildHolidayMap(m.monday.Year(), m.bundesland)
		m.weekHols = WeekHolidays(m.monday.Time, m.holidays)
		m.cursor = [2]int{0, 0}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.SetWidth(msg.Width)
		return m, nil

	case spinner.TickMsg:
		if m.state == stateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil

		case key.Matches(msg, m.keys.Refresh):
			m.state = stateLoading
			return m, tea.Batch(m.spinner.Tick, m.loadTimesheets())

		case key.Matches(msg, m.keys.Left):
			m.monday = MondayTime{m.monday.AddDate(0, 0, -7)}
			m.state = stateLoading
			return m, tea.Batch(m.spinner.Tick, m.loadTimesheets())

		case key.Matches(msg, m.keys.Right):
			m.monday = MondayTime{m.monday.AddDate(0, 0, 7)}
			m.state = stateLoading
			return m, tea.Batch(m.spinner.Tick, m.loadTimesheets())
		}

		if m.state == stateGrid {
			switch {
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
		return m, nil
	}

	return m, nil
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

	case stateGrid:
		sunday := m.monday.AddDate(0, 0, 6)
		title := fmt.Sprintf("  Week: %s — %s\n\n",
			m.monday.Format("Mon 02 Jan 2006"),
			sunday.Format("Mon 02 Jan 2006"))
		grid := RenderGrid(m.grid, m.cursor[0], m.cursor[1], m.width-4, m.limits, m.weekHols)
		helpView := m.help.View(m.keys)
		s = "\n" + title + grid + "\n  " + helpView + "\n"
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
