package db_fetcher

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// dbFetcher pulls Configs from a database. This should be instantiated through the NewPostgres() function.
type dbFetcher struct {
	db *sql.DB
	queryMaker func(int) (string, error)
}

func (fetcher *dbFetcher) GetConfigs(ids []string) (map[string]json.RawMessage, []error) {
	if len(ids) < 1 {
		return nil, nil
	}

	query, err := fetcher.queryMaker(len(ids))
	if err != nil {
		return nil, []error{err}
	}

	idInterfaces := make([]interface{}, len(ids))
	for i := 0; i < len(ids); i++ {
		idInterfaces[i] = ids[i]
	}

	rows, err := fetcher.db.Query(query, idInterfaces...)
	if err != nil {
		return nil, []error{err}
	}
	defer rows.Close()

	configs := make(map[string]json.RawMessage, len(ids))
	var errs []error = nil
	for rows.Next() {
		var id string
		var configData []byte
		if err := rows.Scan(&id, &configData); err != nil {
			errs = append(errs, err)
		}

		configCopy := make([]byte, len(configData))
		copy(configCopy, configData)
		configs[id] = json.RawMessage(configCopy)
	}

	for _, id := range ids {
		if _, ok := configs[id]; !ok {
			errs = append(errs, fmt.Errorf("Config ID not found: %s", id))
		}
	}

	return configs, errs
}
