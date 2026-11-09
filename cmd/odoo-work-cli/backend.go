package main

import (
	"os"
	"strings"
	"time"

	"github.com/seletz/odoo-work-cli/internal/apperr"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/odoo/fake"
)

const (
	backendOdoo = "odoo"
	backendFake = "fake"
)

// checkBackendFlag validates the --backend flag value.
//
// :param name: the flag value ("" when not set)
// :returns: a validation error for an unknown backend
func checkBackendFlag(name string) error {
	switch name {
	case "", backendOdoo, backendFake:
		return nil
	}
	return apperr.New(apperr.KindValidation, "unknown --backend %q: use %s or %s", name, backendOdoo, backendFake)
}

// resolveBackend returns the backend to use: the --backend flag if set,
// else ODOO_WORK_CLI_BACKEND, else odoo. An unknown env value is a config
// error.
//
// :param flag: the --backend flag value ("" when not set)
// :returns: backendOdoo or backendFake
func resolveBackend(flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	switch v := strings.ToLower(strings.TrimSpace(os.Getenv(fake.BackendEnvVar))); v {
	case "", backendOdoo:
		return backendOdoo, nil
	case backendFake:
		return backendFake, nil
	default:
		return "", apperr.New(apperr.KindConfig, "unknown %s %q: use %s or %s",
			fake.BackendEnvVar, v, backendOdoo, backendFake)
	}
}

// loaderFor returns the config + client loader for the chosen backend.
// The fake backend reads no config files and never calls 1Password.
func loaderFor(flag, cfgFile string, setup setupFunc) func() (*config.Config, odoo.Client, error) {
	return func() (*config.Config, odoo.Client, error) {
		backend, err := resolveBackend(flag)
		if err != nil {
			return nil, nil, err
		}
		if backend == backendFake {
			return fakeBackendSetup()
		}
		return setup(cfgFile)
	}
}

// fakeBackendSetup opens the fake backend with a default config.
func fakeBackendSetup() (*config.Config, odoo.Client, error) {
	client, err := fake.Open(fake.StatePath(), time.Now)
	if err != nil {
		return nil, nil, err
	}
	cfg := &config.Config{URL: "fake://", Database: "fake", Username: "fake"}
	cfg.ApplyDefaults()
	return cfg, client, nil
}
