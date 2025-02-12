package database

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/stored_requests/backends/db_provider"
	"github.com/prebid/prebid-server/v3/stored_requests/events"
	"github.com/prebid/prebid-server/v3/util/timeutil"
)

func bytesNull() []byte {
	return []byte{'n', 'u', 'l', 'l'}
}

var storedDataTypeMetricMap = map[config.DataType]metrics.StoredDataType{
	config.RequestDataType:    metrics.RequestDataType,
	config.CategoryDataType:   metrics.CategoryDataType,
	config.VideoDataType:      metrics.VideoDataType,
	config.AMPRequestDataType: metrics.AMPDataType,
	config.AccountDataType:    metrics.AccountDataType,
	config.ResponseDataType:   metrics.ResponseDataType,
}

type DatabaseEventProducerConfig struct {
	Provider           db_provider.DbProvider
	RequestType        config.DataType
	CacheInitQuery     string
	CacheInitTimeout   time.Duration
	CacheUpdateQuery   string
	CacheUpdateTimeout time.Duration
	MetricsEngine      metrics.MetricsEngine
}

type DatabaseEventProducer struct {
	cfg           DatabaseEventProducerConfig
	lastUpdate    time.Time
	invalidations chan events.Invalidation
	saves         chan events.Save
	time          timeutil.Time
}

func NewDatabaseEventProducer(cfg DatabaseEventProducerConfig) (eventProducer *DatabaseEventProducer) {
	if cfg.Provider == nil {
		glog.Fatalf("The Database Stored %s Loader needs a database connection to work.", cfg.RequestType)
	}

	return &DatabaseEventProducer{
		cfg:           cfg,
		lastUpdate:    time.Time{},
		saves:         make(chan events.Save, 1),
		invalidations: make(chan events.Invalidation, 1),
		time:          &timeutil.RealTime{},
	}
}

func (e *DatabaseEventProducer) Run() error {
	if e.lastUpdate.IsZero() {
		return e.fetchAll()
	}

	return e.fetchDelta()
}

func (e *DatabaseEventProducer) Saves() <-chan events.Save {
	return e.saves
}

func (e *DatabaseEventProducer) Invalidations() <-chan events.Invalidation {
	return e.invalidations
}

func (e *DatabaseEventProducer) fetchAll() (fetchErr error) {
	timeout := e.cfg.CacheInitTimeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	startTime := e.time.Now().UTC()
	rows, err := e.cfg.Provider.QueryContext(ctx, e.cfg.CacheInitQuery)
	elapsedTime := time.Since(startTime)
	e.recordFetchTime(elapsedTime, metrics.FetchAll)

	if err != nil {
		glog.Warningf("Failed to fetch all Stored %s data from the DB: %v", e.cfg.RequestType, err)
		if _, ok := err.(net.Error); ok {
			e.recordError(metrics.StoredDataErrorNetwork)
		} else {
			e.recordError(metrics.StoredDataErrorUndefined)
		}
		return err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			glog.Warningf("Failed to close the Stored %s DB connection: %v", e.cfg.RequestType, err)
			e.recordError(metrics.StoredDataErrorUndefined)
			fetchErr = err
		}
	}()
	if err := e.sendEvents(rows); err != nil {
		glog.Warningf("Failed to load all Stored %s data from the DB: %v", e.cfg.RequestType, err)
		e.recordError(metrics.StoredDataErrorUndefined)
		return err
	}

	e.lastUpdate = startTime
	return nil
}

func (e *DatabaseEventProducer) fetchDelta() (fetchErr error) {
	timeout := e.cfg.CacheUpdateTimeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	startTime := e.time.Now().UTC()

	params := []db_provider.QueryParam{
		{Name: "LAST_UPDATED", Value: e.lastUpdate},
	}

	rows, err := e.cfg.Provider.QueryContext(ctx, e.cfg.CacheUpdateQuery, params...)
	elapsedTime := time.Since(startTime)
	e.recordFetchTime(elapsedTime, metrics.FetchDelta)

	if err != nil {
		glog.Warningf("Failed to fetch updated Stored %s data from the DB: %v", e.cfg.RequestType, err)
		if _, ok := err.(net.Error); ok {
			e.recordError(metrics.StoredDataErrorNetwork)
		} else {
			e.recordError(metrics.StoredDataErrorUndefined)
		}
		return err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			glog.Warningf("Failed to close the Stored %s DB connection: %v", e.cfg.RequestType, err)
			e.recordError(metrics.StoredDataErrorUndefined)
			fetchErr = err
		}
	}()
	if err := e.sendEvents(rows); err != nil {
		glog.Warningf("Failed to load updated Stored %s data from the DB: %v", e.cfg.RequestType, err)
		e.recordError(metrics.StoredDataErrorUndefined)
		return err
	}

	e.lastUpdate = startTime
	return nil
}

func (e *DatabaseEventProducer) recordFetchTime(elapsedTime time.Duration, fetchType metrics.StoredDataFetchType) {
	e.cfg.MetricsEngine.RecordStoredDataFetchTime(
		metrics.StoredDataLabels{
			DataType:      storedDataTypeMetricMap[e.cfg.RequestType],
			DataFetchType: fetchType,
		}, elapsedTime)
}

func (e *DatabaseEventProducer) recordError(errorType metrics.StoredDataError) {
	e.cfg.MetricsEngine.RecordStoredDataError(
		metrics.StoredDataLabels{
			DataType: storedDataTypeMetricMap[e.cfg.RequestType],
			Error:    errorType,
		})
}

// sendEvents reads the rows and sends notifications into the channel for any updates.
// If it returns an error, then callers can be certain that no events were sent to the channels.
func (e *DatabaseEventProducer) sendEvents(rows *sql.Rows) (err error) {
	storedRequestData := make(map[string]json.RawMessage)
	storedImpData := make(map[string]json.RawMessage)
	storedRespData := make(map[string]json.RawMessage)

	var requestInvalidations []string
	var impInvalidations []string
	var respInvalidations []string

	for rows.Next() {
		var id string
		var data []byte
		var dataType string

		// discard corrupted data so it is not saved in the cache
		if err := rows.Scan(&id, &data, &dataType); err != nil {
			return err
		}

		switch dataType {
		case "request":
			if len(data) == 0 || bytes.Equal(data, bytesNull()) {
				requestInvalidations = append(requestInvalidations, id)
			} else {
				storedRequestData[id] = data
			}
		case "imp":
			if len(data) == 0 || bytes.Equal(data, bytesNull()) {
				impInvalidations = append(impInvalidations, id)
			} else {
				storedImpData[id] = data
			}
		case "response":
			if len(data) == 0 || bytes.Equal(data, bytesNull()) {
				respInvalidations = append(respInvalidations, id)
			} else {
				storedRespData[id] = data
			}
		default:
			glog.Warningf("Stored Data with id=%s has invalid type: %s. This will be ignored.", id, dataType)
		}
	}

	// discard corrupted data so it is not saved in the cache
	if rows.Err() != nil {
		return rows.Err()
	}

	if len(storedRequestData) > 0 || len(storedImpData) > 0 || len(storedRespData) > 0 {
		e.saves <- events.Save{
			Requests:  storedRequestData,
			Imps:      storedImpData,
			Responses: storedRespData,
		}
	}

	if (len(requestInvalidations) > 0 || len(impInvalidations) > 0 || len(respInvalidations) > 0) && !e.lastUpdate.IsZero() {
		e.invalidations <- events.Invalidation{
			Requests:  requestInvalidations,
			Imps:      impInvalidations,
			Responses: respInvalidations,
		}
	}

	return
}
