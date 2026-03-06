package odoo

import (
	"fmt"

	goOdoo "github.com/skilld-labs/go-odoo"
)

// XMLRPCClient implements Client using the Odoo XML-RPC API.
type XMLRPCClient struct {
	client *goOdoo.Client
	login  string
}

// NewXMLRPCClient creates a new Odoo client and authenticates.
func NewXMLRPCClient(url, database, username, password string) (*XMLRPCClient, error) {
	c, err := goOdoo.NewClient(&goOdoo.ClientConfig{
		Admin:    username,
		Password: password,
		Database: database,
		URL:      url,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to odoo: %w", err)
	}
	return &XMLRPCClient{client: c, login: username}, nil
}

// Close closes the underlying XML-RPC connections.
func (x *XMLRPCClient) Close() {
	x.client.Close()
}

// ListProjects returns all projects from Odoo.
func (x *XMLRPCClient) ListProjects() ([]ProjectInfo, error) {
	criteria := goOdoo.NewCriteria()
	projects, err := x.client.FindProjectProjects(criteria, goOdoo.NewOptions())
	if err != nil {
		return nil, fmt.Errorf("fetching projects: %w", err)
	}

	result := make([]ProjectInfo, 0, len(*projects))
	for _, p := range *projects {
		result = append(result, ProjectInfo{
			ID:     p.Id.Get(),
			Name:   p.Name.Get(),
			Active: p.Active.Get(),
		})
	}
	return result, nil
}

// WhoAmI returns the identity of the currently authenticated user.
func (x *XMLRPCClient) WhoAmI() (*UserInfo, error) {
	criteria := goOdoo.NewCriteria().Add("login", "=", x.login)
	user, err := x.client.FindResUsers(criteria)
	if err != nil {
		return nil, fmt.Errorf("fetching user: %w", err)
	}

	info := &UserInfo{
		ID:    user.Id.Get(),
		Name:  user.Name.Get(),
		Login: user.Login.Get(),
		Email: user.Email.Get(),
	}
	if user.CompanyId != nil {
		info.Company = user.CompanyId.Name
	}

	return info, nil
}
