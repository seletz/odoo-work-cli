package odoo

import (
	"errors"

	goOdoo "github.com/skilld-labs/go-odoo"
)

// ErrNotFound is returned when an Odoo search yields no results.
// This mirrors go-odoo's ErrNotFound so callers don't need to import go-odoo.
var ErrNotFound = goOdoo.ErrNotFound

// IsNotFound reports whether err indicates that no records were found.
func IsNotFound(err error) bool {
	return err != nil && errors.Is(err, ErrNotFound)
}
