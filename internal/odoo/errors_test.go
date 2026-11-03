package odoo

import (
	"errors"
	"fmt"
	"testing"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	goOdoo "github.com/skilld-labs/go-odoo"
)

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "direct ErrNotFound",
			err:  ErrNotFound,
			want: true,
		},
		{
			name: "wrapped ErrNotFound",
			err:  fmt.Errorf("account.analytic.line model was %w with criteria", ErrNotFound),
			want: true,
		},
		{
			name: "unrelated error",
			err:  errors.New("timeout"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFound(tt.err)
			if got != tt.want {
				t.Errorf("IsNotFound(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		want   apperr.Kind
		wantOK bool
	}{
		{name: "go-odoo not found", err: fmt.Errorf("fetching: %w", ErrNotFound), want: apperr.KindNotFound, wantOK: true},
		{name: "go-odoo authentication", err: fmt.Errorf("connecting to odoo: %w", goOdoo.ErrClientAuthentication), want: apperr.KindAuth, wantOK: true},
		{name: "go-odoo invalid configuration", err: goOdoo.ErrClientConfigurationInvalid, want: apperr.KindConfig, wantOK: true},
		{name: "fault 3 access denied is auth", err: errors.New("Fault(3): Access Denied"), want: apperr.KindAuth, wantOK: true},
		{name: "fault 4 access error", err: fmt.Errorf("updating timesheet 5: %w", errors.New("Fault(4): You are not allowed")), want: apperr.KindAccessDenied, wantOK: true},
		{name: "fault 2 user error is validation", err: errors.New("Fault(2): hours must be positive"), want: apperr.KindValidation, wantOK: true},
		{name: "fault 1 application error", err: errors.New("Fault(1): Traceback ..."), want: apperr.KindInternal, wantOK: true},
		{name: "unrelated error", err: errors.New("timeout"), wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ClassifyError(tt.err)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("ClassifyError() = (%q, %v), want (%q, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
