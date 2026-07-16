package odoo

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	goOdoo "github.com/skilld-labs/go-odoo"
)

const emptyArrayResponse = `<?xml version="1.0"?>
<methodResponse><params><param><value><array><data></data></array></value></param></params></methodResponse>`

const writeOKResponse = `<?xml version="1.0"?>
<methodResponse><params><param><value><boolean>1</boolean></value></param></params></methodResponse>`

// employeeResponse answers the findEmployeeID search_read.
const employeeResponse = `<?xml version="1.0"?>
<methodResponse><params><param><value><array><data>
<value><struct>
<member><name>id</name><value><int>7</int></value></member>
</struct></value>
</data></array></value></param></params></methodResponse>`

func intResponse(id int64) string {
	return fmt.Sprintf(`<?xml version="1.0"?>
<methodResponse><params><param><value><int>%d</int></value></param></params></methodResponse>`, id)
}

// attRecordResponse builds a single-record hr.attendance search_read response.
// checkOut == "" renders as false (open record).
func attRecordResponse(id int64, checkIn, checkOut string, workedHours float64) string {
	out := "<boolean>0</boolean>"
	if checkOut != "" {
		out = fmt.Sprintf("<string>%s</string>", checkOut)
	}
	return fmt.Sprintf(`<?xml version="1.0"?>
<methodResponse><params><param><value><array><data>
<value><struct>
<member><name>id</name><value><int>%d</int></value></member>
<member><name>employee_id</name><value><array><data>
<value><int>7</int></value><value><string>Test User</string></value>
</data></array></value></member>
<member><name>check_in</name><value><string>%s</string></value></member>
<member><name>check_out</name><value>%s</value></member>
<member><name>worked_hours</name><value><double>%f</double></value></member>
</struct></value>
</data></array></value></param></params></methodResponse>`, id, checkIn, out, workedHours)
}

// attendanceWriteMock simulates the Odoo XML-RPC endpoint for attendance
// write flows. hr.attendance search_read responses are consumed from
// attSearch in request order; create/write bodies are captured.
type attendanceWriteMock struct {
	t          *testing.T
	attSearch  []string
	createResp string
	writeResp  string

	createBody string
	writeBody  string
}

func (m *attendanceWriteMock) server() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			m.t.Errorf("reading request body: %v", err)
		}
		body := string(raw)
		w.Header().Set("Content-Type", "text/xml")

		switch r.URL.Path {
		case "/xmlrpc/2/common":
			_, _ = fmt.Fprint(w, authenticateResponse)
		case "/xmlrpc/2/object":
			switch {
			case strings.Contains(body, "<string>hr.employee</string>"):
				_, _ = fmt.Fprint(w, employeeResponse)
			case strings.Contains(body, "<string>hr.attendance</string>"):
				switch {
				case strings.Contains(body, "<string>create</string>"):
					m.createBody = body
					if m.createResp == "" {
						m.t.Error("unexpected hr.attendance create call")
						_, _ = fmt.Fprint(w, aclFaultResponse)
						return
					}
					_, _ = fmt.Fprint(w, m.createResp)
				case strings.Contains(body, "<string>write</string>"):
					m.writeBody = body
					if m.writeResp == "" {
						m.t.Error("unexpected hr.attendance write call")
						_, _ = fmt.Fprint(w, aclFaultResponse)
						return
					}
					_, _ = fmt.Fprint(w, m.writeResp)
				case strings.Contains(body, "<string>search_read</string>"):
					if len(m.attSearch) == 0 {
						m.t.Error("unexpected hr.attendance search_read call")
						_, _ = fmt.Fprint(w, emptyArrayResponse)
						return
					}
					resp := m.attSearch[0]
					m.attSearch = m.attSearch[1:]
					_, _ = fmt.Fprint(w, resp)
				default:
					m.t.Errorf("unexpected hr.attendance method in body: %s", body)
					http.NotFound(w, r)
				}
			default:
				m.t.Errorf("unexpected model in body: %s", body)
				http.NotFound(w, r)
			}
		default:
			m.t.Errorf("unexpected request path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
}

func (m *attendanceWriteMock) client(t *testing.T) *XMLRPCClient {
	t.Helper()
	server := m.server()
	t.Cleanup(server.Close)
	client, err := NewXMLRPCClient(server.URL, "testdb", "test@example.com", "api-key", "", "", nil)
	if err != nil {
		t.Fatalf("creating client: %v", err)
	}
	t.Cleanup(client.Close)
	return client
}

var testZone = time.FixedZone("CEST", 2*3600)

func TestClockInAt_CreatesBackdatedRecord(t *testing.T) {
	mock := &attendanceWriteMock{
		t: t,
		// status queries: today range + open records, both empty
		attSearch:  []string{emptyArrayResponse, emptyArrayResponse},
		createResp: intResponse(43),
	}
	client := mock.client(t)

	// 08:30 local CEST == 06:30 UTC
	id, err := client.ClockInAt(time.Date(2026, 7, 16, 8, 30, 0, 0, testZone))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 43 {
		t.Errorf("ID = %d, want 43", id)
	}
	if !strings.Contains(mock.createBody, "<string>2026-07-16 06:30:00</string>") {
		t.Errorf("create body missing UTC check_in, got: %s", mock.createBody)
	}
	if !strings.Contains(mock.createBody, "<name>employee_id</name>") {
		t.Errorf("create body missing employee_id, got: %s", mock.createBody)
	}
	if strings.Contains(mock.createBody, "<name>check_out</name>") {
		t.Errorf("create body must not set check_out, got: %s", mock.createBody)
	}
}

func TestClockInAt_AlreadyClockedIn(t *testing.T) {
	open := attRecordResponse(50, "2026-07-16 05:00:00", "", 0)
	mock := &attendanceWriteMock{
		t:         t,
		attSearch: []string{emptyArrayResponse, open},
	}
	client := mock.client(t)

	_, err := client.ClockInAt(time.Date(2026, 7, 16, 8, 30, 0, 0, testZone))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "already clocked in" {
		t.Errorf("error = %q, want %q", err.Error(), "already clocked in")
	}
}

func TestClockInAt_AccessDenied(t *testing.T) {
	mock := &attendanceWriteMock{
		t:          t,
		attSearch:  []string{emptyArrayResponse, emptyArrayResponse},
		createResp: aclFaultResponse,
	}
	client := mock.client(t)

	_, err := client.ClockInAt(time.Date(2026, 7, 16, 8, 30, 0, 0, testZone))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Attendance Officer") {
		t.Errorf("error should hint at officer rights, got: %v", err)
	}
}

