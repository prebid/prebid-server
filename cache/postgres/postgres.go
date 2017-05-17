package postgres

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"fmt"

	"github.com/coocood/freecache"
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/cache"
)

type PostgresDataCacheConfig struct {
	Host     string
	Port     int
	Dbname   string
	User     string
	Password string
	TTL      int
	Size     int
}

func (c *PostgresDataCacheConfig) uri() string {
	uri := ""
	if c.Host != "" {
		uri += fmt.Sprintf("host=%s ", c.Host)
	}

	if c.Port > 0 {
		uri += fmt.Sprintf("port=%d ", c.Port)
	}

	if c.User != "" {
		uri += fmt.Sprintf("user=%s ", c.User)
	}

	if c.Password != "" {
		uri += fmt.Sprintf("password=%s ", c.Password)
	}

	if c.Dbname != "" {
		uri += fmt.Sprintf("dbname=%s ", c.Dbname)
	}

	return uri
}

// shared configuration that get used by all of the services
type shared struct {
	db         *sql.DB
	lru        *freecache.Cache
	ttlSeconds int
}

func newShared(conf *PostgresDataCacheConfig) (*shared, error) {
	db, err := sql.Open("postgres", conf.uri()+" sslmode=disable")
	if err != nil {
		return nil, err
	}

	s := &shared{
		db:         db,
		lru:        freecache.NewCache(conf.Size),
		ttlSeconds: conf.TTL,
	}

	if err := s.db.Ping(); err != nil {
		/* This is for information only; we'll still operate w/o db */
		glog.Errorf("failed to connect to db store: %v", err)
	}

	return s, nil
}

// Cache postgres
type Cache struct {
	shared *shared

	Accounts *AccountService
	Domains  *DomainService
	Apps     *AppsService
	Config   *ConfigService
}

// New creates new postgres.Cache
func New() (*Cache, error) {

	shared, err := newShared(nil)
	if err != nil {
		return nil, err
	}
	return &Cache{
		shared:   shared,
		Accounts: &AccountService{shared: shared},
		Domains:  &DomainService{shared: shared},
		Apps:     &AppsService{shared: shared},
		Config:   &ConfigService{shared: shared},
	}, nil
}

func (c *Cache) Configure(cfg *cache.Configuration) error {
	return nil
}

func (c *Cache) Close() error {
	return c.shared.db.Close()
}

// AccountService handles the account information
type AccountService struct {
	shared *shared
}

// Get echos back the account
func (s *AccountService) Get(key string) (*cache.Account, error) {

	var account cache.Account

	b, err := s.shared.lru.Get([]byte(key))
	if err == nil {
		return decodeAccount(b), nil
	}

	var id string
	if err := s.shared.db.QueryRow("SELECT uuid FROM accounts_account where uuid = $1 LIMIT 1", key).Scan(&id); err != nil {
		/* TODO -- We should store failed attempts in the LRU as well to stop from hitting to DB */
		return nil, err
	}

	account.ID = id

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
func (s *AccountService) Set(account *cache.Account) error {
	return nil
}

// DomainService handles the domain information
type DomainService struct {
	shared *shared
}

func (s *DomainService) Get(key string) (*cache.Domain, error) {
	var domain string
	var d cache.Domain

	b, err := s.shared.lru.Get([]byte(key))
	if err == nil {
		buf := bytes.NewReader(b)
		if err = gob.NewDecoder(buf).Decode(&d); err != nil {
			panic(err)
		}
		return &d, nil
	}

	if err := s.shared.db.QueryRow("SELECT domain FROM domains_domain where domain = $1 LIMIT 1", key).Scan(&domain); err != nil {
		/* TODO -- We should store failed attempts in the LRU as well to stop from hitting to DB */
		return nil, err
	}

	d.Domain = domain

	buf := bytes.Buffer{}
	if err := gob.NewEncoder(&buf).Encode(&d); err != nil {
		panic(err)
	}

	s.shared.lru.Set([]byte(key), buf.Bytes(), s.shared.ttlSeconds)
	return &d, nil
}

// AppsService handles apps information
type AppsService struct {
	shared *shared
}

func (s *AppsService) Get(key string) (*cache.App, error) {
	var bundle string
	var app cache.App

	b, err := s.shared.lru.Get([]byte(key))
	if err == nil {
		buf := bytes.NewReader(b)
		if err = gob.NewDecoder(buf).Decode(&app); err != nil {
			panic(err)
		}
		return &app, nil
	}

	if err := s.shared.db.QueryRow("SELECT bundle FROM mobile_bundle where bundle = $1 LIMIT 1", key).Scan(&bundle); err != nil {
		/* TODO -- We should store failed attempts in the LRU as well to stop from hitting to DB */
		return nil, err
	}

	app.Bundle = bundle

	buf := bytes.Buffer{}
	if err := gob.NewEncoder(&buf).Encode(&app); err != nil {
		panic(err)
	}

	s.shared.lru.Set([]byte(key), buf.Bytes(), s.shared.ttlSeconds)
	return &app, nil
}

// ConfigService
type ConfigService struct {
	shared *shared
}

func (s *ConfigService) Get(key string) (string, error) {
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
