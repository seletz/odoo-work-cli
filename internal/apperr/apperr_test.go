package apperr

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"syscall"
	"testing"
)

func TestKindOf(t *testing.T) {
	sentinel := errors.New("plain")

	tests := []struct {
		name string
		err  error
		want Kind
	}{
		{name: "nil error", err: nil, want: ""},
		{name: "plain error is internal", err: sentinel, want: KindInternal},
		{name: "typed error", err: New(KindValidation, "bad hours"), want: KindValidation},
		{name: "typed error wrapped by fmt", err: fmt.Errorf("ctx: %w", New(KindNotFound, "gone")), want: KindNotFound},
		{name: "outermost kind wins", err: Wrap(KindConflict, New(KindValidation, "inner")), want: KindConflict},
		{name: "url error is network", err: &url.Error{Op: "Post", URL: "http://x", Err: errors.New("dial tcp: refused")}, want: KindNetwork},
		{name: "op error is network", err: fmt.Errorf("calling: %w", &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}), want: KindNetwork},
		{name: "connection refused errno is network", err: fmt.Errorf("x: %w", syscall.ECONNREFUSED), want: KindNetwork},
		{name: "dns error is network", err: &net.DNSError{Err: "no such host", Name: "odoo.invalid"}, want: KindNetwork},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KindOf(tt.err); got != tt.want {
				t.Errorf("KindOf() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestKindOf_RegisteredClassifier(t *testing.T) {
	marker := errors.New("marker")
	restore := RegisterClassifier(func(err error) (Kind, bool) {
		if errors.Is(err, marker) {
			return KindAuth, true
		}
		return "", false
	})
	defer restore()

	if got := KindOf(fmt.Errorf("wrapped: %w", marker)); got != KindAuth {
		t.Errorf("KindOf() = %q, want %q", got, KindAuth)
	}
	// A typed error takes precedence over classifiers.
	if got := KindOf(Wrap(KindValidation, marker)); got != KindValidation {
		t.Errorf("KindOf(typed) = %q, want %q", got, KindValidation)
	}

	restore()
	if got := KindOf(marker); got != KindInternal {
		t.Errorf("after restore KindOf() = %q, want %q", got, KindInternal)
	}
}

func TestError_MessageAndUnwrap(t *testing.T) {
	inner := errors.New("connection reset")

	tests := []struct {
		name    string
		err     error
		wantMsg string
		wantIs  error
	}{
		{name: "New formats", err: New(KindValidation, "invalid hours %q", "x"), wantMsg: `invalid hours "x"`},
		{name: "Wrap keeps inner message", err: Wrap(KindNetwork, inner), wantMsg: "connection reset", wantIs: inner},
		{name: "Wrapf prefixes", err: Wrapf(KindNetwork, inner, "calling %s", "odoo"), wantMsg: "calling odoo: connection reset", wantIs: inner},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}
			if tt.wantIs != nil && !errors.Is(tt.err, tt.wantIs) {
				t.Errorf("errors.Is(err, inner) = false, want true")
			}
		})
	}
}

func TestWrap_Nil(t *testing.T) {
	if err := Wrap(KindNetwork, nil); err != nil {
		t.Errorf("Wrap(nil) = %v, want nil", err)
	}
}

func TestKinds_AreStableStrings(t *testing.T) {
	// These strings are a public contract (readme, #74 skill docs);
	// renaming one is a breaking change.
	want := map[Kind]string{
		KindAuth:         "auth",
		KindAccessDenied: "access_denied",
		KindNotFound:     "not_found",
		KindValidation:   "validation",
		KindConflict:     "conflict",
		KindNetwork:      "network",
		KindConfig:       "config",
		KindInternal:     "internal",
	}
	for k, s := range want {
		if string(k) != s {
			t.Errorf("kind %q, want %q", k, s)
		}
	}
	if len(Kinds()) != len(want) {
		t.Errorf("Kinds() has %d entries, want %d", len(Kinds()), len(want))
	}
}

// TestKinds_DocumentedInReadme keeps the readme's "Errors" table in sync:
// adding a kind without documenting it fails here.
func TestKinds_DocumentedInReadme(t *testing.T) {
	raw, err := os.ReadFile("../../readme.md")
	if err != nil {
		t.Fatalf("reading readme: %v", err)
	}
	readme := string(raw)
	for _, k := range Kinds() {
		if !strings.Contains(readme, "| `"+string(k)+"` ") {
			t.Errorf("kind %q is not documented in the readme's error kinds table", k)
		}
	}
}
