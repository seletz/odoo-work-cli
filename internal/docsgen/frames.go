package docsgen

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo/fake"
	"github.com/seletz/odoo-work-cli/internal/tui"
)

// Size of the screenshots in terminal cells.
const (
	Width  = 120
	Height = 28
)

// Shot describes one TUI screenshot: the keys pressed after the first load
// and the caption used in the docs. Every shot starts in the grid of the
// demo week at DemoNow.
type Shot struct {
	Name          string
	Width, Height int
	// Keys are pressed in order: a single character or one of enter, tab,
	// shift+tab, esc, up, down, left, right, backspace.
	Keys    []string
	Caption string
}

// Shots are the screenshots of the docs.
var Shots = []Shot{
	{Name: "grid", Caption: "Weekly grid: one row per project/task, today's column selected, attendance per day below the totals"},
	{Name: "prev-week", Keys: []string{"h"}, Caption: "Previous week (h): Good Friday is a public holiday, all entries validated"},
	{Name: "detail", Keys: []string{"enter"}, Caption: "Detail view of the selected cell (enter): every entry of that day with description and hours"},
	{Name: "edit", Keys: []string{"enter", "e"}, Caption: "Edit form (e in the detail view): hours accept 1:30 or 1.5"},
	{Name: "add", Keys: []string{"enter", "a"}, Caption: "Add form (a in the detail view): a new entry for the selected project/task and day"},
	{Name: "search", Keys: []string{"/", "i", "n"}, Caption: "Project/task search (/): type to filter, tab to the list, enter adds a row"},
	{Name: "attendance", Keys: []string{"t"}, Caption: "Attendance form (t): today's running record, check-out still empty"},
	{Name: "help", Height: 46, Keys: []string{"?"}, Caption: "Help overlay (?): every key binding by context"},
}

func init() {
	for i := range Shots {
		if Shots[i].Width == 0 {
			Shots[i].Width = Width
		}
		if Shots[i].Height == 0 {
			Shots[i].Height = Height
		}
	}
}

// Frame is the rendered final frame of a Shot, with ANSI styling.
type Frame struct {
	Shot    Shot
	Content string
}

// Plain returns the frame without escape sequences and trailing blanks.
//
// :returns: the plain text frame
func (f Frame) Plain() string {
	lines := strings.Split(ansi.Strip(f.Content), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

// Render drives the TUI headlessly against a copy of the demo state at
// DemoNow and returns the frame after the shot's keys. Commands run
// synchronously, so every load has landed when the frame is taken.
//
// :param s: the shot
// :returns: the frame
func Render(s Shot) (Frame, error) {
	dir, err := os.MkdirTemp("", "odoo-work-cli-shot-")
	if err != nil {
		return Frame{}, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	path := filepath.Join(dir, "state.json")
	if err := WriteDemoState(path); err != nil {
		return Frame{}, err
	}
	now := func() time.Time { return DemoNow }
	client, err := fake.Open(path, now)
	if err != nil {
		return Frame{}, err
	}
	model := tui.NewModel(client, tui.MondayTime{Time: DemoMonday()}, config.DefaultHoursLimits(),
		config.DefaultBundesland, nil, DemoCompanyColors).
		WithNow(now).
		WithTick(func(time.Duration, func(time.Time) tea.Msg) tea.Cmd { return nil })

	r := &runner{}
	var m tea.Model = model
	m = r.send(m, tea.WindowSizeMsg{Width: s.Width, Height: s.Height})
	m = r.run(m, m.Init())
	for _, k := range s.Keys {
		press, err := keyPress(k)
		if err != nil {
			return Frame{}, fmt.Errorf("shot %s: %w", s.Name, err)
		}
		m = r.send(m, press)
	}
	if r.err != nil {
		return Frame{}, fmt.Errorf("shot %s: %w", s.Name, r.err)
	}
	return Frame{Shot: s, Content: m.View().Content}, nil
}

// RenderAll renders every one of Shots.
//
// :returns: the frames in the order of Shots
func RenderAll() ([]Frame, error) {
	out := make([]Frame, 0, len(Shots))
	for _, s := range Shots {
		f, err := Render(s)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

// keyPress builds the key press for a name.
func keyPress(name string) (tea.KeyPressMsg, error) {
	special := map[string]tea.KeyPressMsg{
		"enter":     {Code: tea.KeyEnter},
		"tab":       {Code: tea.KeyTab},
		"shift+tab": {Code: tea.KeyTab, Mod: tea.ModShift},
		"esc":       {Code: tea.KeyEscape},
		"up":        {Code: tea.KeyUp},
		"down":      {Code: tea.KeyDown},
		"left":      {Code: tea.KeyLeft},
		"right":     {Code: tea.KeyRight},
		"backspace": {Code: tea.KeyBackspace},
	}
	if k, ok := special[name]; ok {
		return k, nil
	}
	r := []rune(name)
	if len(r) != 1 {
		return tea.KeyPressMsg{}, fmt.Errorf("unknown key %q", name)
	}
	return tea.KeyPressMsg{Code: r[0], Text: name}, nil
}

// maxMessages bounds one synchronous run; more means a message loop.
const maxMessages = 10000

// cmdTimeout drops a command that does not return in time: a timer that
// would only animate something (the cursor blinks after 530 ms). The
// commands that load data run against the in-memory fake and return in
// microseconds.
const cmdTimeout = 300 * time.Millisecond

// runner executes bubbletea commands synchronously.
type runner struct {
	msgs int
	err  error
}

// send hands msg to m and runs the returned command to completion.
func (r *runner) send(m tea.Model, msg tea.Msg) tea.Model {
	m, cmd := m.Update(msg)
	return r.run(m, cmd)
}

// run executes cmd against m: batches and sequences are unpacked in order,
// animation messages (spinner ticks, cursor blinks) are dropped, every
// other message goes to m.Update and the command it returns runs in turn.
func (r *runner) run(m tea.Model, cmd tea.Cmd) tea.Model {
	r.exec(func(msg tea.Msg) tea.Cmd {
		var next tea.Cmd
		m, next = m.Update(msg)
		return next
	}, cmd)
	return m
}

func (r *runner) exec(update func(tea.Msg) tea.Cmd, cmd tea.Cmd) {
	if cmd == nil || r.err != nil {
		return
	}
	msg, ok := call(cmd)
	if !ok {
		return
	}
	switch v := msg.(type) {
	case nil, tea.QuitMsg, spinner.TickMsg:
		return
	case tea.BatchMsg:
		for _, c := range v {
			r.exec(update, c)
		}
		return
	}
	rv := reflect.ValueOf(msg)
	if rv.Type().PkgPath() == "charm.land/bubbles/v2/cursor" {
		return // cursor blink
	}
	// tea.Sequence yields an unexported slice of commands.
	if rv.Kind() == reflect.Slice && rv.Type().Elem() == reflect.TypeFor[tea.Cmd]() {
		for i := range rv.Len() {
			if c, ok := rv.Index(i).Interface().(tea.Cmd); ok {
				r.exec(update, c)
			}
		}
		return
	}
	if r.msgs++; r.msgs > maxMessages {
		r.err = fmt.Errorf("more than %d messages: a message loop", maxMessages)
		return
	}
	r.exec(update, update(msg))
}

// call runs cmd and reports whether it returned within cmdTimeout.
func call(cmd tea.Cmd) (tea.Msg, bool) {
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		return msg, true
	case <-time.After(cmdTimeout):
		return nil, false
	}
}
