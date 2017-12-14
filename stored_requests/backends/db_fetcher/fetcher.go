package db_fetcher

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/stored_requests/cache/cacher"
)

// dbFetcher fetches Stored Requests from a database. This should be instantiated through the NewPostgres() function.
type dbFetcher struct {
	db         *sql.DB
	queryMaker func(int) (string, error)
	cache      cacher.Cacher
}

func (fetcher *dbFetcher) FetchRequests(ctx context.Context, ids []string) (map[string]json.RawMessage, []error) {
	if len(ids) < 1 {
		return nil, nil
	}

	var errs []error = nil
	reqData := make(map[string]json.RawMessage, len(ids))
	idInterfaces := make([]interface{}, 0)

	for _, id := range ids {
		var err error
		data, err := fetcher.cache.Get(id)
		if err != nil && err != cacher.ErrDoesNotExist {
			// if there is an error then append to the slice.
			// do not append errors that are cache misses
			errs = append(errs, err)
		}
		if data == "" {
			// if empty string then we know we need to look up the id in the database
			idInterfaces = append(idInterfaces, id)
		} else {
			// if its not an empty string then we can assign it to our map
			reqData[id] = []byte(data)
		}
	}

	if len(idInterfaces) > 0 {
		// fetch ids from database if we have cache misses
		reqData, errs = fetcher.fetchRequests(ctx, reqData, errs, idInterfaces)
	}

	for _, id := range ids {
		if _, ok := reqData[id]; !ok {
			errs = append(errs, fmt.Errorf(`Stored Request with ID="%s" not found.`, id))
		}
	}

	return reqData, errs
}

func (fetcher *dbFetcher) fetchRequests(ctx context.Context, reqData map[string]json.RawMessage, errs []error, idInterfaces ...interface{}) (map[string]json.RawMessage, []error) {

	query, err := fetcher.queryMaker(len(idInterfaces))
	if err != nil {
		return nil, []error{err}
	}

	rows, err := fetcher.db.QueryContext(ctx, query, idInterfaces...)
	if err != nil {
		ctxErr := ctx.Err()
		// This query might fail if the user chose an extremely short timeout.
		// We don't care about these... but there may also be legit connection issues.
		// Log any other errors so we have some idea what's going on.
		if ctxErr == nil || ctxErr != context.DeadlineExceeded {
			glog.Errorf("Error reading from Stored Request DB: %s", err.Error())
		}
		return nil, []error{err}
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var thisReqData []byte
		if err := rows.Scan(&id, &thisReqData); err != nil {
			errs = append(errs, err)
		}
		reqData[id] = thisReqData

		if err := fetcher.cache.Set(id, string(thisReqData), cacher.DefaultTTL); err != nil {
			errs = append(errs, err)
		}
	}
	rows.Close() // close rows

	return reqData, errs
}
