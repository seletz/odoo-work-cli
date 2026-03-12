package odoo

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSONRPCSession_Authenticate(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/web/session/authenticate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)

		http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "abc123"})
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"uid": float64(42),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	s := newJSONRPCSession(srv.URL, "testdb", "user@test.com", "secret123", "")
	err := s.authenticate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.authenticated {
		t.Error("expected authenticated = true")
	}

	// Verify request body structure.
	params, _ := gotBody["params"].(map[string]interface{})
	if params["db"] != "testdb" {
		t.Errorf("db = %v, want testdb", params["db"])
	}
	if params["login"] != "user@test.com" {
		t.Errorf("login = %v, want user@test.com", params["login"])
	}
	if params["password"] != "secret123" {
		t.Errorf("password = %v, want secret123", params["password"])
	}
}

func TestJSONRPCSession_AuthenticateError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"error": map[string]interface{}{
				"code":    200,
				"message": "Odoo Server Error",
				"data": map[string]interface{}{
					"message": "Access Denied",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	s := newJSONRPCSession(srv.URL, "testdb", "bad@test.com", "wrong", "")
	err := s.authenticate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestJSONRPCSession_AuthenticateUIDFalse_NoTOTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"uid": false,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	s := newJSONRPCSession(srv.URL, "testdb", "user@test.com", "pass", "")
	err := s.authenticate()
	if err == nil {
		t.Fatal("expected error for uid=false without TOTP secret, got nil")
	}
	if got := err.Error(); got != "authentication failed: 2FA is enabled but no TOTP secret configured (set totp_secret in [op_secrets] or ODOO_TOTP_SECRET env var)" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestJSONRPCSession_AuthenticateWithTOTP(t *testing.T) {
	var totpPath string
	var totpFormValues map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/web/session/authenticate":
			// Return uid=false to trigger 2FA flow.
			http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "pre-auth-session"})
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result": map[string]interface{}{
					"uid": false,
				},
			}
			_ = json.NewEncoder(w).Encode(resp)

		case "/web/login/totp":
			if r.Method == http.MethodGet {
				// Return an HTML page with a CSRF token.
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte(`<form><input type="hidden" name="csrf_token" value="test-csrf-42"/></form>`))
				return
			}
			// POST: verify TOTP submission.
			totpPath = r.URL.Path
			_ = r.ParseForm()
			totpFormValues = map[string]string{
				"csrf_token": r.FormValue("csrf_token"),
				"totp_token": r.FormValue("totp_token"),
			}
			// Simulate success: redirect to /web.
			w.Header().Set("Location", "/web")
			w.WriteHeader(http.StatusSeeOther)

		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	// Use a known TOTP secret and verify that a code was submitted.
	s := newJSONRPCSession(srv.URL, "testdb", "user@test.com", "pass", "JBSWY3DPEHPK3PXP")
	err := s.authenticate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.authenticated {
		t.Error("expected authenticated = true after TOTP")
	}
	if totpPath != "/web/login/totp" {
		t.Errorf("TOTP path = %q, want /web/login/totp", totpPath)
	}
	if totpFormValues["csrf_token"] != "test-csrf-42" {
		t.Errorf("csrf_token = %q, want test-csrf-42", totpFormValues["csrf_token"])
	}
	if totpFormValues["totp_token"] == "" {
		t.Error("totp_token was empty, expected a generated code")
	}
}

func TestJSONRPCSession_TOTPVerificationFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/web/session/authenticate":
			w.Header().Set("Content-Type", "application/json")
			http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "pre-auth"})
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result": map[string]interface{}{
					"uid": false,
				},
			}
			_ = json.NewEncoder(w).Encode(resp)

		case "/web/login/totp":
			if r.Method == http.MethodGet {
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte(`<form><input type="hidden" name="csrf_token" value="csrf-abc"/></form>`))
				return
			}
			// Return 200 (form re-rendered) = verification failed.
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<form>Error: invalid code</form>`))
		}
	}))
	defer srv.Close()

	s := newJSONRPCSession(srv.URL, "testdb", "user@test.com", "pass", "JBSWY3DPEHPK3PXP")
	err := s.authenticate()
	if err == nil {
		t.Fatal("expected TOTP verification error, got nil")
	}
}

func TestJSONRPCSession_Call(t *testing.T) {
	authCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/web/session/authenticate" {
			authCalled = true
			http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "abc123"})
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result":  map[string]interface{}{"uid": float64(42)},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path != "/hr_attendance/systray_check_in_out" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      2,
			"result":  map[string]interface{}{"attendance_state": "checked_in"},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	s := newJSONRPCSession(srv.URL, "testdb", "user@test.com", "secret123", "")
	result, err := s.call("/hr_attendance/systray_check_in_out", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !authCalled {
		t.Error("expected auto-authentication")
	}
	if result["attendance_state"] != "checked_in" {
		t.Errorf("result attendance_state = %v, want checked_in", result["attendance_state"])
	}
}

func TestJSONRPCSession_CallErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/web/session/authenticate" {
			http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "abc123"})
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result":  map[string]interface{}{"uid": float64(42)},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      2,
			"error": map[string]interface{}{
				"code":    200,
				"message": "Odoo Server Error",
				"data": map[string]interface{}{
					"message": "Something went wrong",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	s := newJSONRPCSession(srv.URL, "testdb", "user@test.com", "secret123", "")
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
