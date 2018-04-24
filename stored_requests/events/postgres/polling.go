package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/buger/jsonparser"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/stored_requests/events"
)

// This EventProducer queries the database on startup to get all the data, and then polls it periodically.
//
// This may be preferred to notifications.go if you use read slaves to help distribute the load,
// since Triggers require everything to use the master database.
//
// Like the Fetchers, the SQL query used to fetch data and updates can be set in the app config.
//
// The Queries should return a ResultSet with the following columns and types:
//
//   1. id: string
//   2. data: JSON
//   3. type: string ("request" or "imp")
//
// If data is null, then the ID will be invalidated (e.g. a deletion).
// If present, it should be the Stored Request or Stored Imp data associated with the given ID.
func PollDatabase(ctxProducer func() (ctx context.Context, canceller func()), db *sql.DB, loadAllQuery string, updateQuery string, refreshRate time.Duration) (eventProducer events.EventProducer) {
	// If we're not given a function to produce Contexts, use the Background one.
	if ctxProducer == nil {
		ctxProducer = func() (ctx context.Context, canceller func()) {
			return context.Background(), func() {}
		}
	}
	if db == nil {
		glog.Fatal("The Stored Request Postgres Poller needs a database connection to work.")
	}

	e := &dbPoller{
		db:            db,
		ctxProducer:   ctxProducer,
		loadAllQuery:  loadAllQuery,
		updateQuery:   updateQuery,
		lastUpdate:    time.Now().UTC(),
		invalidations: make(chan events.Invalidation, 1),
		saves:         make(chan events.Save, 1),
	}
	glog.Infof("Stored Requests will be loaded from Postgres initially with: %s", loadAllQuery)

	if err := e.fetchAll(); err != nil {
		glog.Warningf("Failed to fetch Stored Requests from Postgres on startup. Things might be a bit slow to start: %v", err)
	}

	glog.Infof("Stored Requests will be refreshed from Postgres every %f seconds with: %s", refreshRate.Seconds(), updateQuery)

	go e.refresh(time.Tick(refreshRate))
	return e
}

type dbPoller struct {
	db            *sql.DB
	ctxProducer   func() (ctx context.Context, canceller func())
	loadAllQuery  string
	updateQuery   string
	lastUpdate    time.Time
	invalidations chan events.Invalidation
	saves         chan events.Save
}

func (e *dbPoller) fetchAll() error {
	ctx, cancel := e.ctxProducer()
	defer cancel()

	rows, err := e.db.QueryContext(ctx, e.loadAllQuery)
	if err != nil {
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			glog.Errorf("error closing DB connection: %v", err)
		}
	}()

	return e.sendEvents(rows)
}

func (e *dbPoller) refresh(ticker <-chan time.Time) {
	for {
		select {
		case thisTime := <-ticker:
			thisTimeInUTC := thisTime.UTC()
			ctx, cancel := e.ctxProducer()
			rows, err := e.db.QueryContext(ctx, e.updateQuery, e.lastUpdate)
			if err != nil {
				glog.Errorf("Failed to update Stored Request data from Postgres: %v", err)
				cancel()
				continue
			}
			if err := e.sendEvents(rows); err != nil {
				glog.Errorf("Failed to update Stored Request data from Postgres: %v", err)
			} else {
				e.lastUpdate = thisTimeInUTC
			}
			if err := rows.Close(); err != nil {
				glog.Errorf("error closing DB connection: %v", err)
			}
			cancel()
		}
	}
}

// sendEvents reads the rows and sends notifications into the channel for any updates
func (e *dbPoller) sendEvents(rows *sql.Rows) (err error) {
	storedRequestData := make(map[string]json.RawMessage)
	storedImpData := make(map[string]json.RawMessage)

	var requestInvalidations []string
	var impInvalidations []string

	for rows.Next() {
		var id string
		var data []byte
		var dataType string
		// Beware #338... we don't want to save corrupt data
		if err := rows.Scan(&id, &data, &dataType); err != nil {
			return err
		}

		// We shouldn't get any "nulls" on this startup query, but... just in case, make sure not to save them.
		if len(data) > 0 {
			switch dataType {
			case "request":
				if shouldDelete, err := isDeletion(id, "Request", data); err == nil {
					if shouldDelete {
						requestInvalidations = append(requestInvalidations, id)
					} else {
						storedRequestData[id] = data
					}
				}
			case "imp":
				if shouldDelete, err := isDeletion(id, "Imp", data); err == nil {
					if shouldDelete {
						impInvalidations = append(impInvalidations, id)
					} else {
						storedImpData[id] = data
					}
				}
			default:
				glog.Errorf("Postgres result set with id=%s has invalid type: %s. This will be ignored.", id, dataType)
			}
		}
	}

	// Beware #338... we don't want to save corrupt data
	if rows.Err() != nil {
		return rows.Err()
	}

	if len(storedRequestData) > 0 || len(storedImpData) > 0 {
		e.saves <- events.Save{
			Requests: storedRequestData,
			Imps:     storedImpData,
		}
	}

	if len(requestInvalidations) > 0 || len(impInvalidations) > 0 {
		e.invalidations <- events.Invalidation{
			Requests: requestInvalidations,
			Imps:     impInvalidations,
		}
	}

	return nil
}

func (e *dbPoller) Saves() <-chan events.Save {
	return e.saves
}

func (e *dbPoller) Invalidations() <-chan events.Invalidation {
	return e.invalidations
}

func isDeletion(id string, dataType string, data json.RawMessage) (bool, error) {
	if value, _, _, err := jsonparser.Get(data, "deleted"); err == nil {
		if bytes.Equal(value, []byte("true")) {
			return true, nil
		} else {
			return false, nil
		}
	} else if err != jsonparser.KeyPathNotFoundError {
		glog.Errorf("Postgres Stored %s %s has bad data %s.", dataType, id, string(data))
		return false, err
	}
	return false, nil
}
