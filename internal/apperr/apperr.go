// Package apperr defines the CLI's central error type: an error that
// carries a stable Kind, so that the root command can render every failure
// as {"error":{"kind":…,"message":…}} when JSON output is active.
//
// Errors get a kind in one of two ways:
//
//  1. At the origin, with New, Wrap or Wrapf. This is the preferred way for
//     errors the CLI creates itself.
//  2. By a classifier registered with RegisterClassifier, for errors that
//     come from libraries (go-odoo, net/http) and cannot be typed at the
//     origin.
//
// KindOf resolves the kind of any error; unclassified errors are
// KindInternal.
package apperr

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"sync"
	"syscall"
)

// Error is an error with a stable Kind.
type Error struct {
	// Kind is the stable category of the error.
	Kind Kind
	// Msg is the human-readable message. When empty, the wrapped error's
	// message is used.
	Msg string
	// Err is the optional underlying error.
	Err error
}

// Error returns the message, followed by the wrapped error if both exist.
func (e *Error) Error() string {
	switch {
	case e.Msg != "" && e.Err != nil:
		return e.Msg + ": " + e.Err.Error()
	case e.Msg != "":
		return e.Msg
	case e.Err != nil:
		return e.Err.Error()
	default:
		return string(e.Kind)
	}
}

// Unwrap returns the wrapped error.
func (e *Error) Unwrap() error { return e.Err }

// New creates an error of the given kind with a formatted message.
//
// :param kind: stable error kind
// :param format: fmt format string
// :param args: format arguments
// :returns: the new error
func New(kind Kind, format string, args ...any) *Error {
	return &Error{Kind: kind, Msg: fmt.Sprintf(format, args...)}
}

// Wrap assigns a kind to err, keeping its message. Returns nil for a nil err.
//
// :param kind: stable error kind
// :param err: error to wrap
// :returns: the wrapped error, or nil
func Wrap(kind Kind, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Kind: kind, Err: err}
}

// Wrapf assigns a kind to err and prefixes its message with a formatted
// context string, like fmt.Errorf("ctx: %w", err).
//
// :param kind: stable error kind
// :param err: error to wrap
// :param format: fmt format string for the prefix
// :param args: format arguments
// :returns: the wrapped error
func Wrapf(kind Kind, err error, format string, args ...any) *Error {
	return &Error{Kind: kind, Msg: fmt.Sprintf(format, args...), Err: err}
}

// Classifier maps an error it recognises to a kind. It returns false for
// errors it does not know.
type Classifier func(err error) (Kind, bool)

var (
	classifiersMu sync.RWMutex
	classifiers   = []Classifier{classifyNetwork}
)

// RegisterClassifier adds a classifier consulted by KindOf for errors that
// carry no typed kind. Classifiers run in registration order, after the
// built-in network classifier.
//
// :param c: the classifier
// :returns: a function that removes the classifier again (for tests)
func RegisterClassifier(c Classifier) (restore func()) {
	classifiersMu.Lock()
	defer classifiersMu.Unlock()
	classifiers = append(classifiers, c)
	idx := len(classifiers) - 1
	var once sync.Once
	return func() {
		once.Do(func() {
			classifiersMu.Lock()
			defer classifiersMu.Unlock()
			classifiers[idx] = nil
		})
	}
}

// KindOf returns the kind of err: the outermost typed Error in its chain,
// else the first matching classifier, else KindInternal. Returns "" for nil.
//
// :param err: the error to classify
// :returns: the error's kind
func KindOf(err error) Kind {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	classifiersMu.RLock()
	defer classifiersMu.RUnlock()
	for _, c := range classifiers {
		if c == nil {
			continue
		}
		if k, ok := c(err); ok {
			return k
		}
	}
	return KindInternal
}

// classifyNetwork recognises transport-level failures.
func classifyNetwork(err error) (Kind, bool) {
	var urlErr *url.Error
	var opErr *net.OpError
	var dnsErr *net.DNSError
	switch {
	case errors.As(err, &urlErr), errors.As(err, &opErr), errors.As(err, &dnsErr):
		return KindNetwork, true
	case errors.Is(err, syscall.ECONNREFUSED), errors.Is(err, syscall.ECONNRESET),
		errors.Is(err, syscall.EHOSTUNREACH), errors.Is(err, syscall.ENETUNREACH):
		return KindNetwork, true
	}
	return "", false
}
