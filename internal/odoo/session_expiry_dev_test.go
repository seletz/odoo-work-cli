//go:build devverify

package odoo

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"os"
	"testing"
)

// TestDevVerify_SessionExpiry reproduces #73 against the DEV instance: it
// ends a live web session server-side in the two ways Odoo 17 does it and
// checks that (1) the raw response is recognised as an expired session and
// (2) call() logs in again (TOTP included) and succeeds. With
// DEVVERIFY_CLOCK=1 it also clocks in and out across an invalidated session
// (only when the dev user is clocked out) and unlinks the record afterwards.
// Run with:
//
//	go test -tags devverify -count=1 -run TestDevVerify_SessionExpiry -v ./internal/odoo/
func TestDevVerify_SessionExpiry(t *testing.T) {
	baseURL := os.Getenv("ODOO_URL")
	if baseURL == "" {
		t.Skip("ODOO_URL not set")
	}
	client, err := NewXMLRPCClient(baseURL, os.Getenv("ODOO_DATABASE"),
		os.Getenv("ODOO_USERNAME"), os.Getenv("ODOO_PASSWORD"),
		os.Getenv("ODOO_WEB_PASSWORD"), os.Getenv("ODOO_TOTP_SECRET"), nil)
	if err != nil {
		t.Fatalf("connecting: %v", err)
	}
	defer client.Close()

	s, err := client.jsonSession()
	if err != nil {
		t.Fatalf("jsonSession: %v", err)
	}
	const info = "/web/session/get_session_info"

	expirations := []struct {
		name   string
		expire func(t *testing.T)
	}{
		{
			// Logout that keeps the database binding (what Odoo does when
			// the session token check fails): db-bound, no user → code 100.
			name: "logout, db kept",
			expire: func(t *testing.T) {
				if _, err := s.post("/web/session/destroy", map[string]interface{}{}); err != nil {
					t.Fatalf("destroying session: %v", err)
				}
				if _, err := s.fetchCSRFToken("/web/login?db=" + url.QueryEscape(s.database)); err != nil {
					t.Fatalf("re-binding database: %v", err)
				}
			},
		},
		{
			// The cookie names a session the server does not have (GC'd,
			// restart) → fresh unbound session → nodb dispatch.
			name: "session unknown to server",
			expire: func(t *testing.T) {
				buf := make([]byte, 20)
				_, _ = rand.Read(buf)
				u, _ := url.Parse(baseURL)
				s.httpClient.Jar.SetCookies(u, []*http.Cookie{{Name: "session_id", Value: hex.EncodeToString(buf), Path: "/"}})
			},
		},
	}

	for _, e := range expirations {
		t.Run(e.name, func(t *testing.T) {
			if _, err := s.call(info, map[string]interface{}{}); err != nil {
				t.Fatalf("establishing session: %v", err)
			}
			e.expire(t)

			_, err := s.post(info, map[string]interface{}{})
			if !errors.Is(err, errSessionExpired) {
				t.Fatalf("raw call after expiry: got %v, want errSessionExpired", err)
			}
			t.Logf("detected: %v", err)

			res, err := s.call(info, map[string]interface{}{})
			if err != nil {
				t.Fatalf("call after expiry did not recover: %v", err)
			}
			if uid, _ := res["uid"].(float64); uid <= 0 {
				t.Fatalf("recovered session has no uid: %v", res["uid"])
			}
			t.Logf("recovered: uid=%v (totp=%v)", res["uid"], s.totpCompleted)
		})
	}

	t.Run("clock in/out across expired sessions", func(t *testing.T) {
		if os.Getenv("DEVVERIFY_CLOCK") != "1" {
			t.Skip("set DEVVERIFY_CLOCK=1 to toggle attendance on dev")
		}
		status, err := client.AttendanceStatus()
		if err != nil {
			t.Fatalf("AttendanceStatus: %v", err)
		}
		if status.ClockedIn {
			t.Skip("dev user is clocked in; not touching its attendance")
		}

		expirations[1].expire(t)
		id, err := client.ClockIn()
		if err != nil {
			t.Fatalf("ClockIn after expiry: %v", err)
		}
		defer func() {
			if err := client.client.Delete("hr.attendance", []int64{id}); err != nil {
				t.Errorf("cleanup unlink of attendance %d failed: %v", id, err)
			} else {
				t.Logf("cleanup: unlinked attendance %d", id)
			}
		}()
		t.Logf("clocked in: attendance %d", id)

		expirations[0].expire(t)
		rec, err := client.ClockOut()
		if err != nil {
			t.Fatalf("ClockOut after expiry: %v", err)
		}
		if rec.ID != id || rec.CheckOut == nil {
			t.Fatalf("ClockOut returned %+v, want closed record %d", rec, id)
		}
		t.Logf("clocked out: attendance %d", rec.ID)
	})
}
