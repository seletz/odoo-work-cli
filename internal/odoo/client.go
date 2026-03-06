package odoo

// UserInfo holds identity information for the current Odoo user.
type UserInfo struct {
	ID       int64
	Name     string
	Login    string
	Email    string
	Company  string
}

// Client defines the interface for interacting with an Odoo instance.
type Client interface {
	// WhoAmI returns the identity of the currently authenticated user.
	WhoAmI() (*UserInfo, error)
}
