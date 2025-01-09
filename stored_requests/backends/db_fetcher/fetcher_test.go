package db_fetcher

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/prebid/prebid-server/v3/stored_requests/backends/db_provider"
	"github.com/stretchr/testify/assert"
)

func TestEmptyQuery(t *testing.T) {
	provider, _, err := db_provider.NewDbProviderMock()
	if err != nil {
		t.Fatalf("Unexpected error stubbing DB: %v", err)
	}
	defer provider.Close()

	fetcher := dbFetcher{
		provider:              provider,
		queryTemplate:         "",
		responseQueryTemplate: "",
	}
	storedReqs, storedImps, errs := fetcher.FetchRequests(context.Background(), nil, nil)
	assertErrorCount(t, 0, errs)
	assertMapLength(t, 0, storedReqs)
	assertMapLength(t, 0, storedImps)

	storedResponses, errs := fetcher.FetchResponses(context.Background(), nil)
	assertErrorCount(t, 0, errs)
	assertMapLength(t, 0, storedResponses)
}

// TestGoodResponse makes sure we interpret DB responses properly when all the stored requests are there.
func TestGoodResponse(t *testing.T) {
	mockQuery := "SELECT id, data, 'request' AS dataType FROM req_table WHERE id IN (?) UNION ALL SELECT id, data, 'imp' as dataType FROM imp_table WHERE id IN (?, ?)"
	mockReturn := sqlmock.NewRows([]string{"id", "data", "dataType"}).
		AddRow("request-id", `{"req":true}`, "request").
		AddRow("imp-id", `{"imp":true,"value":1}`, "imp").
		AddRow("imp-id-2", `{"imp":true,"value":2}`, "imp")

	mock, fetcher := newFetcher(t, mockReturn, mockQuery, "request-id")
	defer fetcher.provider.Close()

	storedReqs, storedImps, errs := fetcher.FetchRequests(context.Background(), []string{"request-id"}, nil)

	assertMockExpectations(t, mock)
	assertErrorCount(t, 0, errs)
	assertMapLength(t, 1, storedReqs)
	assertMapLength(t, 2, storedImps)
	assertHasData(t, storedReqs, "request-id", `{"req":true}`)
	assertHasData(t, storedImps, "imp-id", `{"imp":true,"value":1}`)
	assertHasData(t, storedImps, "imp-id-2", `{"imp":true,"value":2}`)
}

func TestFetchResponses(t *testing.T) {
	testCases := []struct {
		description  string
		mockQuery    string
		mockReturn   *sqlmock.Rows
		arguments    []driver.Value
		respIds      []string
		expectedResp map[string]string
	}{
		{
			description: "fetch good response",
			mockQuery:   "SELECT id, data, 'response' AS dataType FROM responses_table WHERE id IN (?, ?)",
			mockReturn: sqlmock.NewRows([]string{"id", "data", "dataType"}).
				AddRow("resp-id-1", `{"resp":false,"value":1}`, "response").
				AddRow("resp-id-2", `{"resp":true,"value":2}`, "response"),
			arguments:    []driver.Value{"resp-id-1", "resp-id-2"},
			respIds:      []string{"resp-id-1", "resp-id-2"},
			expectedResp: map[string]string{"resp-id-1": `{"resp":false,"value":1}`, "resp-id-2": `{"resp":true,"value":2}`},
		},
		{
			description: "fetch partial response",
			mockQuery:   "SELECT id, data, 'response' AS dataType FROM responses_table WHERE id IN (?, ?)",
			mockReturn: sqlmock.NewRows([]string{"id", "data", "dataType"}).
				AddRow("stored-resp-id", "{}", "response"),
			arguments:    []driver.Value{"stored-resp-id", "stored-resp-id-2"},
			respIds:      []string{"stored-resp-id", "stored-resp-id-2"},
			expectedResp: map[string]string{"stored-resp-id": `{}`},
		},
		{
			description:  "fetch empty response",
			mockQuery:    "SELECT id, data, dataType FROM my_table WHERE id IN (?, ?)",
			mockReturn:   sqlmock.NewRows([]string{"id", "data", "dataType"}),
			arguments:    []driver.Value{"stored-resp-id", "stored-resp-id-2"},
			respIds:      []string{"stored-resp-id", "stored-resp-id-2"},
			expectedResp: map[string]string{},
		},
	}

	for _, test := range testCases {
		mock, fetcher := newFetcher(t, test.mockReturn, test.mockQuery, test.arguments...)
		defer fetcher.provider.Close()

		storedResponses, errs := fetcher.FetchResponses(context.Background(), test.respIds)

		assertMockExpectations(t, mock)
		assertErrorCount(t, 0, errs)
		assert.Len(t, storedResponses, len(test.expectedResp))

		for k, v := range test.expectedResp {
			assertHasData(t, storedResponses, k, v)
		}

	}
}

