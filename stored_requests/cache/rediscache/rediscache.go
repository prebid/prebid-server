package rediscache

import (
	"errors"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/prebid/prebid-server/stored_requests/cache/cacher"
)

type client struct {
	client *redis.Client
}

func init() {
	var c = &client{}
	cacher.Register(c)
}

func (c *client) Name() string {
	return cacher.RedisCache
}

func (c *client) Ping() error {
	if _, err := c.client.Ping().Result(); err != nil {
		return err
	}
	return nil
}

func (c *client) Close() {
	c.client.Close()
}

// Configure will connect to Redis
func (c *client) Configure(settings *cacher.Settings) error {
	if c.client != nil {
		return errors.New("rediscache: we have already configured this client")
	}

	// use this size by default
	var addr = "localhost:6379"

	if settings != nil && settings.Address != "" {
		addr = settings.Address
	}

	var database = 0
	if settings != nil && settings.Database != "" {
		database, _ = strconv.Atoi(settings.Database)
	}

	var password string
	if settings != nil {
		password = settings.Password
	}

	c.client = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // no password set
		DB:       database, // use default DB
	})

	if _, err := c.client.Ping().Result(); err != nil {
		return err
	}
	return nil
}

func (c *client) Get(key string) (string, error) {
	value, err := c.client.Get(key).Result()
	if err == redis.Nil {
		// if nil then return back DoesNotExist
		return "", cacher.ErrDoesNotExist
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func (c *client) Set(key, value string, ttl uint) error {
	if err := c.client.Set(key, value, time.Duration(ttl)*time.Second).Err(); err != nil {
		return err
	}
	return nil
}
