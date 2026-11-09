package app

import (
	"fmt"

	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
)

// Loader loads the configuration (including secrets) and connects to Odoo.
type Loader func() (*config.Config, odoo.Client, error)

// Deps holds runtime dependencies. Config and Client are either set
// directly or loaded lazily through Loader on first use, so commands can
// validate their input before secrets are resolved.
type Deps struct {
	Config *config.Config
	Client odoo.Client
	// Loader, when set, is called at most once on the first RequireConfig
	// or RequireClient call. Its error is cached.
	Loader Loader

	loaded  bool
	loadErr error
}

// load runs the Loader once.
func (d *Deps) load() error {
	if d.Loader == nil {
		return nil
	}
	if !d.loaded {
		d.loaded = true
		cfg, client, err := d.Loader()
		if err != nil {
			d.loadErr = err
		} else {
			d.Config = cfg
			d.Client = client
		}
	}
	return d.loadErr
}

// RequireConfig returns the configuration, loading it on first use.
//
// :returns: the config, or the load error
func (d *Deps) RequireConfig() (*config.Config, error) {
	if d == nil {
		return nil, fmt.Errorf("config not initialized")
	}
	if err := d.load(); err != nil {
		return nil, err
	}
	if d.Config == nil {
		return nil, fmt.Errorf("config not initialized")
	}
	return d.Config, nil
}

// RequireClient returns the Odoo client, loading it on first use.
//
// :returns: the client, or the load error
func (d *Deps) RequireClient() (odoo.Client, error) {
	if d == nil {
		return nil, fmt.Errorf("odoo client not initialized")
	}
	if err := d.load(); err != nil {
		return nil, err
	}
	if d.Client == nil {
		return nil, fmt.Errorf("odoo client not initialized")
	}
	return d.Client, nil
}

// Close closes the client if one was created. It never triggers loading.
func (d *Deps) Close() {
	if d != nil && d.Client != nil {
		d.Client.Close()
		d.Client = nil
	}
}
