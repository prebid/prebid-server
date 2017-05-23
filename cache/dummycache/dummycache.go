package dummycache

import (
	"fmt"

	"github.com/prebid/prebid-server/cache"
)

// Cache dummy config that will echo back results
type Cache struct {
	accounts *accountService
	domains  *domainService
	apps     *appsService
	config   *configService
}

// New creates new dummy.Cache
func New() (*Cache, error) {
	return &Cache{
		accounts: &accountService{},
		domains:  &domainService{},
		apps:     &appsService{},
		config:   &configService{},
	}, nil
}

func (c *Cache) Accounts() cache.AccountsService {
	return c.accounts
}
func (c *Cache) Domains() cache.DomainsService {
	return c.domains
}
func (c *Cache) Apps() cache.AppsService {
	return c.apps
}
func (c *Cache) Config() cache.ConfigService {
	return c.config
}

// AccountService handles the account information
type accountService struct {
}

// Get echos back the account
func (s *accountService) Get(id string) (*cache.Account, error) {
	return &cache.Account{
		ID: id,
	}, nil
}

// Set will always return nil since this is a dummy service
func (s *accountService) Set(account *cache.Account) error {
	return nil
}

// DomainService handles the domain information
type domainService struct {
}

// Get echos back the domain
func (s *domainService) Get(id string) (*cache.Domain, error) {
	return &cache.Domain{
		Domain: id,
	}, nil
}

// Set will always return nil since this is a dummy service
func (s *domainService) Set(domain *cache.Domain) error {
	return nil
}

// AppsService handles apps information
type appsService struct {
}

// Get echos back the app
func (s *appsService) Get(id string) (*cache.App, error) {
	return &cache.App{
		Bundle: id,
	}, nil
}

// Set will always return nil since this is a dummy service
func (s *appsService) Set(app *cache.App) error {
	return nil
}

// ConfigService not supported, always returns an error
type configService struct {
	c string
}

// Get not supported, always returns an error
func (s *configService) Get(id string) (string, error) {
	if s.c == "" {
		return s.c, fmt.Errorf("No configuration provided")
	}
	return s.c, nil
}

// Set will set a string in memory as the configuration
// this is so we can use it in tests such as pbs/pbsrequest_test.go
// it will ignore the id so this will pass tests
func (s *configService) Set(id, val string) error {
	s.c = val
	return nil
}

// Close will always return nil
func (c *Cache) Close() error {
	return nil
}
