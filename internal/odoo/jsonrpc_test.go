package odoo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/pquerna/otp/totp"
)

const (
	testTOTPSecret = "JBSWY3DPEHPK3PXP"
	mockLoginCSRF  = "mock-login-csrf"
	mockTOTPCSRF   = "mock-totp-csrf"
)

// mockOdooServer simulates the Odoo web login endpoints, including the
// multi-db behaviour (#54): with no dbfilter, routes like /web/login/totp
// are dispatched in nodb mode and return 404 until the session is bound
// to a database via GET /web/login?db=<db>. The legacy JSON endpoint
// /web/session/authenticate does NOT bind the session when 2FA is
// pending, which is exactly the bug the HTML login flow works around.
type mockOdooServer struct {
	*httptest.Server

	multiDB    bool
	totpSecret string // non-empty means 2FA is enabled for the user
	db         string
	login      string
	password   string
	callError  bool // make the attendance endpoint return a JSON-RPC error

	mu            sync.Mutex
	dbBound       bool
	totpPending   bool
	authenticated bool
	loginForm     url.Values // last POST /web/login form values
}

func newMockOdooServer(t *testing.T, m *mockOdooServer) *mockOdooServer {
	t.Helper()
	m.Server = httptest.NewServer(http.HandlerFunc(m.handler))
	t.Cleanup(m.Close)
	return m
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (m *mockOdooServer) handler(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch {
	case r.URL.Path == "/web/session/authenticate":
		// Legacy JSON auth. A single-db server binds the session as a
		// side effect; a multi-db server with 2FA pending does not.
		if !m.multiDB {
			m.dbBound = true
		}
		if m.totpSecret != "" {
			m.totpPending = true
			writeJSON(w, map[string]interface{}{
				"jsonrpc": "2.0", "id": 1,
				"result": map[string]interface{}{"uid": false},
			})
			return
		}
		m.authenticated = true
		writeJSON(w, map[string]interface{}{
			"jsonrpc": "2.0", "id": 1,
			"result": map[string]interface{}{"uid": 42},
		})

	case r.URL.Path == "/web/login" && r.Method == http.MethodGet:
		if r.URL.Query().Get("db") == m.db || !m.multiDB {
			m.dbBound = true
		}
		if !m.dbBound {
			http.Redirect(w, r, "/web/database/selector", http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, `<form><input type="hidden" name="csrf_token" value="%s"/></form>`, mockLoginCSRF)

	case r.URL.Path == "/web/login" && r.Method == http.MethodPost:
		if !m.dbBound {
			http.NotFound(w, r)
			return
		}
		_ = r.ParseForm()
		m.loginForm = r.PostForm
		if r.PostFormValue("csrf_token") != mockLoginCSRF {
			http.Error(w, "invalid CSRF token", http.StatusBadRequest)
			return
		}
		if r.PostFormValue("login") != m.login || r.PostFormValue("password") != m.password {
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprintf(w, `<form><input type="hidden" name="csrf_token" value="%s"/>Wrong login/password</form>`, mockLoginCSRF)
			return
		}
		if m.totpSecret != "" {
			m.totpPending = true
			w.Header().Set("Location", "/web/login/totp")
			w.WriteHeader(http.StatusSeeOther)
			return
		}
		m.authenticated = true
		w.Header().Set("Location", "/web")
		w.WriteHeader(http.StatusSeeOther)

	case r.URL.Path == "/web/login/totp" && r.Method == http.MethodGet:
		if !m.dbBound {
			http.NotFound(w, r) // nodb dispatch: the #54 symptom
			return
		}
		if !m.totpPending {
			http.Redirect(w, r, "/web/login", http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, `<form><input type="hidden" name="csrf_token" value="%s"/></form>`, mockTOTPCSRF)

	case r.URL.Path == "/web/login/totp" && r.Method == http.MethodPost:
		if !m.dbBound {
			http.NotFound(w, r)
			return
		}
		_ = r.ParseForm()
		if r.PostFormValue("csrf_token") != mockTOTPCSRF {
			http.Error(w, "invalid CSRF token", http.StatusBadRequest)
			return
		}
		if !m.totpPending || !totp.Validate(r.PostFormValue("totp_token"), m.totpSecret) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprintf(w, `<form><input type="hidden" name="csrf_token" value="%s"/>Invalid code</form>`, mockTOTPCSRF)
			return
		}
		m.totpPending = false
		m.authenticated = true
		w.Header().Set("Location", "/web")
		w.WriteHeader(http.StatusSeeOther)

	case r.URL.Path == "/hr_attendance/systray_check_in_out":
		if !m.authenticated {
			writeJSON(w, map[string]interface{}{
				"jsonrpc": "2.0", "id": 1,
				"error": map[string]interface{}{
					"code":    100,
					"message": "Odoo Session Expired",
					"data":    map[string]interface{}{"message": "Session expired"},
				},
			})
			return
		}
		if m.callError {
			writeJSON(w, map[string]interface{}{
				"jsonrpc": "2.0", "id": 1,
				"error": map[string]interface{}{
					"code":    200,
					"message": "Odoo Server Error",
					"data":    map[string]interface{}{"message": "Something went wrong"},
				},
			})
			return
		}
		writeJSON(w, map[string]interface{}{
			"jsonrpc": "2.0", "id": 1,
			"result": map[string]interface{}{"attendance_state": "checked_in"},
		})

	default:
		http.NotFound(w, r)
	}
}

// TestJSONRPCSession_Authenticate_MultiDB_TOTP is the #54 regression test:
// on a multi-db server the TOTP route 404s until the session is bound to
// the database, so authentication must go through the HTML login flow.
func TestJSONRPCSession_Authenticate_MultiDB_TOTP(t *testing.T) {
	m := newMockOdooServer(t, &mockOdooServer{
		multiDB:    true,
		totpSecret: testTOTPSecret,
		db:         "testdb",
		login:      "user@test.com",
		password:   "pass",
	})

	s := newJSONRPCSession(m.URL, "testdb", "user@test.com", "pass", testTOTPSecret)
	if err := s.authenticate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.authenticated {
		t.Error("expected client session authenticated = true")
	}
	if !m.authenticated {
		t.Error("expected server session to be finalized")
	}
}

// TestJSONRPCSession_Authenticate_SingleDB_TOTP guards the previously
// working case: single-db servers must keep authenticating fine.
func TestJSONRPCSession_Authenticate_SingleDB_TOTP(t *testing.T) {
	m := newMockOdooServer(t, &mockOdooServer{
		multiDB:    false,
		totpSecret: testTOTPSecret,
		db:         "testdb",
		login:      "user@test.com",
		password:   "pass",
	})

	s := newJSONRPCSession(m.URL, "testdb", "user@test.com", "pass", testTOTPSecret)
	if err := s.authenticate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.authenticated {
		t.Error("expected server session to be finalized")
	}
}

func TestJSONRPCSession_Authenticate_NoTOTP(t *testing.T) {
	m := newMockOdooServer(t, &mockOdooServer{
		multiDB:  true,
		db:       "testdb",
		login:    "user@test.com",
		password: "secret123",
	})

	s := newJSONRPCSession(m.URL, "testdb", "user@test.com", "secret123", "")
	if err := s.authenticate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.authenticated {
		t.Error("expected authenticated = true")
	}

	// Verify the login form carried the credentials.
	if got := m.loginForm.Get("login"); got != "user@test.com" {
		t.Errorf("login = %q, want user@test.com", got)
	}
	if got := m.loginForm.Get("password"); got != "secret123" {
		t.Errorf("password = %q, want secret123", got)
	}
}

func TestJSONRPCSession_Authenticate_WrongPassword(t *testing.T) {
	m := newMockOdooServer(t, &mockOdooServer{
		multiDB:  true,
		db:       "testdb",
		login:    "user@test.com",
		password: "correct",
	})

	s := newJSONRPCSession(m.URL, "testdb", "user@test.com", "wrong", "")
	if err := s.authenticate(); err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
	if s.authenticated {
		t.Error("expected authenticated = false")
	}
}

func TestJSONRPCSession_Authenticate_TOTPRequiredNoSecret(t *testing.T) {
	m := newMockOdooServer(t, &mockOdooServer{
		multiDB:    true,
		totpSecret: testTOTPSecret,
		db:         "testdb",
		login:      "user@test.com",
		password:   "pass",
	})

	s := newJSONRPCSession(m.URL, "testdb", "user@test.com", "pass", "")
	err := s.authenticate()
	if err == nil {
		t.Fatal("expected error when 2FA is required without TOTP secret, got nil")
	}
	if got := err.Error(); got != "authentication failed: 2FA is enabled but no TOTP secret configured (set totp_secret in [op_secrets] or ODOO_TOTP_SECRET env var)" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestJSONRPCSession_Authenticate_TOTPWrongSecret(t *testing.T) {
	m := newMockOdooServer(t, &mockOdooServer{
		multiDB:    true,
		totpSecret: testTOTPSecret,
		db:         "testdb",
		login:      "user@test.com",
		password:   "pass",
	})

	// Client generates codes from a different secret than the server expects.
	s := newJSONRPCSession(m.URL, "testdb", "user@test.com", "pass", "XYNHNRPRJMRNG6UP")
	if err := s.authenticate(); err == nil {
		t.Fatal("expected TOTP verification error, got nil")
	}
	if m.authenticated {
		t.Error("expected server session to remain unauthenticated")
	}
}

func TestJSONRPCSession_Call(t *testing.T) {
	m := newMockOdooServer(t, &mockOdooServer{
		multiDB:    true,
		totpSecret: testTOTPSecret,
		db:         "testdb",
		login:      "user@test.com",
		password:   "pass",
	})

	s := newJSONRPCSession(m.URL, "testdb", "user@test.com", "pass", testTOTPSecret)
	result, err := s.call("/hr_attendance/systray_check_in_out", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.authenticated {
		t.Error("expected auto-authentication")
	}
	if result["attendance_state"] != "checked_in" {
		t.Errorf("result attendance_state = %v, want checked_in", result["attendance_state"])
	}
}

func TestJSONRPCSession_CallErrorResponse(t *testing.T) {
	m := newMockOdooServer(t, &mockOdooServer{
		multiDB:   true,
		db:        "testdb",
		login:     "user@test.com",
		password:  "pass",
		callError: true,
	})

	s := newJSONRPCSession(m.URL, "testdb", "user@test.com", "pass", "")
	_, err := s.call("/hr_attendance/systray_check_in_out", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseTOTPSecret_RawSecret(t *testing.T) {
	got := parseTOTPSecret("JBSWY3DPEHPK3PXP")
	if got != "JBSWY3DPEHPK3PXP" {
		t.Errorf("got %q, want JBSWY3DPEHPK3PXP", got)
	}
}

func TestParseTOTPSecret_OTPAuthURL(t *testing.T) {
	url := "otpauth://totp/example:user@example.com?secret=XYNHNRPRJMRNG6UP&issuer=example&algorithm=SHA1&digits=6&period=30"
	got := parseTOTPSecret(url)
	if got != "XYNHNRPRJMRNG6UP" {
		t.Errorf("got %q, want XYNHNRPRJMRNG6UP", got)
	}
}
