package db_provider

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/prebid/prebid-server/v3/config"
)

type PostgresDbProvider struct {
	cfg config.DatabaseConnection
	db  *sql.DB
}

func (provider *PostgresDbProvider) Config() config.DatabaseConnection {
	return provider.cfg
}

func (provider *PostgresDbProvider) Open() error {
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

func (provider *PostgresDbProvider) Close() error {
	if provider.db != nil {
		db := provider.db
		provider.db = nil
		return db.Close()
	}

	return nil
}

func (provider *PostgresDbProvider) Ping() error {
	return provider.db.Ping()
}

func (provider *PostgresDbProvider) ConnString() (string, error) {
	buffer := bytes.NewBuffer(nil)
	buffer.WriteString("postgresql://")

	if provider.cfg.Username != "" {
		buffer.WriteString(provider.cfg.Username)
		if provider.cfg.Password != "" {
			buffer.WriteString(":")
			buffer.WriteString(url.QueryEscape(provider.cfg.Password))
		}
		buffer.WriteString("@")
	}

	if provider.cfg.Host != "" {
		buffer.WriteString(provider.cfg.Host)
	}

	if provider.cfg.Port > 0 {
		buffer.WriteString(":")
		buffer.WriteString(strconv.Itoa(provider.cfg.Port))
	}

	if provider.cfg.Database != "" {
		buffer.WriteString("/")
		buffer.WriteString(provider.cfg.Database)
	}

	queryStr, err := provider.generateQueryString()
	if err != nil {
		return "", err
	}

	if queryStr != "" {
		buffer.WriteString("?")
		buffer.WriteString(queryStr)
	}

	return buffer.String(), nil
}

func (provider *PostgresDbProvider) generateQueryString() (string, error) {
	isTlsInConfigStruct := provider.cfg.TLS.RootCert != "" ||
		provider.cfg.TLS.ClientCert != "" ||
		provider.cfg.TLS.ClientKey != ""

	isTlsInQueryString := strings.Contains(provider.cfg.QueryString, "sslrootcert=") ||
		strings.Contains(provider.cfg.QueryString, "sslcert=") ||
		strings.Contains(provider.cfg.QueryString, "sslkey=")

	if isTlsInConfigStruct && isTlsInQueryString {
		return "", errors.New("TLS cert information must either be specified in the TLS object or the query string but not both.")
	}

	sslmode := "disable"
	sslrootcert := ""
	sslcert := ""
	sslkey := ""
	queryString := ""

	if provider.cfg.TLS.RootCert != "" {
		sslmode = "verify-ca"
		sslrootcert = fmt.Sprintf("&sslrootcert=%s", provider.cfg.TLS.RootCert)

		if provider.cfg.TLS.ClientCert != "" && provider.cfg.TLS.ClientKey != "" {
			sslmode = "verify-full"
			sslcert = fmt.Sprintf("&sslcert=%s", provider.cfg.TLS.ClientCert)
			sslkey = fmt.Sprintf("&sslkey=%s", provider.cfg.TLS.ClientKey)
		}
	}
	sslmode = fmt.Sprintf("&sslmode=%s", sslmode)

	if len(provider.cfg.QueryString) != 0 {
		queryString = fmt.Sprintf("&%s", provider.cfg.QueryString)

		if strings.Contains(provider.cfg.QueryString, "sslmode=") {
			sslmode = ""
		}
	}

	params := strings.Join([]string{sslmode, sslrootcert, sslcert, sslkey, queryString}, "")
	return params[1:], nil
}

func (provider *PostgresDbProvider) PrepareQuery(template string, params ...QueryParam) (query string, args []interface{}) {
	query = template
	args = []interface{}{}

	for _, param := range params {
		switch v := param.Value.(type) {
		case []interface{}:
			idList := v
			idListStr := provider.createIdList(len(args), len(idList))
			args = append(args, idList...)
			query = strings.Replace(query, "$"+param.Name, idListStr, -1)
		default:
			args = append(args, param.Value)
			query = strings.Replace(query, "$"+param.Name, fmt.Sprintf("$%d", len(args)), -1)
		}
	}
	return
}

func (provider *PostgresDbProvider) QueryContext(ctx context.Context, template string, params ...QueryParam) (*sql.Rows, error) {
	query, args := provider.PrepareQuery(template, params...)
	return provider.db.QueryContext(ctx, query, args...)
}

func (provider *PostgresDbProvider) createIdList(numSoFar int, numArgs int) string {
	// Any empty list like "()" is illegal in Postgres. A (NULL) is the next best thing,
	// though, since `id IN (NULL)` is valid for all "id" column types, and evaluates to an empty set.
	//
	// The query plan also suggests that it's basically free:
	//
	// explain SELECT id, requestData FROM stored_requests WHERE id in $ID_LIST;
	//
	// QUERY PLAN
	// -------------------------------------------
	// Result  (cost=0.00..0.00 rows=0 width=16)
	//	 One-Time Filter: false
	// (2 rows)
	if numArgs == 0 {
		return "(NULL)"
	}

	final := bytes.NewBuffer(make([]byte, 0, 2+4*numArgs))
	final.WriteString("(")
	for i := numSoFar + 1; i < numSoFar+numArgs; i++ {
		final.WriteString("$")
		final.WriteString(strconv.Itoa(i))
		final.WriteString(", ")
	}
	final.WriteString("$")
	final.WriteString(strconv.Itoa(numSoFar + numArgs))
	final.WriteString(")")

	return final.String()
}