func TestClockOutAt_ClosesOpenRecord(t *testing.T) {
	open := attRecordResponse(50, "2026-07-16 06:00:00", "", 0)
	closed := attRecordResponse(50, "2026-07-16 06:00:00", "2026-07-16 10:30:00", 4.5)
	mock := &attendanceWriteMock{
		t:         t,
		attSearch: []string{emptyArrayResponse, open, closed},
		writeResp: writeOKResponse,
	}
	client := mock.client(t)

	// 12:30 local CEST == 10:30 UTC
	rec, err := client.ClockOutAt(time.Date(2026, 7, 16, 12, 30, 0, 0, testZone))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.ID != 50 {
		t.Errorf("ID = %d, want 50", rec.ID)
	}
	if rec.WorkedHours != 4.5 {
		t.Errorf("WorkedHours = %f, want 4.5", rec.WorkedHours)
	}
	if !strings.Contains(mock.writeBody, "<string>2026-07-16 10:30:00</string>") {
		t.Errorf("write body missing UTC check_out, got: %s", mock.writeBody)
	}
}

func TestClockOutAt_NotClockedIn(t *testing.T) {
	mock := &attendanceWriteMock{
		t:         t,
		attSearch: []string{emptyArrayResponse, emptyArrayResponse},
	}
	client := mock.client(t)

	_, err := client.ClockOutAt(time.Date(2026, 7, 16, 12, 30, 0, 0, testZone))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "not clocked in" {
		t.Errorf("error = %q, want %q", err.Error(), "not clocked in")
	}
}

func TestClockOutAt_BeforeCheckIn(t *testing.T) {
	// open record check_in 09:00 UTC == 11:00 CEST; clock out at 10:00 CEST
	open := attRecordResponse(50, "2026-07-16 09:00:00", "", 0)
	mock := &attendanceWriteMock{
		t:         t,
		attSearch: []string{emptyArrayResponse, open},
	}
	client := mock.client(t)

	_, err := client.ClockOutAt(time.Date(2026, 7, 16, 10, 0, 0, 0, testZone))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not after check-in") {
		t.Errorf("error = %q, want it to mention 'not after check-in'", err.Error())
	}
}

func TestCreateAttendance_SetsBothTimes(t *testing.T) {
	mock := &attendanceWriteMock{
		t:          t,
		createResp: intResponse(44),
	}
	client := mock.client(t)

	id, err := client.CreateAttendance(
		time.Date(2026, 7, 15, 8, 0, 0, 0, testZone),
		time.Date(2026, 7, 15, 16, 0, 0, 0, testZone))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 44 {
		t.Errorf("ID = %d, want 44", id)
	}
	if !strings.Contains(mock.createBody, "<string>2026-07-15 06:00:00</string>") {
		t.Errorf("create body missing UTC check_in, got: %s", mock.createBody)
	}
	if !strings.Contains(mock.createBody, "<string>2026-07-15 14:00:00</string>") {
		t.Errorf("create body missing UTC check_out, got: %s", mock.createBody)
	}
}

