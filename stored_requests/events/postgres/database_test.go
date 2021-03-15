package postgres

import (
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/stored_requests/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

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
		db, dbMock, _ := sqlmock.New()
		dbMock.ExpectQuery(fakeQueryRegex()).WillReturnRows(tt.giveMockRows)

		metricsMock := &metrics.MetricsEngineMock{}
		metricsMock.Mock.On("RecordStoredDataFetchTime", metrics.StoredDataLabels{
			DataType:      metrics.RequestDataType,
			DataFetchType: metrics.FetchAll,
		}, mock.Anything).Return()

		eventProducer := NewPostgresEventProducer(PostgresEventProducerConfig{
			DB:               db,
			RequestType:      config.RequestDataType,
			CacheInitTimeout: 100 * time.Millisecond,
			CacheInitQuery:   fakeQuery,
			MetricsEngine:    metricsMock,
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

		metricsMock.AssertExpectations(t)
	}
}

func TestFetchAllErrors(t *testing.T) {
	tests := []struct {
		description       string
		giveFakeTime      time.Time
		giveTimeoutMS     int
		giveMockRows      *sqlmock.Rows
		wantRecordedError metrics.StoredDataError
		wantLastUpdate    time.Time
	}{
		{
			description:       "fetch all timeout",
			giveFakeTime:      time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveMockRows:      nil,
			wantRecordedError: metrics.StoredDataErrorNetwork,
			wantLastUpdate:    time.Time{},
		},
		{
			description:       "fetch all query error",
			giveFakeTime:      time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveTimeoutMS:     100,
			giveMockRows:      nil,
			wantRecordedError: metrics.StoredDataErrorUndefined,
			wantLastUpdate:    time.Time{},
		},
		{
			description:   "fetch all row error",
			giveFakeTime:  time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveTimeoutMS: 100,
			giveMockRows: sqlmock.NewRows([]string{"id", "data", "dataType"}).
				AddRow("stored-req-id", "true", "request").
				RowError(0, errors.New("Some row error.")),
			wantRecordedError: metrics.StoredDataErrorUndefined,
			wantLastUpdate:    time.Time{},
		},
	}

	for _, tt := range tests {
		db, dbMock, _ := sqlmock.New()
		if tt.giveMockRows == nil {
			dbMock.ExpectQuery(fakeQueryRegex()).WillReturnError(errors.New("Query failed."))
		} else {
			dbMock.ExpectQuery(fakeQueryRegex()).WillReturnRows(tt.giveMockRows)
		}

		metricsMock := &metrics.MetricsEngineMock{}
		metricsMock.Mock.On("RecordStoredDataFetchTime", metrics.StoredDataLabels{
			DataType:      metrics.RequestDataType,
			DataFetchType: metrics.FetchAll,
		}, mock.Anything).Return()
		metricsMock.Mock.On("RecordStoredDataError", metrics.StoredDataLabels{
			DataType: metrics.RequestDataType,
			Error:    tt.wantRecordedError,
		}).Return()

		eventProducer := NewPostgresEventProducer(PostgresEventProducerConfig{
			DB:               db,
			RequestType:      config.RequestDataType,
			CacheInitTimeout: time.Duration(tt.giveTimeoutMS) * time.Millisecond,
			CacheInitQuery:   fakeQuery,
			MetricsEngine:    metricsMock,
		})
		eventProducer.time = &FakeTime{time: tt.giveFakeTime}
		err := eventProducer.Run()

		assert.NotNil(t, err, tt.description)
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

		assert.Nil(t, saves.Requests, tt.description)
		assert.Nil(t, saves.Imps, tt.description)
		assert.Nil(t, invalidations.Requests, tt.description)
		assert.Nil(t, invalidations.Requests, tt.description)

		metricsMock.AssertExpectations(t)
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
		db, dbMock, _ := sqlmock.New()
		dbMock.ExpectQuery(fakeQueryRegex()).WillReturnRows(tt.giveMockRows)

		metricsMock := &metrics.MetricsEngineMock{}
		metricsMock.Mock.On("RecordStoredDataFetchTime", metrics.StoredDataLabels{
			DataType:      metrics.RequestDataType,
			DataFetchType: metrics.FetchDelta,
		}, mock.Anything).Return()

		eventProducer := NewPostgresEventProducer(PostgresEventProducerConfig{
			DB:                 db,
			RequestType:        config.RequestDataType,
			CacheUpdateTimeout: 100 * time.Millisecond,
			CacheUpdateQuery:   fakeQuery,
			MetricsEngine:      metricsMock,
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

		metricsMock.AssertExpectations(t)
	}
}

func TestFetchDeltaErrors(t *testing.T) {
	tests := []struct {
		description       string
		giveFakeTime      time.Time
		giveTimeoutMS     int
		giveLastUpdate    time.Time
		giveMockRows      *sqlmock.Rows
		wantRecordedError metrics.StoredDataError
		wantLastUpdate    time.Time
	}{
		{
			description:       "fetch delta timeout",
			giveFakeTime:      time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveLastUpdate:    time.Date(2020, time.June, 30, 6, 0, 0, 0, time.UTC),
			giveMockRows:      nil,
			wantRecordedError: metrics.StoredDataErrorNetwork,
			wantLastUpdate:    time.Date(2020, time.June, 30, 6, 0, 0, 0, time.UTC),
		},
		{
			description:       "fetch delta query error",
			giveFakeTime:      time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveTimeoutMS:     100,
			giveLastUpdate:    time.Date(2020, time.June, 30, 6, 0, 0, 0, time.UTC),
			giveMockRows:      nil,
			wantRecordedError: metrics.StoredDataErrorUndefined,
			wantLastUpdate:    time.Date(2020, time.June, 30, 6, 0, 0, 0, time.UTC),
		},
		{
			description:    "fetch delta row error",
			giveFakeTime:   time.Date(2020, time.July, 1, 12, 30, 0, 0, time.UTC),
			giveTimeoutMS:  100,
			giveLastUpdate: time.Date(2020, time.June, 30, 6, 0, 0, 0, time.UTC),
			giveMockRows: sqlmock.NewRows([]string{"id", "data", "dataType"}).
				AddRow("stored-req-id", "true", "request").
				RowError(0, errors.New("Some row error.")),
			wantRecordedError: metrics.StoredDataErrorUndefined,
			wantLastUpdate:    time.Date(2020, time.June, 30, 6, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		db, dbMock, _ := sqlmock.New()
		if tt.giveMockRows == nil {
			dbMock.ExpectQuery(fakeQueryRegex()).WillReturnError(errors.New("Query failed."))
		} else {
			dbMock.ExpectQuery(fakeQueryRegex()).WillReturnRows(tt.giveMockRows)
		}

		metricsMock := &metrics.MetricsEngineMock{}
		metricsMock.Mock.On("RecordStoredDataFetchTime", metrics.StoredDataLabels{
			DataType:      metrics.RequestDataType,
			DataFetchType: metrics.FetchDelta,
		}, mock.Anything).Return()
		metricsMock.Mock.On("RecordStoredDataError", metrics.StoredDataLabels{
			DataType: metrics.RequestDataType,
			Error:    tt.wantRecordedError,
		}).Return()

		eventProducer := NewPostgresEventProducer(PostgresEventProducerConfig{
			DB:                 db,
			RequestType:        config.RequestDataType,
			CacheUpdateTimeout: time.Duration(tt.giveTimeoutMS) * time.Millisecond,
			CacheUpdateQuery:   fakeQuery,
			MetricsEngine:      metricsMock,
		})
		eventProducer.lastUpdate = tt.giveLastUpdate
		eventProducer.time = &FakeTime{time: tt.giveFakeTime}
		err := eventProducer.Run()

		assert.NotNil(t, err, tt.description)
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

		assert.Nil(t, saves.Requests, tt.description)
		assert.Nil(t, saves.Imps, tt.description)
		assert.Nil(t, invalidations.Requests, tt.description)
		assert.Nil(t, invalidations.Requests, tt.description)

		metricsMock.AssertExpectations(t)
	}
}
