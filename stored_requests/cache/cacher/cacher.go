package cacher

import (
	"errors"
)

// Cacher is the cacher interface for the db_fetchers
type Cacher interface {
	// Configure opens a new connection to the cache database.
	// This should be called once
	Configure(settings interface{}) error

	// Name of the cache client
	Name() string

	// Ping cache database
	Ping() error

	// Close connection to cache database
	Close()

	// Get a string based off of a key
	Get(key string) (string, error)

	// Set/Update a value with TTL in seconds
	Set(key, value string, ttl uint) error
}

var (
	// ErrDoesNotExist
	ErrDoesNotExist = errors.New("cacher: this key does not exist.")
	// DefaultTTL
	// TODO: what default TTL should we use?
	DefaultTTL = uint(60 * 60 * 24)
)

// Settings is used to configure your cache backend.
type Settings struct {
	Address  string
	Username string
	Password string
	Port     int
	Database string
	Size     int64
}

var (
	// clients is a global map of registered cache clients
	clients = make(map[string]Cacher)
)

const (
	// RedisCache
	RedisCache = "rediscache"
	// InMemoryCache
	InMemoryCache = "inmemorycache"
)

// Register should be called only by the cache drivers during init()
func Register(c Cacher) {
	if clients == nil {
		panic("cacher: something went wrong")
	}
	clients[c.Name()] = c
}

// Get returns back a cache driver
// it will panic if the cache driver has not been registered
func Get(name string) (c Cacher) {
	if clients[name] == nil {
		panic("cacher: no cache client registered for  " + name)
	}
	return clients[name]
}
