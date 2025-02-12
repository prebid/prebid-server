package db_provider

import (
	"context"
	"database/sql"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/config"
)

type DbProvider interface {
	Config() config.DatabaseConnection
	ConnString() (string, error)
	Open() error
	Close() error
	Ping() error
	PrepareQuery(template string, params ...QueryParam) (query string, args []interface{})
	QueryContext(ctx context.Context, template string, params ...QueryParam) (*sql.Rows, error)
}

func NewDbProvider(dataType config.DataType, cfg config.DatabaseConnection) DbProvider {
	var provider DbProvider

	switch cfg.Driver {
	case "mysql":
		provider = &MySqlDbProvider{
			cfg: cfg,
		}
	case "postgres":
		provider = &PostgresDbProvider{
			cfg: cfg,
		}
	default:
		glog.Fatalf("Unsupported database driver %s", cfg.Driver)
		return nil
	}

	if err := provider.Open(); err != nil {
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
