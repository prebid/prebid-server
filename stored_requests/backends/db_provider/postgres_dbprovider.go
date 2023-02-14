package db_provider

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/prebid/prebid-server/config"
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

	if provider.cfg.Host != "" {
		buffer.WriteString("host=")
		buffer.WriteString(provider.cfg.Host)
		buffer.WriteString(" ")
	}

	if provider.cfg.Port > 0 {
		buffer.WriteString("port=")
		buffer.WriteString(strconv.Itoa(provider.cfg.Port))
		buffer.WriteString(" ")
	}

	if provider.cfg.Username != "" {
		buffer.WriteString("user=")
		buffer.WriteString(provider.cfg.Username)
		buffer.WriteString(" ")
	}

	if provider.cfg.Password != "" {
		buffer.WriteString("password=")
		buffer.WriteString(provider.cfg.Password)
		buffer.WriteString(" ")
	}

	if provider.cfg.Database != "" {
		buffer.WriteString("dbname=")
		buffer.WriteString(provider.cfg.Database)
		buffer.WriteString(" ")
	}

	// TLS connection
	var sslmode = "disable"

	if provider.cfg.TLS.RootCert != "" {
		sslmode = "verify-ca"

		buffer.WriteString("sslrootcert=")
		buffer.WriteString(provider.cfg.TLS.RootCert)
		buffer.WriteString(" ")

		if provider.cfg.TLS.ClientCert != "" && provider.cfg.TLS.ClientKey != "" {
			sslmode = "verify-full"

			buffer.WriteString("sslcert=")
			buffer.WriteString(provider.cfg.TLS.ClientCert)
			buffer.WriteString(" ")

			buffer.WriteString("sslkey=")
			buffer.WriteString(provider.cfg.TLS.ClientKey)
			buffer.WriteString(" ")
		}
	}

	if !strings.Contains(provider.cfg.QueryString, "sslmode=") {
		buffer.WriteString("sslmode=")
		buffer.WriteString(sslmode)
	}

	// Rest of connection string parameters passed through query_string
	if provider.cfg.QueryString != "" {
		buffer.WriteString(" ")
		buffer.WriteString(provider.cfg.QueryString)
	}

	return buffer.String(), nil
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
