package db_fetcher

import (
	"database/sql"

	"github.com/prebid/prebid-server/config"
)

func NewPostgresDb(cfg *config.PostgresFetcherConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.ConnectionInfo.ConnString())
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
