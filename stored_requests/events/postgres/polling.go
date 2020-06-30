package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/PubMatic-OpenWrap/prebid-server/stored_requests/events"
	"github.com/golang/glog"
)

// PollForUpdates returns an EventProducer which checks the database for updates every refreshRate.
//
// This object will prioritize thoroughness over efficiency. In rare cases it may produce two "update" events for
// the same DB save, but it should never "miss" a database update either.
//
// The Queries should return a ResultSet with the following columns and types:
//
//   1. id: string
//   2. data: JSON
//   3. type: string ("request" or "imp")
//
// If data is empty or the JSON "null", then the ID will be invalidated (e.g. a deletion).
// If data is not empty, it should be the Stored Request or Stored Imp data associated with the given ID.
func PollForUpdates(ctxProducer func() (ctx context.Context, canceller func()), db *sql.DB, query string, startUpdatesFrom time.Time, refreshRate time.Duration) (eventProducer *PostgresPoller) {
	// If we're not given a function to produce Contexts, use the Background one.
	if ctxProducer == nil {
		ctxProducer = func() (ctx context.Context, canceller func()) {
			return context.Background(), func() {}
		}
	}
	if db == nil {
		glog.Fatal("The Stored Request Postgres Poller needs a database connection to work.")
	}

	e := &PostgresPoller{
		db:            db,
		ctxProducer:   ctxProducer,
		updateQuery:   query,
		lastUpdate:    startUpdatesFrom,
		invalidations: make(chan events.Invalidation, 1),
		saves:         make(chan events.Save, 1),
	}

	glog.Infof("Stored Requests will be refreshed from Postgres every %f seconds with: %s", refreshRate.Seconds(), query)

	if refreshRate > 0 {
		go e.refresh(time.Tick(refreshRate))
	} else {
		glog.Warningf("Postgres Stored Event polling refreshRate was %d. This must be positive. No updates will occur.", refreshRate)
	}
	return e
}

type PostgresPoller struct {
	db            *sql.DB
	ctxProducer   func() (ctx context.Context, canceller func())
	updateQuery   string
	lastUpdate    time.Time
	invalidations chan events.Invalidation
	saves         chan events.Save
}

func (e *PostgresPoller) refresh(ticker <-chan time.Time) {
	for {
		select {
		case thisTime := <-ticker:
			// Make sure to log the time now, *before* running the query,
			// so that next tick's query won't miss any new updates which were made at the same time.
			// This may duplicate some updates, but safety > efficiency.
			thisTimeInUTC := thisTime.UTC()
			ctx, cancel := e.ctxProducer()
			rows, err := e.db.QueryContext(ctx, e.updateQuery, e.lastUpdate)
			if err != nil {
				glog.Warningf("Failed to update Stored Request data: %v", err)
				cancel()
				continue
			}
			if err := sendEvents(rows, e.saves, e.invalidations); err != nil {
				glog.Warningf("Failed to update Stored Request data: %v", err)
			} else {
				e.lastUpdate = thisTimeInUTC
			}
			if err := rows.Close(); err != nil {
				glog.Warningf("Failed to close DB connection: %v", err)
			}
			cancel()
		}
	}
}

// sendEvents reads the rows and sends notifications into the channel for any updates.
// If it returns an error, then callers can be certain that no events were sent to the channels.
func sendEvents(rows *sql.Rows, saves chan<- events.Save, invalidations chan<- events.Invalidation) (err error) {
	storedRequestData := make(map[string]json.RawMessage)
	storedImpData := make(map[string]json.RawMessage)

	var requestInvalidations []string
	var impInvalidations []string

	for rows.Next() {
		var id string
		var data []byte
		var dataType string
		// Beware #338... we really don't want to save corrupt data
		if err := rows.Scan(&id, &data, &dataType); err != nil {
			return err
		}

		switch dataType {
		case "request":
			if len(data) == 0 || bytes.Equal(data, []byte("null")) {
				requestInvalidations = append(requestInvalidations, id)
			} else {
				storedRequestData[id] = data
			}
		case "imp":
			if len(data) == 0 || bytes.Equal(data, []byte("null")) {
				impInvalidations = append(impInvalidations, id)
			} else {
				storedImpData[id] = data
			}
		default:
			glog.Warningf("Stored Data with id=%s has invalid type: %s. This will be ignored.", id, dataType)
		}
	}

	// Beware #338... we really don't want to save corrupt data
	if rows.Err() != nil {
		return rows.Err()
	}

	if len(storedRequestData) > 0 || len(storedImpData) > 0 && saves != nil {
		saves <- events.Save{
			Requests: storedRequestData,
			Imps:     storedImpData,
		}
	}

	// There shouldn't be any invalidations with a nil channel (a "startup" query),
	// but... if there are, we certainly don't want to block forever.
	if len(requestInvalidations) > 0 || len(impInvalidations) > 0 && invalidations != nil {
		invalidations <- events.Invalidation{
			Requests: requestInvalidations,
			Imps:     impInvalidations,
		}
	}

	return nil
}

func (e *PostgresPoller) Saves() <-chan events.Save {
	return e.saves
}

func (e *PostgresPoller) Invalidations() <-chan events.Invalidation {
	return e.invalidations
}
