package db_provider

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/stretchr/testify/assert"
)

func TestConnStringPostgres(t *testing.T) {
	type Params struct {
		db       string
		host     string
		port     int
		username string
		password string
	}

	tests := []struct {
		name       string
		params     Params
		connString string
	}{
		{
			params: Params{
				db: "",
			},
			connString: "sslmode=disable",
		},
		{
			params: Params{
				db: "TestDB",
			},
			connString: "dbname=TestDB sslmode=disable",
		},
		{
			params: Params{
				host: "example.com",
			},
			connString: "host=example.com sslmode=disable",
		},
		{
			params: Params{
				port: 20,
			},
			connString: "port=20 sslmode=disable",
		},
		{
			params: Params{
				username: "someuser",
			},
			connString: "user=someuser sslmode=disable",
		},
		{
			params: Params{
				username: "someuser",
				password: "somepassword",
			},
			connString: "user=someuser password=somepassword sslmode=disable",
		},
		{
			params: Params{
				db:       "TestDB",
				host:     "example.com",
				port:     20,
				username: "someuser",
				password: "somepassword",
			},
			connString: "host=example.com port=20 user=someuser password=somepassword dbname=TestDB sslmode=disable",
		},
	}

	for _, test := range tests {
		cfg := config.DatabaseConnection{
			Database: test.params.db,
			Host:     test.params.host,
			Port:     test.params.port,
			Username: test.params.username,
			Password: test.params.password,
		}

		provider := PostgresDbProvider{
			cfg: cfg,
		}

		connString := provider.ConnString()
		assert.Equal(t, test.connString, connString, "Strings did not match")
	}
}
