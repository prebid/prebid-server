package postgres

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"fmt"

	"github.com/prebid/prebid-server/cache"
	"github.com/coocood/freecache"
	"github.com/golang/glog"
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

type PostgresDataCache struct {
	db         *sql.DB
	lru        *freecache.Cache
	ttlSeconds int
}

func (c *PostgresDataCache) Close() {
	c.db.Close()
}

// New creates new PostgresDataCache
func New(conf *PostgresDataCacheConfig) (*PostgresDataCache, error) {

	db, err := sql.Open("postgres", conf.uri()+" sslmode=disable")
	if err != nil {
		return nil, err
	}

	c := &PostgresDataCache{
		db:         db,
		lru:        freecache.NewCache(conf.Size),
		ttlSeconds: conf.TTL,
	}

	if err := c.db.Ping(); err != nil {
		/* This is for information only; we'll still operate w/o db */
		glog.Errorf("failed to connect to db store: %v", err)
	}

	return c, nil
}

func (c *PostgresDataCache) GetConfig(key string) (string, error) {

	if b, err := c.lru.Get([]byte(key)); err == nil {
		return string(b), nil
	}

	var config string
	if err := c.db.QueryRow("SELECT config FROM s2sconfig_config where uuid = $1 LIMIT 1", key).Scan(&config); err != nil {
		/* TODO -- We should store failed attempts in the LRU as well to stop from hitting to DB */
		return "", err
	}

	c.lru.Set([]byte(key), []byte(config), c.ttlSeconds)
	return config, nil
}

func (c *PostgresDataCache) GetDomain(key string) (*cache.Domain, error) {
	var domain string
	var d cache.Domain

	b, err := c.lru.Get([]byte(key))
	if err == nil {
		buf := bytes.NewReader(b)
		if err = gob.NewDecoder(buf).Decode(&d); err != nil {
			panic(err)
		}
		return &d, nil
	}

	if err := c.db.QueryRow("SELECT domain FROM domains_domain where domain = $1 LIMIT 1", key).Scan(&domain); err != nil {
		/* TODO -- We should store failed attempts in the LRU as well to stop from hitting to DB */
		return nil, err
	}

	d.Domain = domain

	buf := bytes.Buffer{}
	if err := gob.NewEncoder(&buf).Encode(&d); err != nil {
		panic(err)
	}

	c.lru.Set([]byte(key), buf.Bytes(), c.ttlSeconds)
	return &d, nil
}

func (c *PostgresDataCache) GetApp(key string) (*cache.App, error) {
	var bundle string
	var app cache.App

	b, err := c.lru.Get([]byte(key))
	if err == nil {
		buf := bytes.NewReader(b)
		if err = gob.NewDecoder(buf).Decode(&app); err != nil {
			panic(err)
		}
		return &app, nil
	}

	if err := c.db.QueryRow("SELECT bundle FROM mobile_bundle where bundle = $1 LIMIT 1", key).Scan(&bundle); err != nil {
		/* TODO -- We should store failed attempts in the LRU as well to stop from hitting to DB */
		return nil, err
	}

	app.Bundle = bundle

	buf := bytes.Buffer{}
	if err := gob.NewEncoder(&buf).Encode(&app); err != nil {
		panic(err)
	}

	c.lru.Set([]byte(key), buf.Bytes(), c.ttlSeconds)
	return &app, nil
}

func (c *PostgresDataCache) GetAccount(key string) (*cache.Account, error) {
	var account cache.Account

	b, err := c.lru.Get([]byte(key))
	if err == nil {
		return decodeAccount(b), nil
	}

	var id string
	if err := c.db.QueryRow("SELECT uuid FROM accounts_account where uuid = $1 LIMIT 1", key).Scan(&id); err != nil {
		/* TODO -- We should store failed attempts in the LRU as well to stop from hitting to DB */
		return nil, err
	}

	account.ID = id

	buf := bytes.Buffer{}
	if err := gob.NewEncoder(&buf).Encode(&account); err != nil {
		panic(err)
	}

	c.lru.Set([]byte(key), buf.Bytes(), c.ttlSeconds)
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
