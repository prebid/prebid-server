package db_fetcher

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"regexp"
	"testing"
	"time"
)

func TestEmptyQuery(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Unexpected error stubbing DB: %v", err)
	}
	defer db.Close()

	fetcher := dbFetcher{
		db:         db,
		queryMaker: successfulQueryMaker(""),
	}
	storedReqs, errs := fetcher.FetchRequests(context.Background(), nil)
	if len(errs) != 0 {
		t.Errorf("Unexpected errors: %v", errs)
	}
	if len(storedReqs) != 0 {
		t.Errorf("Bad map size. Expected %d, got %d.", 0, len(storedReqs))
	}
}

// TestGoodResponse makes sure we interpret DB responses properly when all the stored requests are there.
func TestGoodResponse(t *testing.T) {
	mockQuery := "SELECT id, requestData FROM my_table WHERE id IN (?, ?)"
	mockReturn := sqlmock.NewRows([]string{"id", "requestData"}).
		AddRow("request-id", "{}")

	mock, fetcher, err := newFetcher(mockReturn, mockQuery, "request-id")
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer fetcher.db.Close()

	storedReqs, errs := fetcher.FetchRequests(context.Background(), []string{"request-id"})

	assertMockExpectations(t, mock)
	assertErrorCount(t, 0, errs)
	assertMapLength(t, 1, storedReqs)
	assertHasData(t, storedReqs, "request-id", "{}")
}

// TestPartialResponse makes sure we unpack things properly when the DB finds some of the stored requests.
func TestPartialResponse(t *testing.T) {
	mockQuery := "SELECT id, requestData FROM my_table WHERE id IN (?, ?)"
	mockReturn := sqlmock.NewRows([]string{"id", "requestData"}).
		AddRow("stored-req-id", "{}")

	mock, fetcher, err := newFetcher(mockReturn, mockQuery, "stored-req-id", "stored-req-id-2")
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer fetcher.db.Close()

	storedReqs, errs := fetcher.FetchRequests(context.Background(), []string{"stored-req-id", "stored-req-id-2"})

	assertMockExpectations(t, mock)
	assertErrorCount(t, 1, errs)
	assertMapLength(t, 1, storedReqs)
	assertHasData(t, storedReqs, "stored-req-id", "{}")
}

// TestEmptyResponse makes sure we handle empty DB responses properly.
func TestEmptyResponse(t *testing.T) {
	mockQuery := "SELECT id, requestData FROM my_table WHERE id IN (?, ?)"
	mockReturn := sqlmock.NewRows([]string{"id", "requestData"})

	mock, fetcher, err := newFetcher(mockReturn, mockQuery, "stored-req-id", "stored-req-id-2")
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer fetcher.db.Close()

	storedReqs, errs := fetcher.FetchRequests(context.Background(), []string{"stored-req-id", "stored-req-id-2"})

	assertMockExpectations(t, mock)
	assertErrorCount(t, 2, errs)
	assertMapLength(t, 0, storedReqs)
}

// TestQueryMakerError makes sure we exit with an error if the queryMaker function fails.
func TestQueryMakerError(t *testing.T) {
	fetcher := &dbFetcher{
		db:         nil,
		queryMaker: failedQueryMaker,
	}

	storedReqs, errs := fetcher.FetchRequests(context.Background(), []string{"stored-req-id"})
	assertErrorCount(t, 1, errs)
	assertMapLength(t, 0, storedReqs)
}

// TestDatabaseError makes sure we exit with an error if the DB query fails.
func TestDatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}

	mock.ExpectQuery(".*").WillReturnError(errors.New("Invalid query."))

	fetcher := &dbFetcher{
		db:         db,
		queryMaker: successfulQueryMaker("SELECT id, requestData FROM my_table WHERE id IN (?, ?)"),
	}

	cfgs, errs := fetcher.FetchRequests(context.Background(), []string{"stored-req-id"})
	assertErrorCount(t, 1, errs)
	assertMapLength(t, 0, cfgs)
}

// TestContextDeadlines makes sure a hung query returns when the timeout expires.
func TestContextDeadlines(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}

	mock.ExpectQuery(".*").WillDelayFor(2 * time.Minute)

	fetcher := &dbFetcher{
		db:         db,
		queryMaker: successfulQueryMaker("SELECT id, requestData FROM my_table WHERE id IN (?, ?)"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	_, errs := fetcher.FetchRequests(ctx, []string{"id"})
	if len(errs) < 1 {
		t.Errorf("dbFetcher should return an error when the context times out.")
	}
}

// TestContextCancelled makes sure a hung query returns when the context is cancelled.
func TestContextCancelled(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}

	mock.ExpectQuery(".*").WillDelayFor(2 * time.Minute)

	fetcher := &dbFetcher{
		db:         db,
		queryMaker: successfulQueryMaker("SELECT id, requestData FROM my_table WHERE id IN (?, ?)"),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, errs := fetcher.FetchRequests(ctx, []string{"id"})
	if len(errs) < 1 {
		t.Errorf("dbFetcher should return an error when the context is cancelled.")
	}
}

func newFetcher(rows *sqlmock.Rows, query string, args ...driver.Value) (sqlmock.Sqlmock, *dbFetcher, error) {
	db, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, err
	}

	queryRegex := fmt.Sprintf("^%s$", regexp.QuoteMeta(query))
	mock.ExpectQuery(queryRegex).WithArgs(args...).WillReturnRows(rows)
	fetcher := &dbFetcher{
		db:         db,
		queryMaker: successfulQueryMaker(query),
	}

	return mock, fetcher, nil
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

func assertHasData(t *testing.T, data map[string]json.RawMessage, key string, value string) {
	t.Helper()
	cfg, ok := data[key]
	if !ok {
		t.Errorf("Missing expected stored request data: %s", key)
	}
	if string(cfg) != value {
		t.Errorf("Bad data[%s] value. Expected %s, Got %s", key, value, cfg)
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

func failedQueryMaker(_ int) (string, error) {
	return "", errors.New("The query maker failed.")
}
