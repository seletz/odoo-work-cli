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
// It handles authentication via /web/session/authenticate and
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

// authenticate performs JSON-RPC session authentication against Odoo.
// When 2FA (TOTP) is enabled, the initial authenticate returns uid=false
// and sets a pre_uid in the session. We then complete the TOTP challenge
// by POSTing the generated code to /web/login/totp.
func (s *jsonRPCSession) authenticate() error {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      s.reqID.Add(1),
		"method":  "call",
		"params": map[string]interface{}{
			"db":       s.database,
			"login":    s.login,
			"password": s.password,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshalling auth request: %w", err)
	}

	resp, err := s.httpClient.Post(s.baseURL+"/web/session/authenticate", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("authenticating: %w", err)
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
		return fmt.Errorf("decoding auth response: %w", err)
	}
	if result.Error != nil {
		msg := result.Error.Message
		if data, ok := result.Error.Data["message"].(string); ok {
			msg = data
		}
		return fmt.Errorf("authentication failed: %s", msg)
	}

	// Odoo returns uid: false when 2FA is required (pre_uid set in session).
	if uid, ok := result.Result["uid"]; ok {
		if uid == false || uid == nil {
			if s.totpSecret != "" {
				return s.completeTOTP()
			}
			return fmt.Errorf("authentication failed: 2FA is enabled but no TOTP secret configured (set totp_secret in [op_secrets] or ODOO_TOTP_SECRET env var)")
		}
	}

	s.authenticated = true
	return nil
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

	// POST the TOTP code as form data. Use a non-redirecting client
	// so we can distinguish success (302/303 redirect) from failure (200).
	noRedirect := &http.Client{
		Jar: s.httpClient.Jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	form := url.Values{
		"csrf_token": {csrfToken},
		"totp_token": {code},
	}
	totpResp, err := noRedirect.PostForm(s.baseURL+"/web/login/totp", form)
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
