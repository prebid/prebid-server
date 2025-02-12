package db_provider

import (
	"context"
	"database/sql"
	"reflect"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/prebid/prebid-server/v3/config"
)

func NewDbProviderMock() (*DbProviderMock, sqlmock.Sqlmock, error) {
	db, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, err
	}

	provider := &DbProviderMock{
		db:   db,
		mock: mock,
	}

	return provider, mock, nil
}

type DbProviderMock struct {
	db   *sql.DB
	mock sqlmock.Sqlmock
}

func (provider DbProviderMock) Config() config.DatabaseConnection {
	return config.DatabaseConnection{}
}

func (provider DbProviderMock) ConnString() (string, error) {
	return "", nil
}

func (provider DbProviderMock) Open() error {
	return nil
}

func (provider DbProviderMock) Close() error {
	return nil
}

func (provider DbProviderMock) Ping() error {
	return nil
}

func (provider DbProviderMock) PrepareQuery(template string, params ...QueryParam) (query string, args []interface{}) {
	for _, param := range params {
		if reflect.TypeOf(param.Value).Kind() == reflect.Slice {
			idList := param.Value.([]interface{})
			args = append(args, idList...)
		} else {
			args = append(args, param.Value)
		}
	}
	return template, args
}

func (provider DbProviderMock) QueryContext(ctx context.Context, template string, params ...QueryParam) (*sql.Rows, error) {
	query, args := provider.PrepareQuery(template, params...)

	return provider.db.QueryContext(ctx, query, args...)
}
