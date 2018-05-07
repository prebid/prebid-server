package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestSuccessfulFetch(t *testing.T) {
	db, mock := newMock(t)
	mockRows := sqlmock.NewRows([]string{"id", "data", "dataType"}).
		AddRow("stored-req-id", "true", "request").
		AddRow("stored-imp-1", `{"id":1}`, "imp").
		AddRow("stored-imp-2", `{"id":2}`, "imp")

	mock.ExpectQuery(initialQueryRegex()).WillReturnRows(mockRows)

	evs := LoadAll(context.Background(), db, initialQuery)
	save := <-evs.Saves()
	assertMapLength(t, 1, save.Requests)
	assertMapValue(t, save.Requests, "stored-req-id", "true")

	assertMapLength(t, 2, save.Imps)
	assertMapValue(t, save.Imps, "stored-imp-1", `{"id":1}`)
	assertMapValue(t, save.Imps, "stored-imp-2", `{"id":2}`)
	assertExpectationsMet(t, mock)
}

// Make sure that an empty save still gets sent on the channel if the SQL query fails.
func TestQueryError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(initialQueryRegex()).WillReturnError(errors.New("Query failed."))

	evs := LoadAll(context.Background(), db, initialQuery)
	save := <-evs.Saves()
	assertMapLength(t, 0, save.Requests)
	assertMapLength(t, 0, save.Imps)
	assertExpectationsMet(t, mock)
}

func TestRowError(t *testing.T) {
	db, mock := newMock(t)
	mockRows := sqlmock.NewRows([]string{"id", "data", "dataType"}).
		AddRow("stored-req-id", "true", "request").
		AddRow("stored-imp-1", `{"id":1}`, "imp").
		RowError(1, errors.New("Some row error."))
	mock.ExpectQuery(initialQueryRegex()).WillReturnRows(mockRows)

	evs := LoadAll(context.Background(), db, initialQuery)
	save := <-evs.Saves()
	assertMapLength(t, 0, save.Requests)
	assertMapLength(t, 0, save.Imps)
	assertExpectationsMet(t, mock)
}

func TestRowCloseError(t *testing.T) {
	db, mock := newMock(t)
	mockRows := sqlmock.NewRows([]string{"id", "data", "dataType"}).
		AddRow("stored-req-id", "true", "request").
		AddRow("stored-imp-id", `{"id":1}`, "imp").
		CloseError(errors.New("Failed to close rows."))
	mock.ExpectQuery(initialQueryRegex()).WillReturnRows(mockRows)

	evs := LoadAll(context.Background(), db, initialQuery)
	save := <-evs.Saves()
	assertMapLength(t, 1, save.Requests)
	assertMapLength(t, 1, save.Imps)
	assertExpectationsMet(t, mock)
}

func newMock(t *testing.T) (db *sql.DB, mock sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	return
}

const initialQuery = "SELECT id, requestData, type FROM stored_data"

func initialQueryRegex() string {
	return "^" + regexp.QuoteMeta(initialQuery) + "$"
}

type result struct {
	id       string
	data     json.RawMessage
	dataType string
}

func assertMapLength(t *testing.T, expectedLen int, theMap map[string]json.RawMessage) {
	t.Helper()
	if len(theMap) != expectedLen {
		t.Errorf("Wrong map length. Expected %d, Got %d.", expectedLen, len(theMap))
	}
}

func assertMapValue(t *testing.T, m map[string]json.RawMessage, key string, val string) {
	t.Helper()
	if mapVal, ok := m[key]; ok {
		if !bytes.Equal(mapVal, []byte(val)) {
			t.Errorf("expected map[%s] to be %s, but got %s", key, val, string(mapVal))
		}
	} else {
		t.Errorf("map missing expected key: %s", key)
	}
}

func assertExpectationsMet(t *testing.T, mock sqlmock.Sqlmock) {
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations were not met: %v", err)
	}
}
