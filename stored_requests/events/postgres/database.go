package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests/events"
	"github.com/prebid/prebid-server/util/timeutil"
)

func bytesNull() []byte {
	return []byte{'n', 'u', 'l', 'l'}
}

type PostgresEventProducerConfig struct {
	DB                 *sql.DB
	RequestType        config.DataType
	CacheInitQuery     string
	CacheInitTimeout   time.Duration
	CacheUpdateQuery   string
	CacheUpdateTimeout time.Duration
}

type PostgresEventProducer struct {
	cfg           PostgresEventProducerConfig
	lastUpdate    time.Time
	invalidations chan events.Invalidation
	saves         chan events.Save
	time          timeutil.Time
}

func NewPostgresEventProducer(cfg PostgresEventProducerConfig) (eventProducer *PostgresEventProducer) {
	if cfg.DB == nil {
		glog.Fatalf("The Postgres Stored %s Loader needs a database connection to work.", cfg.RequestType)
	}

	return &PostgresEventProducer{
		cfg:           cfg,
		lastUpdate:    time.Time{},
		saves:         make(chan events.Save, 1),
		invalidations: make(chan events.Invalidation, 1),
		time:          &timeutil.RealTime{},
	}
}

func (e *PostgresEventProducer) Run() error {
	if e.lastUpdate.IsZero() {
		return e.fetchAll()
	} else {
		return e.fetchDelta()
	}
}

func (e *PostgresEventProducer) Saves() <-chan events.Save {
	return e.saves
}

func (e *PostgresEventProducer) Invalidations() <-chan events.Invalidation {
	return e.invalidations
}

func (e *PostgresEventProducer) fetchAll() error {
	thisTimeInUTC := e.time.Now().UTC()

	timeout := e.cfg.CacheInitTimeout * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	rows, err := e.cfg.DB.QueryContext(ctx, e.cfg.CacheInitQuery)

	if err != nil {
		glog.Warningf("Failed to fetch all Stored %s data from the DB: %v", e.cfg.RequestType, err)
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			glog.Warningf("Failed to close the Stored %s DB connection: %v", e.cfg.RequestType, err)
		}
	}()
	if err := e.sendEvents(rows); err != nil {
		glog.Warningf("Failed to load all Stored %s data from the DB: %v", e.cfg.RequestType, err)
		return err
	} else {
		e.lastUpdate = thisTimeInUTC
	}
	return nil
}

func (e *PostgresEventProducer) fetchDelta() error {
	thisTimeInUTC := e.time.Now().UTC()

	timeout := e.cfg.CacheUpdateTimeout * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	rows, err := e.cfg.DB.QueryContext(ctx, e.cfg.CacheUpdateQuery, e.lastUpdate)

	if err != nil {
		glog.Warningf("Failed to fetch updated Stored %s data from the DB: %v", e.cfg.RequestType, err)
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			glog.Warningf("Failed to close the Stored %s DB connection: %v", e.cfg.RequestType, err)
		}
	}()
	if err := e.sendEvents(rows); err != nil {
		glog.Warningf("Failed to load updated Stored %s data from the DB: %v", e.cfg.RequestType, err)
		return err
	} else {
		e.lastUpdate = thisTimeInUTC
	}
	return nil
}

// sendEvents reads the rows and sends notifications into the channel for any updates.
// If it returns an error, then callers can be certain that no events were sent to the channels.
func (e *PostgresEventProducer) sendEvents(rows *sql.Rows) (err error) {
	storedRequestData := make(map[string]json.RawMessage)
	storedImpData := make(map[string]json.RawMessage)

	var requestInvalidations []string
	var impInvalidations []string

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
		default:
			glog.Warningf("Stored Data with id=%s has invalid type: %s. This will be ignored.", id, dataType)
		}
	}

	// discard corrupted data so it is not saved in the cache
	if rows.Err() != nil {
		return rows.Err()
	}

	if len(storedRequestData) > 0 || len(storedImpData) > 0 {
		e.saves <- events.Save{
			Requests: storedRequestData,
			Imps:     storedImpData,
		}
	}

	if (len(requestInvalidations) > 0 || len(impInvalidations) > 0) && !e.lastUpdate.IsZero() {
		e.invalidations <- events.Invalidation{
			Requests: requestInvalidations,
			Imps:     impInvalidations,
		}
	}

	return
}
