//go:build devverify

package odoo

import (
	"net/url"
	"os"
	"testing"
)

// TestDevVerify_LoginFlowCanary is the version-drift canary for the Odoo
// web-login flow (#57). It exercises every web-controller assumption the
// 2FA web session (clock in|out) encodes, against the DEV instance:
//
//  1. GET /web/login?db=<db> binds the session and serves a csrf_token input
//  2. POST /web/login accepts csrf_token/login/password/redirect and answers
//     with a 303 (to /web/login/totp when 2FA is pending)
//  3. POST /web/login/totp accepts csrf_token/totp_token and answers 303
//  4. the resulting session cookie authenticates JSON-RPC controller calls
//
// Run it via `mise run odoo:login-canary` — and always against dev BEFORE
// any Odoo upgrade reaches test/prod. A failure names the encoded
// assumption that broke.
func TestDevVerify_LoginFlowCanary(t *testing.T) {
	requireEnv := func(name string) string {
		value := os.Getenv(name)
		if value == "" {
			// A canary must fail loudly, not skip: a silently skipped
			// pre-upgrade check reads as a pass.
			t.Fatalf("CANARY MISCONFIGURED: %s is not set — run `mise run prepare_env` first", name)
		}
		return value
	}
	baseURL := requireEnv("ODOO_URL")
	database := requireEnv("ODOO_DATABASE")
	login := requireEnv("ODOO_USERNAME")
	webPassword := requireEnv("ODOO_WEB_PASSWORD")
	totpSecret := os.Getenv("ODOO_TOTP_SECRET")

	s := newJSONRPCSession(baseURL, database, login, webPassword, totpSecret)

	// Assumption 1 in isolation, so a markup change is reported as such
	// even before credentials come into play.
	if _, err := s.fetchCSRFToken("/web/login?db=" + url.QueryEscape(database)); err != nil {
		t.Fatalf("CANARY FAILED: %v", err)
	}
	t.Logf("PASS: GET /web/login?db=%s served a csrf_token input", database)

	// Assumptions 2 and 3: full login including redirect semantics and,
	// when 2FA is enabled, the TOTP challenge. Errors are self-diagnosing.
	if err := s.authenticate(); err != nil {
		t.Fatalf("CANARY FAILED: %v", err)
	}
	t.Log("PASS: POST /web/login accepted the credentials (303 redirect semantics intact)")

	switch {
	case totpSecret == "":
		t.Error("CANARY INCOMPLETE: ODOO_TOTP_SECRET is not set, so the /web/login/totp leg was not exercised — enable 2FA on the dev user and set the secret")
	case !s.totpCompleted:
		t.Error("CANARY INCOMPLETE: ODOO_TOTP_SECRET is set but Odoo never redirected to /web/login/totp — 2FA seems disabled on the dev user, so the TOTP leg was not exercised")
	default:
		t.Log("PASS: POST /web/login/totp completed the 2FA challenge")
	}

	// Assumption 4: the session cookie authenticates controller calls.
	// get_session_info is side-effect free (unlike the attendance toggle).
	info, err := s.call("/web/session/get_session_info", map[string]interface{}{})
	if err != nil {
		t.Fatalf("CANARY FAILED: session-authenticated JSON-RPC call rejected — the login flow no longer yields a usable web session; check the Odoo version: %v", err)
	}
	uid, ok := info["uid"].(float64)
	if !ok || uid <= 0 {
		t.Fatalf("CANARY FAILED: /web/session/get_session_info returned no uid (%v) — the session is not authenticated; check the Odoo version", info["uid"])
	}
	t.Logf("PASS: session-authenticated call succeeded (uid=%d, db=%v)", int64(uid), info["db"])
}
