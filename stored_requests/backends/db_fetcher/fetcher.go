package db_fetcher

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/stored_requests"
)

func NewFetcher(db *sql.DB, queryMaker func(int, int) string) stored_requests.Fetcher {
	return &dbFetcher{
		db:         db,
		queryMaker: queryMaker,
	}
}

// dbFetcher fetches Stored Requests from a database. This should be instantiated through the NewFetcher() function.
type dbFetcher struct {
	db         *sql.DB
	queryMaker func(numReqs int, numImps int) (query string)
}

func (fetcher *dbFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	if len(requestIDs) < 1 && len(impIDs) < 1 {
		return nil, nil, nil
	}

	query := fetcher.queryMaker(len(requestIDs), len(impIDs))
	idInterfaces := make([]interface{}, len(requestIDs)+len(impIDs))
	for i := 0; i < len(requestIDs); i++ {
		idInterfaces[i] = requestIDs[i]
	}
	for i := 0; i < len(impIDs); i++ {
		idInterfaces[i+len(requestIDs)] = impIDs[i]
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
		return nil, nil, []error{err}
	}
	defer func() {
		if err := rows.Close(); err != nil {
			glog.Errorf("error closing DB connection: %v", err)
		}
	}()

	storedRequestData := make(map[string]json.RawMessage, len(requestIDs))
	storedImpData := make(map[string]json.RawMessage, len(impIDs))
	for rows.Next() {
		var id string
		var data []byte
		var dataType string

		// Fixes #338
		if err := rows.Scan(&id, &data, &dataType); err != nil {
			return nil, nil, []error{err}
		}

		switch dataType {
		case "request":
			storedRequestData[id] = data
		case "imp":
			storedImpData[id] = data
		default:
			glog.Errorf("Postgres result set with id=%s has invalid type: %s. This will be ignored.", id, dataType)
		}
	}

	// Fixes #338
	if rows.Err() != nil {
		return nil, nil, []error{rows.Err()}
	}

	errs = appendErrors("Request", requestIDs, storedRequestData, nil)
	errs = appendErrors("Imp", impIDs, storedImpData, errs)

	return storedRequestData, storedImpData, errs
}

func appendErrors(dataType string, ids []string, data map[string]json.RawMessage, errs []error) []error {
	for _, id := range ids {
		if _, ok := data[id]; !ok {
			errs = append(errs, stored_requests.NotFoundError{
				ID:       id,
				DataType: dataType,
			})
		}
	}
	return errs
}
