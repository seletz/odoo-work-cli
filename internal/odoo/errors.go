package odoo

import (
	"errors"
	"regexp"
	"strings"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	goOdoo "github.com/skilld-labs/go-odoo"
)

// ErrNotFound is returned when an Odoo search yields no results.
// This mirrors go-odoo's ErrNotFound so callers don't need to import go-odoo.
var ErrNotFound = goOdoo.ErrNotFound

// IsNotFound reports whether err indicates that no records were found.
func IsNotFound(err error) bool {
	return err != nil && errors.Is(err, ErrNotFound)
}

// faultPattern extracts the XML-RPC fault code. go-odoo's transport
// (kolo/xmlrpc over net/rpc) flattens faults into strings of the form
// "Fault(<code>): <message>", so the code is only available textually.
var faultPattern = regexp.MustCompile(`Fault\((\d+)\):`)

// Odoo's XML-RPC fault codes (odoo/service/wsgi_server.py).
const (
	faultApplicationError = "1" // unhandled exception
	faultWarning          = "2" // UserError, ValidationError, MissingError
	faultAccessDenied     = "3" // AccessDenied: bad login / API key
	faultAccessError      = "4" // AccessError: ACL / record rule
)

// ClassifyError maps errors produced by go-odoo and Odoo XML-RPC faults to
// stable apperr kinds. It is registered with apperr.RegisterClassifier by
// the root command. Errors the CLI creates itself carry their kind at the
// origin and never reach this function.
//
// :param err: error to classify
// :returns: the kind and true, or false if err is not an Odoo error
func ClassifyError(err error) (apperr.Kind, bool) {
	switch {
	case errors.Is(err, goOdoo.ErrNotFound):
		return apperr.KindNotFound, true
	case errors.Is(err, goOdoo.ErrClientAuthentication), errors.Is(err, goOdoo.ErrClientNotAuthenticate):
		return apperr.KindAuth, true
	case errors.Is(err, goOdoo.ErrClientConfigurationInvalid):
		return apperr.KindConfig, true
	}
	var me *missingRecordError
	if errors.As(err, &me) {
		// The rewrite must not change the kind: classify the original.
		return ClassifyError(me.err)
	}
	m := faultPattern.FindStringSubmatch(err.Error())
	if m == nil {
		return "", false
	}
	switch m[1] {
	case faultAccessDenied:
		return apperr.KindAuth, true
	case faultAccessError:
		return apperr.KindAccessDenied, true
	case faultWarning:
		return apperr.KindValidation, true
	default:
		return apperr.KindInternal, true
	}
}

// missingPattern matches Odoo's MissingError text, e.g.
// "Record does not exist or has been deleted. (Record: project.project(67,))".
// The ID list may hold several IDs: "(67, 68)".
var missingPattern = regexp.MustCompile(`Record does not exist or has been deleted\.\s*\(Record: ([\w.]+)\(([\d,\s]*)\)`)

// modelNames gives the user-facing name of the Odoo models the CLI writes.
var modelNames = map[string]string{
	"project.project":       "project",
	"project.task":          "task",
	"account.analytic.line": "timesheet entry",
	"hr.attendance":         "attendance record",
}

// missingRecordError is the friendly form of Odoo's MissingError. Its
// message is the clean text only; the original error stays reachable via
// Unwrap.
type missingRecordError struct {
	msg string
	err error
}

func (e *missingRecordError) Error() string { return e.msg }
func (e *missingRecordError) Unwrap() error { return e.err }

// FriendlyError turns Odoo's MissingError text into "<name> <id> does not
// exist" (the wording of the --dry-run check) without any extra Odoo
// query. The original error stays wrapped and its kind is unchanged. Other
// errors are returned as they are.
//
// :param err: error to rewrite
// :returns: the rewritten error, or err itself if it is not a MissingError
func FriendlyError(err error) error {
	if err == nil {
		return nil
	}
	m := missingPattern.FindStringSubmatch(err.Error())
	if m == nil {
		return err
	}
	name, ok := modelNames[m[1]]
	if !ok {
		name = m[1]
	}
	ids := strings.Join(strings.Fields(strings.ReplaceAll(m[2], ",", " ")), ", ")
	return &missingRecordError{msg: name + " " + ids + " does not exist", err: err}
}
