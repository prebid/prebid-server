package db_fetcher

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/lib/pq"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/stored_requests"
)

func NewFetcher(db *sql.DB, queryMaker func(int, int) string) stored_requests.AllFetcher {
	if db == nil {
		glog.Fatalf("The Postgres Stored Request Fetcher requires a database connection. Please report this as a bug.")
	}
	if queryMaker == nil {
		glog.Fatalf("The Postgres Stored Request Fetcher requires a queryMaker function. Please report this as a bug.")
	}
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

func (fetcher *dbFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (map[string]json.RawMessage, map[string]json.RawMessage, []error) {
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
		if err != context.DeadlineExceeded && !isBadInput(err) {
			glog.Errorf("Error reading from Stored Request DB: %s", err.Error())
			errs := appendErrors("Request", requestIDs, nil, nil)
			errs = appendErrors("Imp", impIDs, nil, errs)
			return nil, nil, errs
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

	errs := appendErrors("Request", requestIDs, storedRequestData, nil)
	errs = appendErrors("Imp", impIDs, storedImpData, errs)

	return storedRequestData, storedImpData, errs
}

func (fetcher *dbFetcher) FetchAccount(ctx context.Context, accountID string) (json.RawMessage, []error) {
	return nil, []error{stored_requests.NotFoundError{accountID, "Account"}}
}

func (fetcher *dbFetcher) FetchCategories(ctx context.Context, primaryAdServer, publisherId, iabCategory string) (string, error) {
	return "", nil
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

// Returns true if the Postgres error signifies some sort of bad user input, and false otherwise.
//
// These errors are documented here: https://www.postgresql.org/docs/9.3/static/errcodes-appendix.html
func isBadInput(err error) bool {
	// Unfortunately, Postgres queries will fail if a non-UUID is passed into a query for a UUID column. For example:
	//
	//    SELECT uuid, data, dataType FROM stored_requests WHERE uuid IN ('abc');
	//
	// Since users can send us strings which are _not_ UUIDs, and we don't want the code to assume anything about
	// the database schema, we can just convert these into standard NotFoundErrors.
	if pqErr, ok := err.(*pq.Error); ok && string(pqErr.Code) == "22P02" {
		return true
	}

	return false
}
