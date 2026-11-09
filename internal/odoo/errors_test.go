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

func TestFriendlyMissingError(t *testing.T) {
	defer apperr.RegisterClassifier(ClassifyError)()
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"project fault 2", errors.New("Fault(2): Record does not exist or has been deleted.\n(Record: project.project(67,), User: 2)"), "project 67 does not exist"},
		{"project short form", errors.New("Record does not exist or has been deleted. (Record: project.project(67,))"), "project 67 does not exist"},
		{"task", errors.New("Fault(2): Record does not exist or has been deleted. (Record: project.task(12,))"), "task 12 does not exist"},
		{"timesheet entry", errors.New("Record does not exist or has been deleted. (Record: account.analytic.line(5,))"), "timesheet entry 5 does not exist"},
		{"attendance", errors.New("Record does not exist or has been deleted. (Record: hr.attendance(9,))"), "attendance record 9 does not exist"},
		{"unknown model keeps model name", errors.New("Record does not exist or has been deleted. (Record: res.partner(3,))"), "res.partner 3 does not exist"},
		{"multiple ids", errors.New("Record does not exist or has been deleted. (Record: project.project(67, 68))"), "project 67, 68 does not exist"},
		{"wrapped", fmt.Errorf("creating: %w", errors.New("Record does not exist or has been deleted. (Record: project.task(1,))")), "task 1 does not exist"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FriendlyError(tt.err)
			if got.Error() != tt.want {
				t.Errorf("message = %q, want %q", got.Error(), tt.want)
			}
			if !errors.Is(got, tt.err) {
				t.Error("original error is not wrapped")
			}
			if k, want := apperr.KindOf(got), apperr.KindOf(tt.err); k != want {
				t.Errorf("kind = %q, want unchanged %q", k, want)
			}
		})
	}
}

func TestFriendlyError_Fault2StaysValidation(t *testing.T) {
	defer apperr.RegisterClassifier(ClassifyError)()
	orig := errors.New("Fault(2): Record does not exist or has been deleted. (Record: project.project(67,))")
	if k := apperr.KindOf(FriendlyError(orig)); k != apperr.KindValidation {
		t.Errorf("kind = %q, want validation", k)
	}
}

func TestFriendlyError_Passthrough(t *testing.T) {
	if FriendlyError(nil) != nil {
		t.Error("nil must stay nil")
	}
	orig := errors.New("Fault(2): something else")
	if got := FriendlyError(orig); got != orig {
		t.Errorf("unrelated error changed: %v", got)
	}
}

func TestFriendlyError_KeepsExistingKind(t *testing.T) {
	orig := apperr.New(apperr.KindNotFound, "Record does not exist or has been deleted. (Record: project.task(1,))")
	if k := apperr.KindOf(FriendlyError(orig)); k != apperr.KindNotFound {
		t.Errorf("kind = %q, want not_found", k)
	}
}
