package cache

import (
	"bytes"
	"database/sql"
	"encoding/binary"

	"fmt"

	"github.com/coocood/freecache"
	"github.com/golang/glog"
	_ "github.com/lib/pq"
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

func NewPostgresDataCache(conf *PostgresDataCacheConfig) (*PostgresDataCache, error) {

	db, err := sql.Open("postgres", conf.uri()+" sslmode=disable")
	if err != nil {
		return nil, err
	}

	c := &PostgresDataCache{
		db:         db,
		lru:        freecache.NewCache(conf.Size),
		ttlSeconds: conf.TTL,
	}

	if err = c.db.Ping(); err != nil {
		/* This is for information only; we'll still operate w/o db */
		glog.Errorf("failed to connect to db store: %v", err)
	}

	return c, nil
}

func (c *PostgresDataCache) GetConfig(key string) (string, error) {
	var config string

	b, err := c.lru.Get([]byte(key))
	if err == nil {
		return string(b), nil
	}

	err = c.db.QueryRow("SELECT config FROM s2sconfig_config where uuid = $1 LIMIT 1", key).Scan(&config)
	if err != nil {
		/* TODO -- We should store failed attempts in the LRU as well to stop from hitting to DB */
		return "", err
	}

	c.lru.Set([]byte(key), []byte(config), c.ttlSeconds)
	return config, nil
}

func (c *PostgresDataCache) GetDomain(key string) (*Domain, error) {
	var domain string
	d := Domain{}

	b, err := c.lru.Get([]byte(key))
	if err == nil {
		buf := bytes.NewReader(b)
		err = binary.Read(buf, binary.BigEndian, &d)
		if err != nil {
			panic(err)
		}
		return &d, nil
	}

	err = c.db.QueryRow("SELECT domain FROM domains_domain where domain = $1 LIMIT 1", key).Scan(&domain)
	if err != nil {
		/* TODO -- We should store failed attempts in the LRU as well to stop from hitting to DB */
		return nil, err
	}

	d.Domain = domain

	buf := &bytes.Buffer{}
	err = binary.Write(buf, binary.BigEndian, &d)
	if err != nil {
		panic(err)
	}

	c.lru.Set([]byte(key), buf.Bytes(), c.ttlSeconds)
	return &d, nil
}

func (c *PostgresDataCache) GetAccount(key string) (*Account, error) {
	var id string
	account := Account{}

	b, err := c.lru.Get([]byte(key))
	if err == nil {
		buf := bytes.NewReader(b)
		err = binary.Read(buf, binary.BigEndian, &account)
		if err != nil {
			panic(err)
		}
		return &account, nil
	}

	err = c.db.QueryRow("SELECT domain FROM domains_domain where domain = $1 LIMIT 1", key).Scan(&id)
	if err != nil {
		/* TODO -- We should store failed attempts in the LRU as well to stop from hitting to DB */
		return nil, err
	}

	account.ID = id

	buf := &bytes.Buffer{}
	err = binary.Write(buf, binary.BigEndian, &account)
	if err != nil {
		panic(err)
	}

	c.lru.Set([]byte(key), buf.Bytes(), c.ttlSeconds)
	return &account, nil
}
