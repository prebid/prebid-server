package db_provider

import (
	"bytes"
	"context"
	"database/sql"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/prebid/prebid-server/config"
)

type MySqlDbProvider struct {
	cfg config.DatabaseConnection
	db  *sql.DB
}

func (provider *MySqlDbProvider) Config() config.DatabaseConnection {
	return provider.cfg
}

func (provider *MySqlDbProvider) Open(cfg config.DatabaseConnection) error {
	db, err := sql.Open(cfg.Driver, provider.ConnString(cfg))

	if err != nil {
		return err
	}

	provider.cfg = cfg
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

func (provider *MySqlDbProvider) ConnString(cfg config.DatabaseConnection) string {
	buffer := bytes.NewBuffer(nil)

	if cfg.Username != "" {
		buffer.WriteString(cfg.Username)
		if cfg.Password != "" {
			buffer.WriteString(":")
			buffer.WriteString(cfg.Password)
		}
		buffer.WriteString("@")
	}

	buffer.WriteString("tcp(")
	if cfg.Host != "" {
		buffer.WriteString(cfg.Host)
	}

	if cfg.Port > 0 {
		buffer.WriteString(":")
		buffer.WriteString(strconv.Itoa(cfg.Port))
	}
	buffer.WriteString(")")

	buffer.WriteString("/")

	if cfg.Database != "" {
		buffer.WriteString(cfg.Database)
	}

	return buffer.String()
}

func (provider *MySqlDbProvider) PrepareQuery(template string, params ...QueryParam) (query string, args []interface{}) {
	query = template
	args = []interface{}{}

	type occurrence struct {
		startIndex int
		paramIndex int
		paramKind  reflect.Kind
	}
	occurrences := []occurrence{}

	for index, param := range params {
		re := regexp.MustCompile("\\$" + param.Name)
		matches := re.FindAllIndex([]byte(query), -1)
		for _, match := range matches {
			occurrences = append(occurrences,
				occurrence{
					startIndex: match[0],
					paramIndex: index,
					paramKind:  reflect.TypeOf(param.Value).Kind(),
				})
		}
	}
	sort.Slice(occurrences, func(i, j int) bool {
		return occurrences[i].startIndex < occurrences[j].startIndex
	})

	for _, occurrence := range occurrences {
		if occurrence.paramKind == reflect.Slice {
			idList := params[occurrence.paramIndex].Value.([]interface{})
			args = append(args, idList...)
		} else {
			args = append(args, params[occurrence.paramIndex].Value)
		}
	}

	for _, param := range params {
		if reflect.TypeOf(param.Value).Kind() == reflect.Slice {
			len := len(param.Value.([]interface{}))
			idList := provider.createIdList(len)
			query = strings.Replace(query, "$"+param.Name, idList, -1)
		} else {
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
