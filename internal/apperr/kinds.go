package apperr

// Kind is a stable, machine-readable error category. Its string value is
// part of the CLI's public contract: it is emitted as
// {"error":{"kind":…}} in JSON output and documented in the readme.
// Never rename an existing kind; add a new constant instead.
type Kind string

// Error kinds. To add a kind, declare a constant here, append it to
// Kinds, and document it in the readme ("Error kinds"). Nothing else in
// the rendering path needs to change.
const (
	// KindAuth: authentication failed (wrong API key or password, TOTP
	// rejected, session expired).
	KindAuth Kind = "auth"
	// KindAccessDenied: authenticated, but Odoo's access rules forbid the
	// operation (Odoo AccessError).
	KindAccessDenied Kind = "access_denied"
	// KindNotFound: the requested record does not exist or is not visible.
	KindNotFound Kind = "not_found"
	// KindValidation: invalid input — flags, arguments, or values Odoo
	// rejected as a user error.
	KindValidation Kind = "validation"
	// KindConflict: the request contradicts the current state (already
	// clocked in, not clocked in, …).
	KindConflict Kind = "conflict"
	// KindNetwork: Odoo could not be reached (DNS, connection refused,
	// timeout, TLS).
	KindNetwork Kind = "network"
	// KindConfig: the local configuration is missing or invalid.
	KindConfig Kind = "config"
	// KindSecretsLocked: the secret store (1Password) is locked or not
	// reachable, so op:// references could not be resolved.
	KindSecretsLocked Kind = "secrets_locked"
	// KindInternal: anything not classified otherwise (bug, unexpected
	// Odoo response, server error).
	KindInternal Kind = "internal"
)

// Kinds returns every defined kind, in documentation order.
func Kinds() []Kind {
	return []Kind{
		KindAuth,
		KindAccessDenied,
		KindNotFound,
		KindValidation,
		KindConflict,
		KindNetwork,
		KindConfig,
		KindSecretsLocked,
		KindInternal,
	}
}
