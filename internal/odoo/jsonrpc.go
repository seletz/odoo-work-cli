package odoo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pquerna/otp/totp"
)

// jsonRPCSession manages a JSON-RPC session with Odoo.
// It authenticates through the HTML web login flow (/web/login) and
// maintains session cookies for subsequent requests.
// This is needed for controller endpoints (like attendance toggle)
// that require an authenticated web session.
type jsonRPCSession struct {
	baseURL       string
	database      string
	login         string
	password      string
	totpSecret    string
	httpClient    *http.Client
	authenticated bool
	reqID         atomic.Int64
}

// newJSONRPCSession creates a new JSON-RPC session (not yet authenticated).
func newJSONRPCSession(baseURL, database, login, password, totpSecret string) *jsonRPCSession {
	jar, _ := cookiejar.New(nil)
	return &jsonRPCSession{
		baseURL:    baseURL,
		database:   database,
		login:      login,
		password:   password,
		totpSecret: totpSecret,
		httpClient: &http.Client{
			Jar: jar,
		},
	}
}

// authenticate logs in through the HTML web login flow: GET
// /web/login?db=<db> for the CSRF token, then POST the credentials to
// /web/login. On multi-db servers this is the only flow that binds the
// session to a database — the JSON /web/session/authenticate endpoint
// leaves a 2FA-pending session unbound, so /web/login/totp is dispatched
// in nodb mode and 404s (#54). When Odoo redirects to /web/login/totp,
// the 2FA challenge is completed with a generated TOTP code.
func (s *jsonRPCSession) authenticate() error {
	// GET /web/login?db=<db> binds the session to the database
	// (ensure_db) and yields the CSRF token for the login form.
	csrfToken, err := s.fetchCSRFToken("/web/login?db=" + url.QueryEscape(s.database))
	if err != nil {
		return fmt.Errorf("fetching login form: %w", err)
	}

	// POST credentials without following redirects: a redirect means the
	// credentials were accepted (to /web/login/totp when 2FA is pending),
	// a 200 means the login form was re-rendered with an error.
	form := url.Values{
		"csrf_token": {csrfToken},
		"login":      {s.login},
		"password":   {s.password},
		"redirect":   {""},
	}
	resp, err := s.noRedirectClient().PostForm(s.baseURL+"/web/login", form)
	if err != nil {
		return fmt.Errorf("submitting login form: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusSeeOther, http.StatusFound:
		if strings.Contains(resp.Header.Get("Location"), "/web/login/totp") {
			if s.totpSecret == "" {
				return fmt.Errorf("authentication failed: 2FA is enabled but no TOTP secret configured (set totp_secret in [op_secrets] or ODOO_TOTP_SECRET env var)")
			}
			return s.completeTOTP()
		}
		s.authenticated = true
		return nil
	case http.StatusOK:
		return fmt.Errorf("authentication failed: Odoo rejected the login (the web session needs the login password, not the API key)")
	default:
		return fmt.Errorf("authentication failed: unexpected status %d from /web/login", resp.StatusCode)
	}
}

// noRedirectClient returns a client sharing the session cookie jar that
// does not follow redirects, so redirect status codes stay observable.
func (s *jsonRPCSession) noRedirectClient() *http.Client {
	return &http.Client{
		Jar: s.httpClient.Jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// completeTOTP finishes the 2FA flow by generating a TOTP code and
// submitting it to /web/login/totp. The session cookie from authenticate()
// carries the pre_uid that Odoo uses to identify the pending login.
func (s *jsonRPCSession) completeTOTP() error {
	secret := parseTOTPSecret(s.totpSecret)

	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		return fmt.Errorf("generating TOTP code: %w", err)
	}

	// GET /web/login/totp to obtain the CSRF token from the form.
	csrfToken, err := s.fetchCSRFToken("/web/login/totp")
	if err != nil {
		return fmt.Errorf("fetching CSRF token: %w", err)
	}

	// POST the TOTP code as form data. Success is a redirect (302/303),
	// failure re-renders the form (200).
	form := url.Values{
		"csrf_token": {csrfToken},
		"totp_token": {code},
	}
	totpResp, err := s.noRedirectClient().PostForm(s.baseURL+"/web/login/totp", form)
	if err != nil {
		return fmt.Errorf("submitting TOTP code: %w", err)
	}
	defer func() { _ = totpResp.Body.Close() }()

	// 303 redirect means session was finalized (success).
	// 200 means the form was re-rendered (wrong code).
	if totpResp.StatusCode == http.StatusSeeOther || totpResp.StatusCode == http.StatusFound {
		s.authenticated = true
		return nil
	}

	return fmt.Errorf("TOTP verification failed (status %d): check that totp_secret is correct", totpResp.StatusCode)
}

// csrfTokenRe matches the CSRF token hidden input in Odoo HTML forms.
var csrfTokenRe = regexp.MustCompile(`name="csrf_token"\s+value="([^"]+)"`)

// fetchCSRFToken GETs a page and extracts the csrf_token from the HTML form.
func (s *jsonRPCSession) fetchCSRFToken(path string) (string, error) {
	resp, err := s.httpClient.Get(s.baseURL + path)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	matches := csrfTokenRe.FindSubmatch(body)
	if matches == nil {
		return "", fmt.Errorf("CSRF token not found in %s response", path)
	}
	return string(matches[1]), nil
}

// parseTOTPSecret extracts the base32 secret from either a raw secret
// string or an otpauth:// URL.
func parseTOTPSecret(s string) string {
	if strings.HasPrefix(s, "otpauth://") {
		u, err := url.Parse(s)
		if err == nil {
			if secret := u.Query().Get("secret"); secret != "" {
				return secret
			}
		}
	}
	return s
}

// call makes a JSON-RPC call to the given path, auto-authenticating if needed.
func (s *jsonRPCSession) call(path string, params map[string]interface{}) (map[string]interface{}, error) {
	if !s.authenticated {
		if err := s.authenticate(); err != nil {
			return nil, err
		}
	}

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      s.reqID.Add(1),
		"method":  "call",
		"params":  params,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	resp, err := s.httpClient.Post(s.baseURL+path, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("calling %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		Error *struct {
			Message string                 `json:"message"`
			Data    map[string]interface{} `json:"data"`
		} `json:"error"`
		Result map[string]interface{} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response from %s: %w", path, err)
	}
	if result.Error != nil {
		msg := result.Error.Message
		if data, ok := result.Error.Data["message"].(string); ok {
			msg = data
		}
		return nil, fmt.Errorf("JSON-RPC error from %s: %s", path, msg)
	}

	return result.Result, nil
}
