package db_provider

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"reflect"
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

func (provider *PostgresDbProvider) Open(cfg config.DatabaseConnection) error {
	db, err := sql.Open(cfg.Driver, provider.ConnString(cfg))

	if err != nil {
		return err
	}

	provider.cfg = cfg
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

func (provider *PostgresDbProvider) ConnString(cfg config.DatabaseConnection) string {
	buffer := bytes.NewBuffer(nil)

	if cfg.Host != "" {
		buffer.WriteString("host=")
		buffer.WriteString(cfg.Host)
		buffer.WriteString(" ")
	}

	if cfg.Port > 0 {
		buffer.WriteString("port=")
		buffer.WriteString(strconv.Itoa(cfg.Port))
		buffer.WriteString(" ")
	}

	if cfg.Username != "" {
		buffer.WriteString("user=")
		buffer.WriteString(cfg.Username)
		buffer.WriteString(" ")
	}

	if cfg.Password != "" {
		buffer.WriteString("password=")
		buffer.WriteString(cfg.Password)
		buffer.WriteString(" ")
	}

	if cfg.Database != "" {
		buffer.WriteString("dbname=")
		buffer.WriteString(cfg.Database)
		buffer.WriteString(" ")
	}

	buffer.WriteString("sslmode=disable")
	return buffer.String()
}

func (provider *PostgresDbProvider) PrepareQuery(template string, params ...QueryParam) (query string, args []interface{}) {
	query = template
	args = []interface{}{}

	type occurrence struct {
		startIndex int
		paramIndex int
		paramKind  reflect.Kind
	}

	for _, param := range params {
		if reflect.TypeOf(param.Value).Kind() == reflect.Slice {

			idList := param.Value.([]interface{})

			idListStr := provider.createIdList(len(args), len(idList))
			args = append(args, idList...)
			query = strings.Replace(query, "$"+param.Name, idListStr, -1)

		} else {
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
	// explain SELECT id, requestData FROM stored_requests WHERE id in %ID_LIST%;
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
