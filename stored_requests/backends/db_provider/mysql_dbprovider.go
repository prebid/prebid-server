package db_provider

import (
	"bytes"
	"context"
	"database/sql"
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

func (provider *MySqlDbProvider) Open() error {
	db, err := sql.Open(provider.cfg.Driver, provider.ConnString())

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

func (provider *MySqlDbProvider) ConnString() string {
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

	return buffer.String()
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
