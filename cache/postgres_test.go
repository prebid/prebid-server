package cache

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPostgresConfig(t *testing.T) {
	conf := PostgresDataCacheConfig{
		Host:     "host",
		Port:     1234,
		Dbname:   "dbname",
		User:     "user",
		Password: "password",
		TTL:      3434,
		Size:     100,
	}

	u := conf.uri()
	assert.True(t, strings.Contains(u, "host=host"))
	assert.True(t, strings.Contains(u, "port=1234"))
	assert.True(t, strings.Contains(u, "dbname=dbname"))
	assert.True(t, strings.Contains(u, "user=user"))
	assert.True(t, strings.Contains(u, "password=password"))
}
