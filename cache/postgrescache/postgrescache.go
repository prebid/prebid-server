package postgrescache

import (
	"bytes"
	"database/sql"
	"encoding/gob"

	"github.com/coocood/freecache"
	"github.com/prebid/prebid-server/cache"
)

type CacheConfig struct {
	TTL  int
	Size int
}

// shared configuration that get used by all of the services
type shared struct {
	db         *sql.DB
	lru        *freecache.Cache
	ttlSeconds int
}

// Cache postgres
type Cache struct {
	shared   *shared
	accounts *accountService
	config   *configService
}

// New creates new postgres.Cache
func New(db *sql.DB, cfg CacheConfig) *Cache {
	shared := &shared{
		db:         db,
		lru:        freecache.NewCache(cfg.Size),
		ttlSeconds: cfg.TTL,
	}
	return &Cache{
		shared:   shared,
		accounts: &accountService{shared: shared},
		config:   &configService{shared: shared},
	}
}

func (c *Cache) Accounts() cache.AccountsService {
	return c.accounts
}
func (c *Cache) Config() cache.ConfigService {
	return c.config
}

func (c *Cache) Close() error {
	return c.shared.db.Close()
}

// AccountService handles the account information
type accountService struct {
	shared *shared
}

// Get echos back the account
func (s *accountService) Get(key string) (*cache.Account, error) {
	var account cache.Account

	b, err := s.shared.lru.Get([]byte(key))
	if err == nil {
		return decodeAccount(b), nil
	}

	var id string
	var priceGranularity sql.NullString
	if err := s.shared.db.QueryRow("SELECT uuid, price_granularity FROM accounts_account where uuid = $1 LIMIT 1", key).Scan(&id, &priceGranularity); err != nil {
		/* TODO -- We should store failed attempts in the LRU as well to stop from hitting to DB */
		return nil, err
	}

	account.ID = id
	if priceGranularity.Valid {
		account.PriceGranularity = priceGranularity.String
	}

	buf := bytes.Buffer{}
	if err := gob.NewEncoder(&buf).Encode(&account); err != nil {
		panic(err)
	}

	s.shared.lru.Set([]byte(key), buf.Bytes(), s.shared.ttlSeconds)
	return &account, nil
}

func decodeAccount(b []byte) *cache.Account {
	var account cache.Account
	buf := bytes.NewReader(b)
	if err := gob.NewDecoder(buf).Decode(&account); err != nil {
		panic(err)
	}
	return &account
}

// Set the account in postgres and the lru cache
func (s *accountService) Set(account *cache.Account) error {
	return nil
}

// ConfigService
type configService struct {
	shared *shared
}

func (s *configService) Set(id, value string) error {
	return nil
}

func (s *configService) Get(key string) (string, error) {
	if b, err := s.shared.lru.Get([]byte(key)); err == nil {
		return string(b), nil
	}
	var config string
	if err := s.shared.db.QueryRow("SELECT config FROM s2sconfig_config where uuid = $1 LIMIT 1", key).Scan(&config); err != nil {
		/* TODO -- We should store failed attempts in the LRU as well to stop from hitting to DB */
		return "", err
	}
	s.shared.lru.Set([]byte(key), []byte(config), s.shared.ttlSeconds)
	return config, nil
}
