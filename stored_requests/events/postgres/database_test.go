package postgres

import (
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/prebid/prebid-server/stored_requests/events"
	"github.com/stretchr/testify/assert"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// FakeTime implements the Time interface
type FakeTime struct {
	time time.Time
}

func (mc *FakeTime) Now() time.Time {
	return mc.time
}

const fakeQuery = "SELECT id, requestData, type FROM stored_data"

func fakeQueryRegex() string {
	return "^" + regexp.QuoteMeta(fakeQuery) + "$"
}

func TestFetchAllSuccess(t *testing.T) {
	tests := []struct {
		description         string
		giveFakeTime        time.Time
		giveMockRows        *sqlmock.Rows
		wantLastUpdate      time.Time
		wantSavedReqs       map[string]json.RawMessage
		wantSavedImps       map[string]json.RawMessage
		wantInvalidatedReqs []string
		wantInvalidatedImps []string
	}{
		{
			description:    "saved reqs = 0, saved imps = 0, invalidated reqs = 0, invalidated imps = 0",
			giveFakeTime:   time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveMockRows:   sqlmock.NewRows([]string{"id", "data", "dataType"}),
			wantLastUpdate: time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
		},
		{
			description:    "saved reqs > 0, saved imps = 0, invalidated reqs = 0, invalidated imps = 0",
			giveFakeTime:   time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveMockRows:   sqlmock.NewRows([]string{"id", "data", "dataType"}).AddRow("req-1", "true", "request"),
			wantLastUpdate: time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			wantSavedReqs:  map[string]json.RawMessage{"req-1": json.RawMessage(`true`)},
			wantSavedImps:  map[string]json.RawMessage{},
		},
		{
			description:    "saved reqs = 0, saved imps > 0, invalidated reqs = 0, invalidated imps = 0",
			giveFakeTime:   time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveMockRows:   sqlmock.NewRows([]string{"id", "data", "dataType"}).AddRow("imp-1", "true", "imp"),
			wantLastUpdate: time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			wantSavedReqs:  map[string]json.RawMessage{},
			wantSavedImps:  map[string]json.RawMessage{"imp-1": json.RawMessage(`true`)},
		},
		{
			description:    "saved reqs = 0, saved imps = 0, invalidated reqs > 0, invalidated imps = 0",
			giveFakeTime:   time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveMockRows:   sqlmock.NewRows([]string{"id", "data", "dataType"}).AddRow("req-1", "", "request"),
			wantLastUpdate: time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
		},
		{
			description:    "saved reqs = 0, saved imps = 0, invalidated reqs = 0, invalidated imps > 0",
			giveFakeTime:   time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveMockRows:   sqlmock.NewRows([]string{"id", "data", "dataType"}).AddRow("imp-1", "", "imp"),
			wantLastUpdate: time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
		},
		{
			description:  "saved reqs > 0, saved imps > 0, invalidated reqs > 0, invalidated imps > 0",
			giveFakeTime: time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveMockRows: sqlmock.NewRows([]string{"id", "data", "dataType"}).
				AddRow("req-1", "true", "request").
				AddRow("imp-1", "true", "imp").
				AddRow("req-2", "", "request").
				AddRow("imp-2", "", "imp"),
			wantLastUpdate: time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			wantSavedReqs:  map[string]json.RawMessage{"req-1": json.RawMessage(`true`)},
			wantSavedImps:  map[string]json.RawMessage{"imp-1": json.RawMessage(`true`)},
		},
	}

	for _, tt := range tests {
		db, mock, _ := sqlmock.New()
		mock.ExpectQuery(fakeQueryRegex()).WillReturnRows(tt.giveMockRows)

		eventProducer := NewPostgresEventProducer(PostgresEventProducerConfig{
			DB:               db,
			CacheInitTimeout: 100 * time.Millisecond,
			CacheInitQuery:   fakeQuery,
		})
		eventProducer.time = &FakeTime{time: tt.giveFakeTime}
		err := eventProducer.Run()

		assert.Nil(t, err, tt.description)
		assert.Equal(t, tt.wantLastUpdate, eventProducer.lastUpdate, tt.description)

		var saves events.Save
		// Read data from saves channel with timeout to avoid test suite deadlock
		select {
		case saves = <-eventProducer.Saves():
		case <-time.After(20 * time.Millisecond):
		}
		var invalidations events.Invalidation
		// Read data from invalidations channel with timeout to avoid test suite deadlock
		select {
		case invalidations = <-eventProducer.Invalidations():
		case <-time.After(20 * time.Millisecond):
		}

		assert.Equal(t, tt.wantSavedReqs, saves.Requests, tt.description)
		assert.Equal(t, tt.wantSavedImps, saves.Imps, tt.description)
		assert.Equal(t, tt.wantInvalidatedReqs, invalidations.Requests, tt.description)
		assert.Equal(t, tt.wantInvalidatedImps, invalidations.Imps, tt.description)
	}
}

func TestFetchAllErrors(t *testing.T) {
	tests := []struct {
		description         string
		giveFakeTime        time.Time
		giveMockRows        *sqlmock.Rows
		wantReturnedError   bool
		wantLastUpdate      time.Time
		wantSavedReqs       map[string]json.RawMessage
		wantSavedImps       map[string]json.RawMessage
		wantInvalidatedReqs []string
		wantInvalidatedImps []string
	}{
		{
			description:       "fetch all query error",
			giveFakeTime:      time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveMockRows:      nil,
			wantReturnedError: true,
			wantLastUpdate:    time.Time{},
		},
		{
			description:  "fetch all row error",
			giveFakeTime: time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveMockRows: sqlmock.NewRows([]string{"id", "data", "dataType"}).
				AddRow("stored-req-id", "true", "request").
				RowError(0, errors.New("Some row error.")),
			wantReturnedError: true,
			wantLastUpdate:    time.Time{},
		},
		{
			description:  "fetch all close error",
			giveFakeTime: time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveMockRows: sqlmock.NewRows([]string{"id", "data", "dataType"}).
				AddRow("req-1", "true", "request").
				AddRow("imp-1", "true", "imp").
				AddRow("req-2", "", "request").
				AddRow("imp-2", "", "imp").
				CloseError(errors.New("Some close error.")),
			wantReturnedError: false,
			wantLastUpdate:    time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			wantSavedReqs:     map[string]json.RawMessage{"req-1": json.RawMessage(`true`)},
			wantSavedImps:     map[string]json.RawMessage{"imp-1": json.RawMessage(`true`)},
		},
	}

	for _, tt := range tests {
		db, mock, _ := sqlmock.New()
		if tt.giveMockRows == nil {
			mock.ExpectQuery(fakeQueryRegex()).WillReturnError(errors.New("Query failed."))
		} else {
			mock.ExpectQuery(fakeQueryRegex()).WillReturnRows(tt.giveMockRows)
		}

		eventProducer := NewPostgresEventProducer(PostgresEventProducerConfig{
			DB:               db,
			CacheInitTimeout: 100 * time.Millisecond,
			CacheInitQuery:   fakeQuery,
		})
		eventProducer.time = &FakeTime{time: tt.giveFakeTime}
		err := eventProducer.Run()

		if tt.wantReturnedError {
			assert.NotNil(t, err, tt.description)
		} else {
			assert.Nil(t, err, tt.description)
		}
		assert.Equal(t, tt.wantLastUpdate, eventProducer.lastUpdate, tt.description)

		var saves events.Save
		// Read data from saves channel with timeout to avoid test suite deadlock
		select {
		case saves = <-eventProducer.Saves():
		case <-time.After(10 * time.Millisecond):
		}
		var invalidations events.Invalidation
		// Read data from invalidations channel with timeout to avoid test suite deadlock
		select {
		case invalidations = <-eventProducer.Invalidations():
		case <-time.After(10 * time.Millisecond):
		}

		assert.Equal(t, tt.wantSavedReqs, saves.Requests, tt.description)
		assert.Equal(t, tt.wantSavedImps, saves.Imps, tt.description)
		assert.Equal(t, tt.wantInvalidatedReqs, invalidations.Requests, tt.description)
		assert.Equal(t, tt.wantInvalidatedImps, invalidations.Imps, tt.description)
	}
}

func TestFetchDeltaSuccess(t *testing.T) {
	tests := []struct {
		description         string
		giveFakeTime        time.Time
		giveMockRows        *sqlmock.Rows
		wantLastUpdate      time.Time
		wantSavedReqs       map[string]json.RawMessage
		wantSavedImps       map[string]json.RawMessage
		wantInvalidatedReqs []string
		wantInvalidatedImps []string
	}{
		{
			description:    "saved reqs = 0, saved imps = 0, invalidated reqs = 0, invalidated imps = 0",
			giveFakeTime:   time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveMockRows:   sqlmock.NewRows([]string{"id", "data", "dataType"}),
			wantLastUpdate: time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
		},
		{
			description:    "saved reqs > 0, saved imps = 0, invalidated reqs = 0, invalidated imps = 0",
			giveFakeTime:   time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveMockRows:   sqlmock.NewRows([]string{"id", "data", "dataType"}).AddRow("req-1", "true", "request"),
			wantLastUpdate: time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			wantSavedReqs:  map[string]json.RawMessage{"req-1": json.RawMessage(`true`)},
			wantSavedImps:  map[string]json.RawMessage{},
		},
		{
			description:    "saved reqs = 0, saved imps > 0, invalidated reqs = 0, invalidated imps = 0",
			giveFakeTime:   time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveMockRows:   sqlmock.NewRows([]string{"id", "data", "dataType"}).AddRow("imp-1", "true", "imp"),
			wantLastUpdate: time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			wantSavedReqs:  map[string]json.RawMessage{},
			wantSavedImps:  map[string]json.RawMessage{"imp-1": json.RawMessage(`true`)},
		},
		{
			description:         "saved reqs = 0, saved imps = 0, invalidated reqs > 0, invalidated imps = 0, empty data",
			giveFakeTime:        time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveMockRows:        sqlmock.NewRows([]string{"id", "data", "dataType"}).AddRow("req-1", "", "request"),
			wantLastUpdate:      time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			wantInvalidatedReqs: []string{"req-1"},
			wantInvalidatedImps: nil,
		},
		{
			description:         "saved reqs = 0, saved imps = 0, invalidated reqs > 0, invalidated imps = 0, null data",
			giveFakeTime:        time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveMockRows:        sqlmock.NewRows([]string{"id", "data", "dataType"}).AddRow("req-1", "null", "request"),
			wantLastUpdate:      time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			wantInvalidatedReqs: []string{"req-1"},
			wantInvalidatedImps: nil,
		},
		{
			description:         "saved reqs = 0, saved imps = 0, invalidated reqs = 0, invalidated imps > 0, empty data",
			giveFakeTime:        time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveMockRows:        sqlmock.NewRows([]string{"id", "data", "dataType"}).AddRow("imp-1", "", "imp"),
			wantLastUpdate:      time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			wantInvalidatedImps: []string{"imp-1"},
		},
		{
			description:         "saved reqs = 0, saved imps = 0, invalidated reqs = 0, invalidated imps > 0, null data",
			giveFakeTime:        time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveMockRows:        sqlmock.NewRows([]string{"id", "data", "dataType"}).AddRow("imp-1", "null", "imp"),
			wantLastUpdate:      time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			wantInvalidatedImps: []string{"imp-1"},
		},
		{
			description:  "saved reqs > 0, saved imps > 0, invalidated reqs > 0, invalidated imps > 0",
			giveFakeTime: time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveMockRows: sqlmock.NewRows([]string{"id", "data", "dataType"}).
				AddRow("req-1", "true", "request").
				AddRow("imp-1", "true", "imp").
				AddRow("req-2", "", "request").
				AddRow("imp-2", "", "imp"),
			wantLastUpdate:      time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			wantSavedReqs:       map[string]json.RawMessage{"req-1": json.RawMessage(`true`)},
			wantSavedImps:       map[string]json.RawMessage{"imp-1": json.RawMessage(`true`)},
			wantInvalidatedReqs: []string{"req-2"},
			wantInvalidatedImps: []string{"imp-2"},
		},
	}

	for _, tt := range tests {
		db, mock, _ := sqlmock.New()
		mock.ExpectQuery(fakeQueryRegex()).WillReturnRows(tt.giveMockRows)

		eventProducer := NewPostgresEventProducer(PostgresEventProducerConfig{
			DB:                 db,
			CacheUpdateTimeout: 100 * time.Millisecond,
			CacheUpdateQuery:   fakeQuery,
		})
		eventProducer.lastUpdate = time.Date(2020, time.June, 30, 6, 0, 0, 0, time.UTC)
		eventProducer.time = &FakeTime{time: tt.giveFakeTime}
		err := eventProducer.Run()

		assert.Nil(t, err, tt.description)
		assert.Equal(t, tt.wantLastUpdate, eventProducer.lastUpdate, tt.description)

		var saves events.Save
		// Read data from saves channel with timeout to avoid test suite deadlock
		select {
		case saves = <-eventProducer.Saves():
		case <-time.After(20 * time.Millisecond):
		}
		var invalidations events.Invalidation
		// Read data from invalidations channel with timeout to avoid test suite deadlock
		select {
		case invalidations = <-eventProducer.Invalidations():
		case <-time.After(20 * time.Millisecond):
		}

		assert.Equal(t, tt.wantSavedReqs, saves.Requests, tt.description)
		assert.Equal(t, tt.wantSavedImps, saves.Imps, tt.description)
		assert.Equal(t, tt.wantInvalidatedReqs, invalidations.Requests, tt.description)
		assert.Equal(t, tt.wantInvalidatedImps, invalidations.Imps, tt.description)
	}
}

func TestFetchDeltaErrors(t *testing.T) {
	tests := []struct {
		description         string
		giveFakeTime        time.Time
		giveLastUpdate      time.Time
		giveMockRows        *sqlmock.Rows
		wantReturnedError   bool
		wantLastUpdate      time.Time
		wantSavedReqs       map[string]json.RawMessage
		wantSavedImps       map[string]json.RawMessage
		wantInvalidatedReqs []string
		wantInvalidatedImps []string
	}{
		{
			description:       "fetch delta query error",
			giveFakeTime:      time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveLastUpdate:    time.Date(2020, time.June, 30, 6, 0, 0, 0, time.UTC),
			giveMockRows:      nil,
			wantReturnedError: true,
			wantLastUpdate:    time.Date(2020, time.June, 30, 6, 0, 0, 0, time.UTC),
		},
		{
			description:    "fetch all row error",
			giveFakeTime:   time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveLastUpdate: time.Date(2020, time.June, 30, 6, 0, 0, 0, time.UTC),
			giveMockRows: sqlmock.NewRows([]string{"id", "data", "dataType"}).
				AddRow("stored-req-id", "true", "request").
				RowError(0, errors.New("Some row error.")),
			wantReturnedError: true,
			wantLastUpdate:    time.Date(2020, time.June, 30, 6, 0, 0, 0, time.UTC),
		},
		{
			description:    "fetch all close error",
			giveFakeTime:   time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveLastUpdate: time.Date(2020, time.June, 30, 6, 0, 0, 0, time.UTC),
			giveMockRows: sqlmock.NewRows([]string{"id", "data", "dataType"}).
				AddRow("req-1", "true", "request").
				AddRow("imp-1", "true", "imp").
				AddRow("req-2", "", "request").
				AddRow("imp-2", "", "imp").
				CloseError(errors.New("Some close error.")),
			wantReturnedError:   false,
			wantLastUpdate:      time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			wantSavedReqs:       map[string]json.RawMessage{"req-1": json.RawMessage(`true`)},
			wantSavedImps:       map[string]json.RawMessage{"imp-1": json.RawMessage(`true`)},
			wantInvalidatedReqs: []string{"req-2"},
			wantInvalidatedImps: []string{"imp-2"},
		},
	}

	for _, tt := range tests {
		db, mock, _ := sqlmock.New()
		if tt.giveMockRows == nil {
			mock.ExpectQuery(fakeQueryRegex()).WillReturnError(errors.New("Query failed."))
		} else {
			mock.ExpectQuery(fakeQueryRegex()).WillReturnRows(tt.giveMockRows)
		}

		eventProducer := NewPostgresEventProducer(PostgresEventProducerConfig{
			DB:                 db,
			CacheUpdateTimeout: 100 * time.Millisecond,
			CacheUpdateQuery:   fakeQuery,
		})
		eventProducer.lastUpdate = tt.giveLastUpdate
		eventProducer.time = &FakeTime{time: tt.giveFakeTime}
		err := eventProducer.Run()

		if tt.wantReturnedError {
			assert.NotNil(t, err, tt.description)
		} else {
			assert.Nil(t, err, tt.description)
		}
		assert.Equal(t, tt.wantLastUpdate, eventProducer.lastUpdate, tt.description)

		var saves events.Save
		// Read data from saves channel with timeout to avoid test suite deadlock
		select {
		case saves = <-eventProducer.Saves():
		case <-time.After(10 * time.Millisecond):
		}
		var invalidations events.Invalidation
		// Read data from invalidations channel with timeout to avoid test suite deadlock
		select {
		case invalidations = <-eventProducer.Invalidations():
		case <-time.After(10 * time.Millisecond):
		}

		assert.Equal(t, tt.wantSavedReqs, saves.Requests, tt.description)
		assert.Equal(t, tt.wantSavedImps, saves.Imps, tt.description)
		assert.Equal(t, tt.wantInvalidatedReqs, invalidations.Requests, tt.description)
		assert.Equal(t, tt.wantInvalidatedImps, invalidations.Imps, tt.description)
	}
}
