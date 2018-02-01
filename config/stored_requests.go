package config

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// StoredRequests configures the backend used to store requests on the server.
type StoredRequests struct {
	// Files should be true if Stored Requests should be loaded from the filesystem.
	Files bool `mapstructure:"filesystem"`
	// Postgres should be non-nil if Stored Requests should be loaded from a Postgres database.
	Postgres *PostgresConfig `mapstructure:"postgres"`
	// Cache should be non-nil if an in-memory cache should be used to store Stored Requests locally.
	InMemoryCache *InMemoryCache `mapstructure:"in_memory_cache"`
}

func (cfg *StoredRequests) validate() error {
	if cfg.Files && cfg.Postgres != nil {
		return errors.New("stored request backend is ambiguous. If stored_requests.postgres is defined, then stored_requests.filesystem must be false")
	}

	return nil
}

// PostgresConfig configures the Postgres connection for Stored Requests
type PostgresConfig struct {
	Database string `mapstructure:"dbname"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"user"`
	Password string `mapstructure:"password"`

	// QueryTemplate is the Postgres Query which can be used to fetch configs from the database.
	// It is a Template, rather than a full Query, because a single HTTP request may reference multiple Stored Requests.
	//
	// In the simplest case, this could be something like:
	//   SELECT id, requestData FROM stored_requests WHERE id in %ID_LIST%
	//
	// The MakeQuery function will transform this query into:
	//   SELECT id, requestData FROM stored_requests WHERE id in ($1, $2, $3, ...)
	//
	// ... where the number of "$x" args depends on how many IDs are nested within the HTTP request.
	QueryTemplate string `mapstructure:"query"`
	// Hacky way to add Amp queries that won't fit in the global query above. A better solution would
	// rename all the queries to be specific for what they are for. But that can wait for a general
	// stored requests overhaul.
	AmpQueryTemplate string `mapstructure:"amp_query"`
}

// MakeQuery gets a stored-request-fetching query which can be used to fetch numRequests requests at once.
func (cfg *PostgresConfig) MakeQuery(numRequests int) (string, error) {
	if numRequests < 1 {
		return "", fmt.Errorf("can't generate query to fetch %d stored requests", numRequests)
	}
	return strings.Replace(cfg.QueryTemplate, "%ID_LIST%", makeIdList(numRequests), -1), nil
}

// MakeAmpQuery gets a stored-request-fetching query which can be used to fetch numRequests requests at once.
// A hacky solution for AMP, but a good solution would require an overhaul of the stored request system to
// pass around the ID type. This should probably be its own PR.
func (cfg *PostgresConfig) MakeAmpQuery(numRequests int) (string, error) {
	if numRequests < 1 {
		return "", fmt.Errorf("can't generate query to fetch %d stored requests", numRequests)
	}
	return strings.Replace(cfg.AmpQueryTemplate, "%ID_LIST%", makeIdList(numRequests), -1), nil
}

func makeIdList(numRequests int) string {
	final := bytes.NewBuffer(make([]byte, 0, 2+4*numRequests))
	final.WriteString("(")
	for i := 1; i < numRequests; i++ {
		final.WriteString("$")
		final.WriteString(strconv.Itoa(i))
		final.WriteString(", ")
	}
	final.WriteString("$")
	final.WriteString(strconv.Itoa(numRequests))
	final.WriteString(")")

	return final.String()
}

type InMemoryCache struct {
	// TTL is the maximum number of seconds that an unused value will stay in the cache.
	// TTL <= 0 can be used for "no ttl". Elements will still be evicted based on the Size.
	TTL int `mapstructure:"ttl_seconds"`
	// Size is the max number of bytes allowed in the cache.
	Size int `mapstructure:"size_bytes"`
}
