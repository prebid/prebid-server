package db_fetcher

import (
	"bytes"
	"database/sql"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests"
	"strconv"
)

func NewPostgres(cfg *config.PostgresConfig) (stored_requests.Fetcher, error) {
	db, err := sql.Open("postgres", confToPostgresDSN(cfg))
	if err != nil {
		return nil, err
	}

	return &dbFetcher{
		db:         db,
		queryMaker: cfg.MakeQuery,
	}, nil
}

// confToPostgresDSN converts our app config into a string for the pq driver.
// For their docs, and the intended behavior of this function, see:  https://godoc.org/github.com/lib/pq
func confToPostgresDSN(cfg *config.PostgresConfig) string {
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
