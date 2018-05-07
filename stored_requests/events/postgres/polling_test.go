package postgres

import (
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

const updateQuery = "SELECT id, requestData, type FROM stored_data"

func updateQueryRegex() string {
	return "^" + regexp.QuoteMeta(updateQuery) + "$"
}

func TestSuccessfulUpdates(t *testing.T) {
	db, mock := newMock(t)
	mockRows := sqlmock.NewRows([]string{"id", "data", "dataType"}).
		AddRow("stored-req-1", "true", "request").
		AddRow("stored-req-2", "null", "request").
		AddRow("stored-imp-1", `{"id":1}`, "imp").
		AddRow("stored-imp-2", `{"id":2}`, "imp").
		AddRow("stored-imp-3", "", "imp")

	updateStart := time.Now()

	mock.ExpectQuery(initialQueryRegex()).WillReturnRows(mockRows)

	evs := PollForUpdates(nil, db, updateQuery, updateStart, time.Duration(-1))
	timeChan := make(chan time.Time)
	go evs.refresh(timeChan)
	timeChan <- time.Now()

	save := <-evs.Saves()
	assertMapLength(t, 1, save.Requests)
	assertMapValue(t, save.Requests, "stored-req-1", "true")
	assertMapLength(t, 2, save.Imps)
	assertMapValue(t, save.Imps, "stored-imp-1", `{"id":1}`)
	assertMapValue(t, save.Imps, "stored-imp-2", `{"id":2}`)

	invalidate := <-evs.Invalidations()
	assertNumInvalidations(t, 1, invalidate.Requests)
	assertSliceContains(t, invalidate.Requests, "stored-req-2")
	assertNumInvalidations(t, 1, invalidate.Imps)
	assertSliceContains(t, invalidate.Imps, "stored-imp-3")
}

func assertNumInvalidations(t *testing.T, expected int, vals []string) {
	t.Helper()

	if len(vals) != expected {
		t.Errorf("Expected %d invalidations. Got: %v", expected, vals)
	}
}

func assertSliceContains(t *testing.T, haystack []string, needle string) {
	t.Helper()
	for _, elm := range haystack {
		if elm == needle {
			return
		}
	}
	t.Errorf("expected element %s to be in list %v", needle, haystack)
}
