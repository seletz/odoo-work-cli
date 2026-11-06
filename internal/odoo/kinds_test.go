package odoo

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/seletz/odoo-work-cli/internal/apperr"
)

// timesheetRecordResponse builds a single-record account.analytic.line
// search_read response.
func timesheetRecordResponse(id int64) string {
	return fmt.Sprintf(`<?xml version="1.0"?>
<methodResponse><params><param><value><array><data>
<value><struct>
<member><name>id</name><value><int>%d</int></value></member>
<member><name>date</name><value><string>2026-07-16</string></value></member>
<member><name>project_id</name><value><array><data>
<value><int>3</int></value><value><string>Internal</string></value>
</data></array></value></member>
<member><name>task_id</name><value><boolean>0</boolean></value></member>
<member><name>name</name><value><string>Review</string></value></member>
<member><name>unit_amount</name><value><double>1.5</double></value></member>
<member><name>employee_id</name><value><array><data>
<value><int>7</int></value><value><string>Test User</string></value>
</data></array></value></member>
<member><name>validated_status</name><value><string>draft</string></value></member>
<member><name>company_id</name><value><array><data>
<value><int>1</int></value><value><string>ACME</string></value>
</data></array></value></member>
</struct></value>
</data></array></value></param></params></methodResponse>`, id)
}

// objectServer answers every /xmlrpc/2/object call with the given body.
func objectServer(t *testing.T, objectResp string) *XMLRPCClient {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/xml")
		if r.URL.Path == "/xmlrpc/2/common" {
			_, _ = fmt.Fprint(w, authenticateResponse)
			return
		}
		_, _ = fmt.Fprint(w, objectResp)
	}))
	t.Cleanup(server.Close)
	client, err := NewXMLRPCClient(server.URL, "testdb", "test@example.com", "api-key", "", "", nil)
	if err != nil {
		t.Fatalf("creating client: %v", err)
	}
	t.Cleanup(client.Close)
	return client
}

func TestGetTimesheet(t *testing.T) {
	client := objectServer(t, timesheetRecordResponse(55))
	got, err := client.GetTimesheet(55)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 55 || got.ProjectID != 3 || got.Project != "Internal" || got.TaskID != 0 ||
		got.Hours != 1.5 || got.ValidatedStatus != "draft" || got.Company != "ACME" || got.Name != "Review" {
		t.Errorf("unexpected entry: %+v", got)
	}
}

func TestGetAttendance(t *testing.T) {
	client := objectServer(t, attRecordResponse(17, "2026-07-16 06:00:00", "2026-07-16 10:00:00", 4))
	got, err := client.GetAttendance(17)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 17 || got.WorkedHours != 4 || got.CheckOut == nil {
		t.Errorf("unexpected record: %+v", got)
	}
}

// TestErrorKinds pins the kind of every error the odoo package creates at
// the origin. Kinds are a public contract (see apperr).
func TestErrorKinds(t *testing.T) {
	restore := apperr.RegisterClassifier(ClassifyError)
	defer restore()

	day := time.Date(2026, 7, 16, 8, 0, 0, 0, testZone)
	later := day.Add(time.Hour)
	open := attRecordResponse(50, "2026-07-16 05:00:00", "", 0)

	tests := []struct {
		name string
		run  func(t *testing.T) error
		want apperr.Kind
	}{
		{
			name: "timesheet params invalid",
			run:  func(t *testing.T) error { return ValidateTimesheetParams(TimesheetWriteParams{}) },
			want: apperr.KindValidation,
		},
		{
			name: "get timesheet not found",
			run: func(t *testing.T) error {
				_, err := objectServer(t, emptyArrayResponse).GetTimesheet(9)
				return err
			},
			want: apperr.KindNotFound,
		},
		{
			name: "get attendance not found",
			run: func(t *testing.T) error {
				_, err := objectServer(t, emptyArrayResponse).GetAttendance(9)
				return err
			},
			want: apperr.KindNotFound,
		},
		{
			name: "update timesheet access error",
			run: func(t *testing.T) error {
				return objectServer(t, aclFaultResponse).UpdateTimesheet(5, map[string]interface{}{"name": "x"})
			},
			want: apperr.KindAccessDenied,
		},
		{
			name: "clock in at while clocked in",
			run: func(t *testing.T) error {
				m := &attendanceWriteMock{t: t, attSearch: []string{emptyArrayResponse, open}}
				_, err := m.client(t).ClockInAt(day)
				return err
			},
			want: apperr.KindConflict,
		},
		{
			name: "clock out at while not clocked in",
			run: func(t *testing.T) error {
				m := &attendanceWriteMock{t: t, attSearch: []string{emptyArrayResponse, emptyArrayResponse}}
				_, err := m.client(t).ClockOutAt(day)
				return err
			},
			want: apperr.KindConflict,
		},
		{
			name: "create attendance out before in",
			run: func(t *testing.T) error {
				m := &attendanceWriteMock{t: t}
				_, err := m.client(t).CreateAttendance(later, day)
				return err
			},
			want: apperr.KindValidation,
		},
		{
			name: "edit attendance nothing to change",
			run: func(t *testing.T) error {
				m := &attendanceWriteMock{t: t}
				_, err := m.client(t).EditAttendance(5, nil, nil)
				return err
			},
			want: apperr.KindValidation,
		},
		{
			name: "clock in without web password",
			run: func(t *testing.T) error {
				m := &attendanceWriteMock{t: t, attSearch: []string{emptyArrayResponse, emptyArrayResponse}}
				_, err := m.client(t).ClockIn()
				return err
			},
			want: apperr.KindConfig,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(t)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got := apperr.KindOf(err); got != tt.want {
				t.Errorf("KindOf(%q) = %q, want %q", err, got, tt.want)
			}
		})
	}
}

func TestErrorKinds_AccessHintPreserved(t *testing.T) {
	m := &attendanceWriteMock{t: t, attSearch: []string{emptyArrayResponse, emptyArrayResponse}, createResp: aclFaultResponse}
	_, err := m.client(t).ClockInAt(time.Date(2026, 7, 16, 8, 0, 0, 0, testZone))
	if err == nil || !strings.Contains(err.Error(), AttendanceAccessHint) {
		t.Fatalf("expected access hint, got %v", err)
	}
}
