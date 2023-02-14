package db_provider

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"errors"
	"io/ioutil"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/prebid/prebid-server/config"
)

const customTLSKey = "prebid-tls"

type MySqlDbProvider struct {
	cfg config.DatabaseConnection
	db  *sql.DB
}

func (provider *MySqlDbProvider) Config() config.DatabaseConnection {
	return provider.cfg
}

func (provider *MySqlDbProvider) Open() error {
	connStr, err := provider.ConnString()
	if err != nil {
		return err
	}

	db, err := sql.Open(provider.cfg.Driver, connStr)
	if err != nil {
		return err
	}

	provider.db = db
	return nil
}

func (provider *MySqlDbProvider) Close() error {
	if provider.db != nil {
		db := provider.db
		provider.db = nil
		return db.Close()
	}

	return nil
}

func (provider *MySqlDbProvider) Ping() error {
	return provider.db.Ping()
}

func (provider *MySqlDbProvider) ConnString() (string, error) {
	buffer := bytes.NewBuffer(nil)

	if provider.cfg.Username != "" {
		buffer.WriteString(provider.cfg.Username)
		if provider.cfg.Password != "" {
			buffer.WriteString(":")
			buffer.WriteString(provider.cfg.Password)
		}
		buffer.WriteString("@")
	}

	buffer.WriteString("tcp(")
	if provider.cfg.Host != "" {
		buffer.WriteString(provider.cfg.Host)
	}

	if provider.cfg.Port > 0 {
		buffer.WriteString(":")
		buffer.WriteString(strconv.Itoa(provider.cfg.Port))
	}
	buffer.WriteString(")")

	buffer.WriteString("/")

	if provider.cfg.Database != "" {
		buffer.WriteString(provider.cfg.Database)
	}

	// TLS connection
	var tls = "false"

	if provider.cfg.TLS.RootCert != "" {
		tls = "true"

		if err := setupTLSConfig(provider); err != nil {
			return "", err
		}

		if provider.cfg.TLS.ClientCert != "" && provider.cfg.TLS.ClientKey != "" {
			tls = customTLSKey
		}
	}

	if !strings.Contains(provider.cfg.QueryString, "tls=") {
		buffer.WriteString("?tls=")
		buffer.WriteString(tls)
		// Rest of connection string parameters passed through query_string
		if provider.cfg.QueryString != "" {
			buffer.WriteString("&")
			buffer.WriteString(provider.cfg.QueryString)
		}
	}

	// Rest of connection string parameters passed through query_string
	if provider.cfg.QueryString != "" {
		buffer.WriteString("?")
		buffer.WriteString(provider.cfg.QueryString)
	}

	return buffer.String(), nil
}

func setupTLSConfig(provider *MySqlDbProvider) error {
	rootCertPool := x509.NewCertPool()

	pem, err := ioutil.ReadFile(provider.cfg.TLS.RootCert)
	if err != nil {
		return err
	}

	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		return err
	}

	var clientCert []tls.Certificate
	if provider.cfg.TLS.ClientCert != "" && provider.cfg.TLS.ClientKey != "" {
		clientCert = make([]tls.Certificate, 0, 1)
		certs, err := tls.LoadX509KeyPair(provider.cfg.TLS.ClientCert, provider.cfg.TLS.ClientKey)
		if err != nil {
			return err
		}

		clientCert = append(clientCert, certs)
	}

	mysql.RegisterTLSConfig(customTLSKey, &tls.Config{
		RootCAs:               rootCertPool,
		Certificates:          clientCert,
		InsecureSkipVerify:    true,
		VerifyPeerCertificate: verifyPeerCertFunc(rootCertPool),
	})

	return nil
}

// verifyPeerCertFunc returns a function that verifies the peer certificate is
// in the cert pool.
func verifyPeerCertFunc(pool *x509.CertPool) func([][]byte, [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return errors.New("no certificates available to verify")
		}

		cert, err := x509.ParseCertificate(rawCerts[0])
		if err != nil {
			return err
		}

		opts := x509.VerifyOptions{Roots: pool}
		if _, err = cert.Verify(opts); err != nil {
			return err
		}
		return nil
	}
}

func (provider *MySqlDbProvider) PrepareQuery(template string, params ...QueryParam) (query string, args []interface{}) {
	query = template
	args = []interface{}{}

	type occurrence struct {
		startIndex int
		param      QueryParam
	}
	occurrences := []occurrence{}

	for _, param := range params {
		re := regexp.MustCompile("\\$" + param.Name)
		matches := re.FindAllIndex([]byte(query), -1)
		for _, match := range matches {
			occurrences = append(occurrences,
				occurrence{
					startIndex: match[0],
					param:      param,
				})
		}
	}
	sort.Slice(occurrences, func(i, j int) bool {
		return occurrences[i].startIndex < occurrences[j].startIndex
	})

	for _, occurrence := range occurrences {
		switch occurrence.param.Value.(type) {
		case []interface{}:
			idList := occurrence.param.Value.([]interface{})
			args = append(args, idList...)
		default:
			args = append(args, occurrence.param.Value)
		}
	}

	for _, param := range params {
		switch param.Value.(type) {
		case []interface{}:
			len := len(param.Value.([]interface{}))
			idList := provider.createIdList(len)
			query = strings.Replace(query, "$"+param.Name, idList, -1)
		default:
			query = strings.Replace(query, "$"+param.Name, "?", -1)
		}
	}
	return
}

func (provider *MySqlDbProvider) QueryContext(ctx context.Context, template string, params ...QueryParam) (*sql.Rows, error) {
	query, args := provider.PrepareQuery(template, params...)
	return provider.db.QueryContext(ctx, query, args...)
}

func (provider *MySqlDbProvider) createIdList(numArgs int) string {
	// Any empty list like "()" is illegal in MySql. A (NULL) is the next best thing,
	// though, since `id IN (NULL)` is valid for all "id" column types, and evaluates to an empty set.
	if numArgs == 0 {
		return "(NULL)"
	}

	result := bytes.NewBuffer(make([]byte, 0, 2+3*numArgs))
	result.WriteString("(")
	for i := 1; i < numArgs; i++ {
		result.WriteString("?")
		result.WriteString(", ")
	}
	result.WriteString("?")
	result.WriteString(")")

	return result.String()
}