func TestCreateAttendance_CheckOutNotAfterCheckIn(t *testing.T) {
	mock := &attendanceWriteMock{t: t}
	client := mock.client(t)

	_, err := client.CreateAttendance(
		time.Date(2026, 7, 15, 16, 0, 0, 0, testZone),
		time.Date(2026, 7, 15, 8, 0, 0, 0, testZone))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not after check-in") {
		t.Errorf("error = %q, want it to mention 'not after check-in'", err.Error())
	}
}

func TestEditAttendance_UpdatesCheckInOnly(t *testing.T) {
	updated := attRecordResponse(50, "2026-07-16 06:00:00", "2026-07-16 14:00:00", 8.0)
	mock := &attendanceWriteMock{
		t:         t,
		attSearch: []string{updated},
		writeResp: writeOKResponse,
	}
	client := mock.client(t)

	checkIn := time.Date(2026, 7, 16, 8, 0, 0, 0, testZone) // 06:00 UTC
	rec, err := client.EditAttendance(50, &checkIn, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.ID != 50 {
		t.Errorf("ID = %d, want 50", rec.ID)
	}
	if rec.WorkedHours != 8.0 {
		t.Errorf("WorkedHours = %f, want 8.0", rec.WorkedHours)
	}
	if !strings.Contains(mock.writeBody, "<name>check_in</name>") {
		t.Errorf("write body missing check_in, got: %s", mock.writeBody)
	}
	if !strings.Contains(mock.writeBody, "<string>2026-07-16 06:00:00</string>") {
		t.Errorf("write body missing UTC check_in value, got: %s", mock.writeBody)
	}
	if strings.Contains(mock.writeBody, "<name>check_out</name>") {
		t.Errorf("write body must not set check_out, got: %s", mock.writeBody)
	}
}

func TestEditAttendance_UpdatesBothTimes(t *testing.T) {
	updated := attRecordResponse(50, "2026-07-16 06:00:00", "2026-07-16 14:00:00", 8.0)
	mock := &attendanceWriteMock{
		t:         t,
		attSearch: []string{updated},
		writeResp: writeOKResponse,
	}
	client := mock.client(t)

	checkIn := time.Date(2026, 7, 16, 8, 0, 0, 0, testZone)
	checkOut := time.Date(2026, 7, 16, 16, 0, 0, 0, testZone)
	rec, err := client.EditAttendance(50, &checkIn, &checkOut)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.CheckOut == nil {
		t.Fatal("expected CheckOut to be set")
	}
	if !strings.Contains(mock.writeBody, "<name>check_out</name>") {
		t.Errorf("write body missing check_out, got: %s", mock.writeBody)
	}
}

func TestEditAttendance_Validation(t *testing.T) {
	checkIn := time.Date(2026, 7, 16, 16, 0, 0, 0, testZone)
	checkOut := time.Date(2026, 7, 16, 8, 0, 0, 0, testZone)

	tests := []struct {
		name     string
		id       int64
		checkIn  *time.Time
		checkOut *time.Time
		wantMsg  string
	}{
		{
			name:    "no fields to update",
			id:      50,
			wantMsg: "nothing to update: provide a new check-in and/or check-out time",
		},
		{
			name:    "invalid id",
			id:      0,
			checkIn: &checkIn,
			wantMsg: "attendance ID is required",
		},
		{
			name:     "check-out not after check-in",
			id:       50,
			checkIn:  &checkIn,
			checkOut: &checkOut,
			wantMsg:  "check-out 08:00 is not after check-in 16:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &attendanceWriteMock{t: t}
			client := mock.client(t)

			_, err := client.EditAttendance(tt.id, tt.checkIn, tt.checkOut)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tt.wantMsg {
				t.Errorf("error = %q, want %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestEditAttendance_AccessDenied(t *testing.T) {
	mock := &attendanceWriteMock{
		t:         t,
		writeResp: aclFaultResponse,
	}
	client := mock.client(t)

	checkIn := time.Date(2026, 7, 16, 8, 0, 0, 0, testZone)
	_, err := client.EditAttendance(50, &checkIn, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Attendance Officer") {
		t.Errorf("error should hint at officer rights, got: %v", err)
	}
}

func TestFetchAttendanceRange_ConvertsBoundsToUTC(t *testing.T) {
	var captured []string
	searchFn := func(_ string, criteria *goOdoo.Criteria, _ *goOdoo.Options) ([]map[string]interface{}, error) {
		captured = append(captured, fmt.Sprintf("%v", *criteria))
		return nil, nil
	}

	// local midnight CEST == 22:00 UTC the previous day
	from := time.Date(2026, 7, 16, 0, 0, 0, 0, testZone)
	to := from.AddDate(0, 0, 1)
	if _, err := fetchAttendanceRange(searchFn, 7, from, to); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	all := strings.Join(captured, " | ")
	if !strings.Contains(all, "2026-07-15 22:00:00") {
		t.Errorf("criteria should contain UTC-converted from bound, got: %s", all)
	}
	if strings.Contains(all, "2026-07-16 00:00:00") {
		t.Errorf("criteria must not contain local wall-clock bound, got: %s", all)
	}
}
