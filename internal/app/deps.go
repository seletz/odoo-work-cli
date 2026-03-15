package app

import (
	"fmt"

	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

// Deps holds runtime dependencies initialized by the root command.
type Deps struct {
	Config *config.Config
	Client odoo.Client
}

func (d *Deps) RequireConfig() (*config.Config, error) {
	if d == nil || d.Config == nil {
		return nil, fmt.Errorf("config not initialized")
	}
	return d.Config, nil
}

func (d *Deps) RequireClient() (odoo.Client, error) {
	if d == nil || d.Client == nil {
		return nil, fmt.Errorf("odoo client not initialized")
	}
	return d.Client, nil
}
