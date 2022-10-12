package db_provider

import (
	"context"
	"database/sql"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/config"
)

type DbProvider interface {
	Config() config.DatabaseConnection
	ConnString(cfg config.DatabaseConnection) string
	Open(cfg config.DatabaseConnection) error
	Close() error
	Ping() error
	PrepareQuery(template string, params ...QueryParam) (query string, args []interface{})
	QueryContext(ctx context.Context, template string, params ...QueryParam) (*sql.Rows, error)
}

func NewDbProvider(dataType config.DataType, cfg config.DatabaseConnection) DbProvider {
	var provider DbProvider

	switch cfg.Driver {
	case "mysql":
		provider = &MySqlDbProvider{}
	case "postgres":
		provider = &PostgresDbProvider{}
	default:
		glog.Fatalf("Unsupported database driver %s", cfg.Driver)
		return nil
	}

	err := provider.Open(cfg)

	if err != nil {
		glog.Fatalf("Failed to open %s database connection: %v", dataType, err)
	}
	if err := provider.Ping(); err != nil {
		glog.Fatalf("Failed to ping %s database: %v", dataType, err)
	}

	return provider
}

type QueryParam struct {
	Name  string
	Value interface{}
}
