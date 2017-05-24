package filecache

import (
	"fmt"
	"io/ioutil"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/cache"
	"gopkg.in/yaml.v2"
)

type shared struct {
	Configs  map[string]string
	Domains  map[string]bool
	Apps     map[string]bool
	Accounts map[string]bool
}

// Cache is a file backed cache
type Cache struct {
	shared   *shared
	accounts *accountService
	domains  *domainService
	apps     *appsService
	config   *configService
}

type fileConfig struct {
	ID     string `yaml:"id"`
	Config string `yaml:"config"`
}

type fileCacheFile struct {
	Configs  []fileConfig `yaml:"configs"`
	Domains  []string     `yaml:"domains"`
	Apps     []string     `yaml:"apps"`
	Accounts []string     `yaml:"accounts"`
}

// New will load the file into memory
func New(filename string) (*Cache, error) {
	if glog.V(2) {
		glog.Infof("Reading inventory urls from %s", filename)
	}

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	if glog.V(2) {
		glog.Infof("Parsing filecache YAML")
	}

	var u fileCacheFile
	if err = yaml.Unmarshal(b, &u); err != nil {
		return nil, err
	}

	if glog.V(2) {
		glog.Infof("Building URL map")
	}

	s := &shared{}

	s.Configs = make(map[string]string, len(u.Configs))
	for _, config := range u.Configs {
		s.Configs[config.ID] = config.Config
	}
	glog.Infof("Loaded %d configs", len(u.Configs))

	s.Domains = make(map[string]bool, len(u.Domains))
	for _, domain := range u.Domains {
		s.Domains[domain] = true
	}
	glog.Infof("Loaded %d domains", len(u.Domains))

	s.Apps = make(map[string]bool, len(u.Apps))
	for _, app := range u.Apps {
		s.Apps[app] = true
	}
	glog.Infof("Loaded %d apps", len(u.Apps))

	s.Accounts = make(map[string]bool, len(u.Accounts))
	for _, Account := range u.Accounts {
		s.Accounts[Account] = true
	}
	glog.Infof("Loaded %d accounts", len(u.Accounts))

	return &Cache{
		shared:   s,
		accounts: &accountService{s},
		domains:  &domainService{s},
		apps:     &appsService{s},
		config:   &configService{s},
	}, nil
}

// Close does nothing
// TODO: close the file
func (c *Cache) Close() error {
	return nil
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
	shared *shared
}

// Get will return Account from memory if it exists
func (s *accountService) Get(id string) (*cache.Account, error) {
	if _, ok := s.shared.Accounts[id]; !ok {
		return nil, fmt.Errorf("Not found")
	}
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
	shared *shared
}

// Get will return back the domain if it exists in memory
func (s *domainService) Get(id string) (*cache.Domain, error) {
	if _, ok := s.shared.Domains[id]; !ok {
		return nil, fmt.Errorf("Not found")
	}
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
	shared *shared
}

// Get will return the App if it exists
func (s *appsService) Get(id string) (*cache.App, error) {
	if _, ok := s.shared.Apps[id]; !ok {
		return nil, fmt.Errorf("Not found")
	}
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
	shared *shared
}

// Get will return config from memory if it exists
func (s *configService) Get(id string) (string, error) {
	cfg, ok := s.shared.Configs[id]
	if !ok {
		return "", fmt.Errorf("Not found")
	}
	return cfg, nil
}

// Set not supported, always returns an error
func (s *configService) Set(id, value string) error {
	return fmt.Errorf("Not supported")
}