// TestPartialResponse makes sure we unpack things properly when the DB finds some of the stored requests.
func TestPartialResponse(t *testing.T) {
	mockQuery := "SELECT id, data, 'request' AS dataType FROM req_table WHERE id IN (?, ?) UNION ALL SELECT id, data, 'imp' as dataType FROM imp_table WHERE id IN (NULL)"
	mockReturn := sqlmock.NewRows([]string{"id", "data", "dataType"}).
		AddRow("stored-req-id", "{}", "request")

	mock, fetcher := newFetcher(t, mockReturn, mockQuery, "stored-req-id", "stored-req-id-2")
	defer fetcher.provider.Close()

	storedReqs, storedImps, errs := fetcher.FetchRequests(context.Background(), []string{"stored-req-id", "stored-req-id-2"}, nil)

	assertMockExpectations(t, mock)
	assertErrorCount(t, 1, errs)
	assertMapLength(t, 0, storedImps)
	assertMapLength(t, 1, storedReqs)
	assertHasData(t, storedReqs, "stored-req-id", "{}")
}

// TestEmptyResponse makes sure we handle empty DB responses properly.
func TestEmptyResponse(t *testing.T) {
	mockQuery := "SELECT id, data, dataType FROM my_table WHERE id IN (?, ?)"
	mockReturn := sqlmock.NewRows([]string{"id", "data", "dataType"})

	mock, fetcher := newFetcher(t, mockReturn, mockQuery, "stored-req-id", "stored-req-id-2", "stored-imp-id")
	defer fetcher.provider.Close()

	storedReqs, storedImps, errs := fetcher.FetchRequests(context.Background(), []string{"stored-req-id", "stored-req-id-2"}, []string{"stored-imp-id"})

	assertMockExpectations(t, mock)
	assertErrorCount(t, 3, errs)
	assertMapLength(t, 0, storedReqs)
	assertMapLength(t, 0, storedImps)
}

// TestDatabaseError makes sure we exit with an error if the DB query fails.
func TestDatabaseError(t *testing.T) {
	provider, mock, err := db_provider.NewDbProviderMock()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}

	mock.ExpectQuery(".*").WillReturnError(errors.New("Invalid query."))

	fetcher := &dbFetcher{
		provider:      provider,
		queryTemplate: "SELECT id, data, dataType FROM my_table WHERE id IN (?, ?)",
	}

	storedReqs, storedImps, errs := fetcher.FetchRequests(context.Background(), []string{"stored-req-id"}, nil)
	assertErrorCount(t, 1, errs)
	assertMapLength(t, 0, storedReqs)
	assertMapLength(t, 0, storedImps)
}

// TestContextDeadlines makes sure a hung query returns when the timeout expires.
func TestContextDeadlines(t *testing.T) {
	provider, mock, err := db_provider.NewDbProviderMock()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}

	mock.ExpectQuery(".*").WillDelayFor(2 * time.Minute)

	fetcher := &dbFetcher{
		provider:              provider,
		queryTemplate:         "SELECT id, requestData FROM my_table WHERE id IN (?, ?)",
		responseQueryTemplate: "SELECT id, responseData FROM my_table WHERE id IN (?, ?)",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	_, _, errs := fetcher.FetchRequests(ctx, []string{"id"}, nil)
	if len(errs) < 1 {
		t.Errorf("dbFetcher should return an error when the context times out.")
	}
	_, errs = fetcher.FetchResponses(ctx, []string{"id"})
	if len(errs) < 1 {
		t.Errorf("dbFetcher should return an error when the context times out.")
	}
}

