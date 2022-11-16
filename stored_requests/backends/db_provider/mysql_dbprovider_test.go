package db_provider

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/stretchr/testify/assert"
)

func TestConnStringMySql(t *testing.T) {
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
			connString: "tcp()/",
		},
		{
			params: Params{
				db: "TestDB",
			},
			connString: "tcp()/TestDB",
		},
		{
			params: Params{
				host: "example.com",
			},
			connString: "tcp(example.com)/",
		},
		{
			params: Params{
				port: 20,
			},
			connString: "tcp(:20)/",
		},
		{
			params: Params{
				username: "someuser",
			},
			connString: "someuser@tcp()/",
		},
		{
			params: Params{
				username: "someuser",
				password: "somepassword",
			},
			connString: "someuser:somepassword@tcp()/",
		},
		{
			params: Params{
				db:       "TestDB",
				host:     "example.com",
				port:     20,
				username: "someuser",
				password: "somepassword",
			},
			connString: "someuser:somepassword@tcp(example.com:20)/TestDB",
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

		provider := MySqlDbProvider{
			cfg: cfg,
		}

		connString := provider.ConnString()
		assert.Equal(t, test.connString, connString, "Strings did not match")
	}
}
