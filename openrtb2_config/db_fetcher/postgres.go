package db_fetcher

import (
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb2_config"
	"database/sql"
	"bytes"
	"strconv"
)

func NewPostgres(cfg *config.PostgresConfig) (openrtb2_config.ConfigFetcher, error) {
	db, err := sql.Open("postgres", confToPostgresDSN(cfg))
	if err != nil {
		return nil, err
	}

	return &dbFetcher{
		db: db,
		queryMaker: cfg.MakeQuery,
	}, nil
}

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
