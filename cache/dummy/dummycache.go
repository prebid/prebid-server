package dummy

import (
	"fmt"

	"github.com/prebid/prebid-server/cache"
)

// Cache dummy config that will echo back results
type Cache struct {
	Accounts *AccountService
	Domains  *DomainService
	Apps     *AppsService
	Config   *ConfigService
}

// New creates new dummy.Cache
func New() (*Cache, error) {
	return &Cache{
		Accounts: &AccountService{},
		Domains:  &DomainService{},
		Apps:     &AppsService{},
		Config:   &ConfigService{},
	}, nil
}

func (c *Cache) Configure(cfg *cache.Configuration) error {
	return nil
}

// AccountService handles the account information
type AccountService struct {
}

// Get echos back the account
func (s *AccountService) Get(id string) (*cache.Account, error) {
	return &cache.Account{
		ID: id,
	}, nil
}

// Set will always return nil since this is a dummy service
func (s *AccountService) Set(account *cache.Account) error {
	return nil
}

// DomainService handles the domain information
type DomainService struct {
}

// Get echos back the domain
func (s *DomainService) Get(id string) (*cache.Domain, error) {
	return &cache.Domain{
		Domain: id,
	}, nil
}

// Set will always return nil since this is a dummy service
func (s *DomainService) Set(domain *cache.Domain) error {
	return nil
}

// AppsService handles apps information
type AppsService struct {
}

// Get echos back the app
func (s *AppsService) Get(id string) (*cache.App, error) {
	return &cache.App{
		Bundle: id,
	}, nil
}

// Set will always return nil since this is a dummy service
func (s *AppsService) Set(app *cache.App) error {
	return nil
}

// ConfigService not supported, always returns an error
type ConfigService struct {
}

// Get not supported, always returns an error
func (s *ConfigService) Get(id string) (*cache.Configuration, error) {
	return nil, fmt.Errorf("Not supported")
}

// Set not supported, always returns an error
func (s *ConfigService) Set(config *cache.Configuration) error {
	return fmt.Errorf("Not supported")
}

// Close will always return nil
func (c *Cache) Close() error {
	return nil
}
