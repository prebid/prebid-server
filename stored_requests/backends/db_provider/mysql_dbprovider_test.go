package db_provider

import (
	"fmt"
	"path"
	"runtime"
	"testing"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/stretchr/testify/assert"
)

func TestConnStringMySql(t *testing.T) {

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

	_, callerFilename, _, _ := runtime.Caller(0)
	workingDir := path.Dir(callerFilename)

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
				username: "someuser",
				password: "somepassword:/?#[]@!$&()*+,;=",
			},
			connString: "someuser:somepassword:/?#[]@!$&()*+,;=@tcp()/",
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
		{
			params: Params{
				db:          "TestDB",
				host:        "example.com",
				port:        20,
				username:    "someuser",
				password:    "somepassword",
				QueryString: "param=value",
			},
			connString: "someuser:somepassword@tcp(example.com:20)/TestDB?param=value",
		},
		{
			params: Params{
				db:          "TestDB",
				host:        "example.com",
				port:        20,
				username:    "someuser",
				password:    "somepassword",
				QueryString: "param=value&tls=preferred",
			},
			connString: "someuser:somepassword@tcp(example.com:20)/TestDB?param=value&tls=preferred",
		},
		{
			params: Params{
				db:       "TestDB",
				host:     "example.com",
				port:     20,
				username: "someuser",
				password: "somepassword",
				TLS: TLS{
					RootCert: path.Join(workingDir, "test_assets/root-cert.pem"),
				},
			},
			connString: fmt.Sprintf("someuser:somepassword@tcp(example.com:20)/TestDB?tls=%s", customTLSKey),
		},
		{
			params: Params{
				db:          "TestDB",
				host:        "example.com",
				port:        20,
				username:    "someuser",
				password:    "somepassword",
				QueryString: "tls=tlsKeyByQueryString",
				TLS: TLS{
					RootCert: path.Join(workingDir, "test_assets/root-cert.pem"),
				},
			},
			connString: "someuser:somepassword@tcp(example.com:20)/TestDB?tls=tlsKeyByQueryString",
		},
		{
			params: Params{
				db:       "TestDB",
				host:     "example.com",
				port:     20,
				username: "someuser",
				password: "somepassword",
				TLS: TLS{
					RootCert:   path.Join(workingDir, "test_assets/root-cert.pem"),
					ClientCert: path.Join(workingDir, "test_assets/client-cert.pem"),
					ClientKey:  path.Join(workingDir, "test_assets/client-key.pem"),
				},
			},
			connString: fmt.Sprintf("someuser:somepassword@tcp(example.com:20)/TestDB?tls=%s", customTLSKey),
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
					RootCert:   path.Join(workingDir, "test_assets/root-cert.pem"),
					ClientCert: path.Join(workingDir, "test_assets/client-cert.pem"),
					ClientKey:  path.Join(workingDir, "test_assets/client-key.pem"),
				},
			},
			connString: fmt.Sprintf("someuser:somepassword@tcp(example.com:20)/TestDB?tls=%s&param=value", customTLSKey),
		},
		{
			params: Params{
				db:          "TestDB",
				host:        "example.com",
				port:        20,
				username:    "someuser",
				password:    "somepassword",
				QueryString: "tls=preferred",
				TLS: TLS{
					RootCert:   path.Join(workingDir, "test_assets/root-cert.pem"),
					ClientCert: path.Join(workingDir, "test_assets/client-cert.pem"),
					ClientKey:  path.Join(workingDir, "test_assets/client-key.pem"),
				},
			},
			connString: "someuser:somepassword@tcp(example.com:20)/TestDB?tls=preferred",
		},
		{
			params: Params{
				db:          "TestDB",
				host:        "example.com",
				port:        20,
				username:    "someuser",
				password:    "somepassword",
				QueryString: "param=value&tls=preferred",
				TLS: TLS{
					RootCert:   path.Join(workingDir, "test_assets/root-cert.pem"),
					ClientCert: path.Join(workingDir, "test_assets/client-cert.pem"),
					ClientKey:  path.Join(workingDir, "test_assets/client-key.pem"),
				},
			},
			connString: "someuser:somepassword@tcp(example.com:20)/TestDB?param=value&tls=preferred",
		},
		{
			params: Params{
				db:          "TestDB",
				host:        "example.com",
				port:        20,
				username:    "someuser",
				password:    "somepassword",
				QueryString: "param=value&tls=tlsKeyByQueryString",
				TLS: TLS{
					RootCert:   path.Join(workingDir, "test_assets/root-cert.pem"),
					ClientCert: path.Join(workingDir, "test_assets/client-cert.pem"),
					ClientKey:  path.Join(workingDir, "test_assets/client-key.pem"),
				},
			},
			connString: "someuser:somepassword@tcp(example.com:20)/TestDB?param=value&tls=tlsKeyByQueryString",
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

		provider := MySqlDbProvider{
			cfg: cfg,
		}

		connString, _ := provider.ConnString()
		assert.Equal(t, test.connString, connString, "Strings did not match")
	}
}
