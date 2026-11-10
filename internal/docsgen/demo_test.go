package docsgen

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/seletz/odoo-work-cli/internal/odoo/fake"
)

func TestDemoState_IsDeterministic(t *testing.T) {
	a, err := json.Marshal(DemoState())
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(DemoState())
	if string(a) != string(b) {
		t.Fatal("DemoState differs between calls")
	}
}

func TestDemoState_MadeUpDataOnly(t *testing.T) {
	data, _ := json.Marshal(DemoState())
	// Every e-mail address must use the reserved .invalid TLD.
	for _, m := range regexp.MustCompile(`[\w.+-]+@[\w.-]+`).FindAllString(string(data), -1) {
		if !strings.HasSuffix(m, ".invalid") {
			t.Errorf("e-mail %q is not on a reserved .invalid domain", m)
		}
	}
	if strings.Contains(string(data), "http") {
		t.Error("demo state must not contain URLs")
	}
}

func TestDemoState_WeekOfDemoNow(t *testing.T) {
	st := DemoState()
	monday := DemoMonday()
	if monday.Weekday() != time.Monday || DemoNow.Sub(monday) < 0 || DemoNow.Sub(monday) > 7*24*time.Hour {
		t.Fatalf("DemoMonday %v is not the Monday of DemoNow %v", monday, DemoNow)
	}
	var thisWeek, open int
	for _, e := range st.Timesheets {
		if e.Date >= monday.Format("2006-01-02") {
			thisWeek++
		}
	}
	for _, r := range st.Attendances {
		if r.CheckOut == nil {
			open++
			if r.CheckIn.After(DemoNow) {
				t.Errorf("open attendance %v starts after DemoNow", r.CheckIn)
			}
		}
	}
	if thisWeek == 0 {
		t.Error("no timesheet entries in the demo week")
	}
	if open != 1 {
		t.Errorf("want exactly one running attendance (clocked in), got %d", open)
	}
}

func TestWriteDemoState_OpensWithFakeBackend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := WriteDemoState(path); err != nil {
		t.Fatal(err)
	}
	c, err := fake.Open(path, func() time.Time { return DemoNow })
	if err != nil {
		t.Fatal(err)
	}
	st, err := c.AttendanceStatus()
	if err != nil {
		t.Fatal(err)
	}
	if !st.ClockedIn {
		t.Error("demo user should be clocked in at DemoNow")
	}
	u, err := c.WhoAmI()
	if err != nil || u.Name != DemoUser {
		t.Errorf("WhoAmI = %v, %v; want %s", u, err, DemoUser)
	}
}