// TestContextCancelled makes sure a hung query returns when the context is cancelled.
func TestContextCancelled(t *testing.T) {
	provider, mock, err := db_provider.NewDbProviderMock()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}

	mock.ExpectQuery(".*").WillDelayFor(2 * time.Minute)

	fetcher := &dbFetcher{
		provider:              provider,
		queryTemplate:         "SELECT id, requestData FROM my_table WHERE id IN (?, ?)",
		responseQueryTemplate: "SELECT id, responseData FROM my_table WHERE id IN (?, ?)",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, errs := fetcher.FetchRequests(ctx, []string{"id"}, nil)
	if len(errs) < 1 {
		t.Errorf("dbFetcher should return an error when the context is cancelled.")
	}
	_, errs = fetcher.FetchResponses(ctx, []string{"id"})
	if len(errs) < 1 {
		t.Errorf("dbFetcher should return an error when the context is cancelled.")
	}
}

// Prevents #338
func TestRowErrors(t *testing.T) {
	provider, mock, err := db_provider.NewDbProviderMock()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	rows := sqlmock.NewRows([]string{"id", "data", "dataType"})
	rows.AddRow("foo", []byte(`{"data":1}`), "request")
	rows.AddRow("bar", []byte(`{"data":2}`), "request")
	rows.RowError(1, errors.New("Error reading from row 1"))
	mock.ExpectQuery(".*").WillReturnRows(rows)
	fetcher := &dbFetcher{
		provider:      provider,
		queryTemplate: "SELECT id, data, dataType FROM my_table WHERE id IN (?)",
	}
	data, _, errs := fetcher.FetchRequests(context.Background(), []string{"foo", "bar"}, nil)
	assertErrorCount(t, 1, errs)
	if errs[0].Error() != "Error reading from row 1" {
		t.Errorf("Unexpected error message: %v", errs[0].Error())
	}
	assertMapLength(t, 0, data)
}

func TestRowErrorsFetchResponses(t *testing.T) {
	provider, mock, err := db_provider.NewDbProviderMock()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	rows := sqlmock.NewRows([]string{"id", "data", "dataType"})
	rows.AddRow("foo", []byte(`{"data":1}`), "response")
	rows.AddRow("bar", []byte(`{"data":2}`), "response")
	rows.RowError(1, errors.New("Error reading from row 1"))
	mock.ExpectQuery(".*").WillReturnRows(rows)
	fetcher := &dbFetcher{
		provider:              provider,
		queryTemplate:         "SELECT id, data, dataType FROM my_table WHERE id IN (?)",
		responseQueryTemplate: "SELECT id, data, dataType FROM my_table WHERE id IN (?)",
	}
	data, errs := fetcher.FetchResponses(context.Background(), []string{"foo", "bar"})
	assertErrorCount(t, 1, errs)
	if errs[0].Error() != "Error reading from row 1" {
		t.Errorf("Unexpected error message: %v", errs[0].Error())
	}
	assertMapLength(t, 0, data)
}

func newFetcher(t *testing.T, rows *sqlmock.Rows, query string, args ...driver.Value) (sqlmock.Sqlmock, *dbFetcher) {
	provider, mock, err := db_provider.NewDbProviderMock()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
		return nil, nil
	}

	queryRegex := fmt.Sprintf("^%s$", regexp.QuoteMeta(query))
	mock.ExpectQuery(queryRegex).WithArgs(args...).WillReturnRows(rows)
	fetcher := &dbFetcher{
		provider:              provider,
		queryTemplate:         query,
		responseQueryTemplate: query,
	}

	return mock, fetcher
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
		t.Errorf("Wrong number of errors. Expected %d. Got %d. Errors are %v", num, len(errs), errs)
	}
}
