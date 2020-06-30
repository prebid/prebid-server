package postgres

import (
	"context"
	"database/sql"

	"github.com/PubMatic-OpenWrap/prebid-server/stored_requests/events"
	"github.com/golang/glog"
)

// This function queries the database to get all the data, and is guaranteed to return
// an EventProducer with a single "events.Save" object already in the channel before returning.
//
// The string query should return Rows with the following columns and types:
//
//   1. id: string
//   2. data: JSON
//   3. type: string ("request" or "imp")
//
func LoadAll(ctx context.Context, db *sql.DB, query string) (eventProducer *PostgresLoader) {
	if db == nil {
		glog.Fatal("The Stored Request Postgres Startup needs a database connection to work.")
	}
	eventProducer = &PostgresLoader{
		saves: make(chan events.Save, 1),
	}
	eventProducer.doFetch(ctx, db, query)
	return
}

type PostgresLoader struct {
	saves chan events.Save
}

func (loader *PostgresLoader) doFetch(ctx context.Context, db *sql.DB, query string) {
	glog.Infof("Loading all Stored Requests from Postgres with: %s", query)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		glog.Warningf("Failed to fetch Stored Requests from Postgres on startup. The app might be a bit slow to start. Error was: %v", err)
		loader.saves <- events.Save{}
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			glog.Warningf("Failed to close DB connection: %v", err)
		}
	}()

	if err := sendEvents(rows, loader.saves, nil); err != nil {
		glog.Warningf("Failed to fetch Stored Requests from Postgres on startup. Things might be a bit slow to start: %v", err)
		loader.saves <- events.Save{}
	}
}

func (e *PostgresLoader) Saves() <-chan events.Save {
	return e.saves
}

func (e *PostgresLoader) Invalidations() <-chan events.Invalidation {
	return nil
}
