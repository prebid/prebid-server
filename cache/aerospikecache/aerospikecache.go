package aerospikecache

import (
	"fmt"

	"github.com/aerospike/aerospike-client-go"
	"github.com/prebid/prebid-server/cache"
)

/*
  Aerospike Sets
  can be configured on init()
*/

var SetAccounts = "account"
var SetDomains = "domain"
var SetApps = "app"
var SetConfig = "config"

// Cache allows us to get and set data in Aerospike
type Cache struct {
	shared *shared

	accounts *accountService
	domains  *domainService
	apps     *appsService
	config   *configService
}

type shared struct {
	as  *aerospike.Client
	ns  string
	ttl int
}

func (s *shared) writePolicy() *aerospike.WritePolicy {
	return aerospike.NewWritePolicy(0, uint32(s.ttl))
}

func (s *shared) asKey(set string, id interface{}) (*aerospike.Key, error) {
	return aerospike.NewKey(s.ns, set, id)
}

// Configuration information for connecting to an Aerospike cluster
type Configuration struct {
	Hosts     []string `json:"hosts"`
	Namespace string   `json:"namespace"`
	TTL       int      `json:"ttl"`
}

// DefaultTTL will never expire for Aerospike 2 server versions >= 2.7.2 and Aerospike 3 server.
var DefaultTTL = aerospike.TTLServerDefault

// DefaultConfig provides configuration for running Aerospike locally
func DefaultConfig() Configuration {
	return Configuration{
		Hosts:     []string{"localhost"},
		Namespace: "test",
		TTL:       DefaultTTL,
	}
}

// connectCluster will attempt to return an Aerospike client
func connectCluster(hostnames []string) (*aerospike.Client, error) {
	var hosts = make([]*aerospike.Host, len(hostnames))
	for i, h := range hostnames {
		hosts[i] = aerospike.NewHost(h, 3000)
	}
	client, err := aerospike.NewClientWithPolicyAndHost(nil, hosts...)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func newShared(conf Configuration) (*shared, error) {
	client, err := connectCluster(conf.Hosts)
	if err != nil {
		return nil, err
	}
	return &shared{as: client, ns: conf.Namespace, ttl: conf.TTL}, nil
}

// New creates new Aerospike Cache Client
func New(cfg Configuration) (*Cache, error) {

	shared, err := newShared(cfg)
	if err != nil {
		return nil, err
	}
	return &Cache{
		shared:   shared,
		accounts: &accountService{shared: shared},
		domains:  &domainService{shared: shared},
		apps:     &appsService{shared: shared},
		config:   &configService{shared: shared},
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
	shared *shared
}

// Get echos back the account
func (s *accountService) Get(id string) (*cache.Account, error) {
	key, err := s.shared.asKey(SetAccounts, id)
	if err != nil {
		return nil, err
	}
	var account cache.Account
	if err := s.shared.as.GetObject(nil, key, &account); err != nil {
		return nil, err
	}
	return &account, nil
}

// Set uses GetObject (which uses reflect behind the scenes).
func (s *accountService) Set(account *cache.Account) error {
	key, err := s.shared.asKey(SetAccounts, account.ID)
	if err != nil {
		return err
	}
	if err := s.shared.as.PutObject(s.shared.writePolicy(), key, account); err != nil {
		return err
	}
	return nil
}

// DomainService handles the domain information
type domainService struct {
	shared *shared
}

// Set uses GetObject (which uses reflect behind the scenes).
func (s *domainService) Set(domain *cache.Domain) error {
	key, err := s.shared.asKey(SetDomains, domain.Domain)
	if err != nil {
		return err
	}
	if err := s.shared.as.PutObject(s.shared.writePolicy(), key, domain); err != nil {
		return err
	}
	return nil
}

func (s *domainService) Get(id string) (*cache.Domain, error) {
	key, err := s.shared.asKey(SetDomains, id)
	if err != nil {
		return nil, err
	}
	var domain cache.Domain
	if err := s.shared.as.GetObject(nil, key, &domain); err != nil {
		return nil, err
	}
	return &domain, nil
}

// AppsService handles apps information
type appsService struct {
	shared *shared
}

// Set uses GetObject (which uses reflect behind the scenes).
func (s *appsService) Set(app *cache.App) error {
	key, err := s.shared.asKey(SetApps, app.Bundle)
	if err != nil {
		return err
	}
	if err := s.shared.as.PutObject(s.shared.writePolicy(), key, app); err != nil {
		return err
	}
	return nil
}

func (s *appsService) Get(id string) (*cache.App, error) {
	key, err := s.shared.asKey(SetApps, id)
	if err != nil {
		return nil, err
	}
	var app cache.App
	if err := s.shared.as.GetObject(nil, key, &app); err != nil {
		return nil, err
	}
	return &app, nil
}

// ConfigService
type configService struct {
	shared *shared
}

func (s *configService) Set(id, value string) error {
	key, err := s.shared.asKey(SetConfig, id)
	if err != nil {
		return err
	}
	if err := s.shared.as.PutBins(s.shared.writePolicy(), key, aerospike.NewBin("config", value)); err != nil {
		return err
	}
	return nil
}

func (s *configService) Get(id string) (string, error) {
	key, err := s.shared.asKey(SetConfig, id)
	if err != nil {
		return "", err
	}
	rec, err := s.shared.as.Get(nil, key, "config")
	if err != nil {
		return "", err
	}
	value, ok := rec.Bins["config"].(string)
	if !ok {
		return "", fmt.Errorf("could not parse %v as string", rec.Bins["config"])
	}
	return value, nil
}
