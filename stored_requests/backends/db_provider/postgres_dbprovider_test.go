package db_provider

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/stretchr/testify/assert"
)

func TestConnStringPostgres(t *testing.T) {

	type TLS struct {
		RootCert   string
		ClientCert string
		ClientKey  string
	}

	type Params struct {
		db          string
		host        string
		port        int
		username    string
		password    string
		QueryString string
		TLS         TLS
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
			connString: "postgresql://?sslmode=disable",
		},
		{
			params: Params{
				db: "TestDB",
			},
			connString: "postgresql:///TestDB?sslmode=disable",
		},
		{
			params: Params{
				host: "example.com",
			},
			connString: "postgresql://example.com?sslmode=disable",
		},
		{
			params: Params{
				port: 20,
			},
			connString: "postgresql://:20?sslmode=disable",
		},
		{
			params: Params{
				username: "someuser",
			},
			connString: "postgresql://someuser@?sslmode=disable",
		},
		{
			params: Params{
				username: "someuser",
				password: "somepassword",
			},
			connString: "postgresql://someuser:somepassword@?sslmode=disable",
		},
		{
			params: Params{
				db:       "TestDB",
				host:     "example.com",
				port:     20,
				username: "someuser",
				password: "somepassword",
			},
			connString: "postgresql://someuser:somepassword@example.com:20/TestDB?sslmode=disable",
		},
		{
			params: Params{
				db:          "TestDB",
				host:        "example.com",
				port:        20,
				username:    "someuser",
				password:    "somepassword",
				QueryString: "param=value",
			},
			connString: "postgresql://someuser:somepassword@example.com:20/TestDB?sslmode=disable&param=value",
		},
		{
			params: Params{
				db:          "TestDB",
				host:        "example.com",
				port:        20,
				username:    "someuser",
				password:    "somepassword",
				QueryString: "param=value&sslmode=require",
			},
			connString: "postgresql://someuser:somepassword@example.com:20/TestDB?param=value&sslmode=require",
		},
		{
			params: Params{
				db:       "TestDB",
				host:     "example.com",
				port:     20,
				username: "someuser",
				password: "somepassword",
				TLS: TLS{
					RootCert: "root-cert.pem",
				},
			},
			connString: "postgresql://someuser:somepassword@example.com:20/TestDB?sslmode=verify-ca&sslrootcert=root-cert.pem",
		},
		{
			params: Params{
				db:       "TestDB",
				host:     "example.com",
				port:     20,
				username: "someuser",
				password: "somepassword",
				TLS: TLS{
					RootCert:   "root-cert.pem",
					ClientCert: "client-cert.pem",
					ClientKey:  "client-key.pem",
				},
			},
			connString: "postgresql://someuser:somepassword@example.com:20/TestDB?sslmode=verify-full&sslrootcert=root-cert.pem&sslcert=client-cert.pem&sslkey=client-key.pem",
		},
		{
			params: Params{
				db:          "TestDB",
				host:        "example.com",
				port:        20,
				username:    "someuser",
				password:    "somepassword",
				QueryString: "param=value",
				TLS: TLS{
					RootCert:   "root-cert.pem",
					ClientCert: "client-cert.pem",
					ClientKey:  "client-key.pem",
				},
			},
			connString: "postgresql://someuser:somepassword@example.com:20/TestDB?sslmode=verify-full&sslrootcert=root-cert.pem&sslcert=client-cert.pem&sslkey=client-key.pem&param=value",
		},
		{
			params: Params{
				db:          "TestDB",
				host:        "example.com",
				port:        20,
				username:    "someuser",
				password:    "somepassword",
				QueryString: "sslmode=prefer",
				TLS: TLS{
					RootCert:   "root-cert.pem",
					ClientCert: "client-cert.pem",
					ClientKey:  "client-key.pem",
				},
			},
			connString: "postgresql://someuser:somepassword@example.com:20/TestDB?sslrootcert=root-cert.pem&sslcert=client-cert.pem&sslkey=client-key.pem&sslmode=prefer",
		},
		{
			params: Params{
				db:          "TestDB",
				host:        "example.com",
				port:        20,
				username:    "someuser",
				password:    "somepassword",
				QueryString: "param=value&sslmode=prefer",
				TLS: TLS{
					RootCert:   "root-cert.pem",
					ClientCert: "client-cert.pem",
					ClientKey:  "client-key.pem",
				},
			},
			connString: "postgresql://someuser:somepassword@example.com:20/TestDB?sslrootcert=root-cert.pem&sslcert=client-cert.pem&sslkey=client-key.pem&param=value&sslmode=prefer",
		},
	}

	for _, test := range tests {
		cfg := config.DatabaseConnection{
			Database:    test.params.db,
			Host:        test.params.host,
			Port:        test.params.port,
			Username:    test.params.username,
			Password:    test.params.password,
			QueryString: test.params.QueryString,
			TLS:         config.TLS(test.params.TLS),
		}

		provider := PostgresDbProvider{
			cfg: cfg,
		}

		connString, _ := provider.ConnString()
		assert.Equal(t, test.connString, connString, "Strings did not match")
	}
}
