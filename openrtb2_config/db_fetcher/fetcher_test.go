package db_fetcher

import (
	"testing"
	"github.com/DATA-DOG/go-sqlmock"
	"regexp"
	"fmt"
	"encoding/json"
	"database/sql/driver"
	"errors"
)

func TestEmptyQuery(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Unexpected error stubbing DB: %v", err)
	}
	defer db.Close()

	fetcher := dbFetcher{
		db: db,
		queryMaker: successfulQueryMaker(""),
	}
	configs, errs := fetcher.GetConfigs(nil)
	if len(errs) != 0 {
		t.Errorf("Unexpected errors: %v", errs)
	}
	if len(configs) != 0 {
		t.Errorf("Bad configmap size. Expected %d, got %d.", 0, len(configs))
	}
}

// TestGoodResponse makes sure we interpret DB responses properly when all the configs are there.
func TestGoodResponse(t *testing.T) {
	mockQuery := "SELECT id, config FROM my_table WHERE id IN (?, ?)"
	mockReturn := sqlmock.NewRows([]string{"id", "config"}).
				AddRow("config-id", "{}")

	mock, fetcher, err := newFetcher(mockReturn, mockQuery, "config-id")
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer fetcher.db.Close()

	configs, errs := fetcher.GetConfigs([]string{"config-id"})

	assertMockExpectations(t, mock)
	assertErrorCount(t, 0, errs)
	assertMapLength(t, 1, configs)
	assertHasConfig(t, configs, "config-id", "{}")
}

// TestPartialResponse makes sure we unpack things properly when the DB finds some of the configs.
func TestPartialResponse(t *testing.T) {
	mockQuery := "SELECT id, config FROM my_table WHERE id IN (?, ?)"
	mockReturn := sqlmock.NewRows([]string{"id", "config"}).
		AddRow("config-id", "{}")

	mock, fetcher, err := newFetcher(mockReturn, mockQuery, "config-id", "config-id-2")
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer fetcher.db.Close()

	configs, errs := fetcher.GetConfigs([]string{"config-id", "config-id-2"})

	assertMockExpectations(t, mock)
	assertErrorCount(t, 1, errs)
	assertMapLength(t, 1, configs)
	assertHasConfig(t, configs, "config-id", "{}")
}

// TestEmptyResponse makes sure we handle empty DB responses properly.
func TestEmptyResponse(t *testing.T) {
	mockQuery := "SELECT id, config FROM my_table WHERE id IN (?, ?)"
	mockReturn := sqlmock.NewRows([]string{"id", "config"})

	mock, fetcher, err := newFetcher(mockReturn, mockQuery, "config-id", "config-id-2")
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer fetcher.db.Close()

	configs, errs := fetcher.GetConfigs([]string{"config-id", "config-id-2"})

	assertMockExpectations(t, mock)
	assertErrorCount(t, 2, errs)
	assertMapLength(t, 0, configs)
}

// TestQueryMakerError makes sure we exit with an error if the queryMaker function fails.
func TestQueryMakerError(t *testing.T) {
	fetcher := &dbFetcher{
		db: nil,
		queryMaker: failedQueryMaker,
	}

	cfgs, errs := fetcher.GetConfigs([]string{"config-id"})
	assertErrorCount(t, 1, errs)
	assertMapLength(t, 0, cfgs)
}

// TestDatabaseError makes sure we exit with an error if the DB query fails.
func TestDatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}

	mock.ExpectQuery(".*").WillReturnError(errors.New("Invalid query."))

	fetcher := &dbFetcher{
		db: db,
		queryMaker: successfulQueryMaker("SELECT id, config FROM my_table WHERE id IN (?, ?)"),
	}

	cfgs, errs := fetcher.GetConfigs([]string{"config-id"})
	assertErrorCount(t, 1, errs)
	assertMapLength(t, 0, cfgs)
}

func newFetcher(rows *sqlmock.Rows, query string, args ...driver.Value) (sqlmock.Sqlmock, *dbFetcher, error) {
	db, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, err
	}

	queryRegex := fmt.Sprintf("^%s$", regexp.QuoteMeta(query))
	mock.ExpectQuery(queryRegex).WithArgs(args...).WillReturnRows(rows)
	fetcher := &dbFetcher{
		db: db,
		queryMaker: successfulQueryMaker(query),
	}

	return 	mock, fetcher, nil
}

func assertMapLength(t *testing.T, numExpected int, configs map[string]json.RawMessage) {
	t.Helper()
	if len(configs) != numExpected {
		t.Errorf("Wrong num configs. Expected %d, Got %d.", numExpected, len(configs))
	}
}

func assertMockExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Mock expectations not met: %v", err)
	}
}

func assertHasConfig(t *testing.T, configs map[string]json.RawMessage, key string, value string) {
	t.Helper()
	cfg, ok := configs[key]
	if !ok {
		t.Errorf("Missing expected config: %s", key)
	}
	if string(cfg) != value {
		t.Errorf("Bad configs[%s] value. Expected %s, Got %s", key, value, cfg)
	}
}

func assertErrorCount(t *testing.T, num int, errs []error) {
	t.Helper()
	if len(errs) != num {
		t.Errorf("Wrong number of errors. Expected %d. Got %d", num, len(errs))
	}
}

func successfulQueryMaker(response string) func(int) (string, error) {
	return func(numIds int) (string, error) {
		return response, nil
	}
}

func failedQueryMaker(_ int)(string, error) {
		return "", errors.New("The query maker failed.")
}
