package config

import (
	"strconv"
	"strings"
	"testing"
)

const sampleQueryTemplate = "SELECT id, requestData, 'request' as type FROM stored_requests WHERE id in %REQUEST_ID_LIST% UNION ALL SELECT id, impData, 'imp' as type FROM stored_requests WHERE id in %IMP_ID_LIST%"

func TestNormalQueryMaker(t *testing.T) {
	madeQuery := buildQuery(sampleQueryTemplate, 1, 3)
	assertStringsEqual(t, madeQuery, "SELECT id, requestData, 'request' as type FROM stored_requests WHERE id in ($1) UNION ALL SELECT id, impData, 'imp' as type FROM stored_requests WHERE id in ($2, $3, $4)")
}
func TestQueryMakerManyImps(t *testing.T) {
	madeQuery := buildQuery(sampleQueryTemplate, 1, 11)
	assertStringsEqual(t, madeQuery, "SELECT id, requestData, 'request' as type FROM stored_requests WHERE id in ($1) UNION ALL SELECT id, impData, 'imp' as type FROM stored_requests WHERE id in ($2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)")
}

func TestQueryMakerNoRequests(t *testing.T) {
	madeQuery := buildQuery(sampleQueryTemplate, 0, 3)
	assertStringsEqual(t, madeQuery, "SELECT id, requestData, 'request' as type FROM stored_requests WHERE id in (NULL) UNION ALL SELECT id, impData, 'imp' as type FROM stored_requests WHERE id in ($1, $2, $3)")
}

func TestQueryMakerNoImps(t *testing.T) {
	madeQuery := buildQuery(sampleQueryTemplate, 1, 0)
	assertStringsEqual(t, madeQuery, "SELECT id, requestData, 'request' as type FROM stored_requests WHERE id in ($1) UNION ALL SELECT id, impData, 'imp' as type FROM stored_requests WHERE id in (NULL)")
}

func TestQueryMakerMultilists(t *testing.T) {
	madeQuery := buildQuery("SELECT id, config FROM table WHERE id in %IMP_ID_LIST% UNION ALL SELECT id, config FROM other_table WHERE id in %IMP_ID_LIST%", 0, 3)
	assertStringsEqual(t, madeQuery, "SELECT id, config FROM table WHERE id in ($1, $2, $3) UNION ALL SELECT id, config FROM other_table WHERE id in ($1, $2, $3)")
}

func TestQueryMakerNegative(t *testing.T) {
	query := buildQuery(sampleQueryTemplate, -1, -2)
	expected := buildQuery(sampleQueryTemplate, 0, 0)
	assertStringsEqual(t, query, expected)
}

func TestPostgressConnString(t *testing.T) {
	db := "TestDB"
	host := "somehost.com"
	port := 20
	username := "someuser"
	password := "somepassword"

	cfg := PostgresConnection{
		Database: db,
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}

	dataSourceName := cfg.ConnString()
	paramList := strings.Split(dataSourceName, " ")
	params := make(map[string]string, len(paramList))
	for _, param := range paramList {
		keyVals := strings.Split(param, "=")
		if len(keyVals) != 2 {
			t.Fatalf(`param "%s" must only have one equals sign`, param)
		}
		if _, ok := params[keyVals[0]]; ok {
			t.Fatalf("found duplicate param at key %s", keyVals[0])
		}
		params[keyVals[0]] = keyVals[1]
	}

	assertHasValue(t, params, "dbname", db)
	assertHasValue(t, params, "host", host)
	assertHasValue(t, params, "port", strconv.Itoa(port))
	assertHasValue(t, params, "user", username)
	assertHasValue(t, params, "password", password)
	assertHasValue(t, params, "sslmode", "disable")
}

func TestEventChannels(t *testing.T) {
	validConfig := PostgresEventsChannels{
		OpenRTBRequestUpdates: "request-updates",
		OpenRTBRequestDeletes: "request-deletes",
		OpenRTBImpUpdates:     "imp-updates",
		OpenRTBImpDeletes:     "imp-deletes",
		AMPRequestUpdates:     "amp-request-updates",
		AMPRequestDeletes:     "amp-imp-deletes",
	}

	assertError(t, validConfig, func(cfg PostgresEventsChannels) *PostgresEventsChannels {
		cfg.OpenRTBRequestUpdates = ""
		return &cfg
	})
	assertError(t, validConfig, func(cfg PostgresEventsChannels) *PostgresEventsChannels {
		cfg.OpenRTBRequestDeletes = ""
		return &cfg
	})
	assertError(t, validConfig, func(cfg PostgresEventsChannels) *PostgresEventsChannels {
		cfg.OpenRTBImpUpdates = ""
		return &cfg
	})
	assertError(t, validConfig, func(cfg PostgresEventsChannels) *PostgresEventsChannels {
		cfg.OpenRTBImpDeletes = ""
		return &cfg
	})
	assertError(t, validConfig, func(cfg PostgresEventsChannels) *PostgresEventsChannels {
		cfg.AMPRequestUpdates = ""
		return &cfg
	})
	assertError(t, validConfig, func(cfg PostgresEventsChannels) *PostgresEventsChannels {
		cfg.AMPRequestDeletes = ""
		return &cfg
	})
}

func assertError(t *testing.T, cfg PostgresEventsChannels, transform func(PostgresEventsChannels) *PostgresEventsChannels) {
	t.Helper()
	if err := transform(cfg).validate(); err == nil {
		t.Errorf("config should not be valid: %v", cfg)
	}
}

func assertHasValue(t *testing.T, m map[string]string, key string, val string) {
	t.Helper()
	realVal, ok := m[key]
	if !ok {
		t.Errorf("Map missing required key: %s", key)
	}
	if val != realVal {
		t.Errorf("Unexpected value at key %s. Expected %s, Got %s", key, val, realVal)
	}
}

func buildQuery(template string, numReqs int, numImps int) string {
	cfg := PostgresQueries{
		QueryTemplate: template,
	}
	return cfg.MakeQuery(numReqs, numImps)
}

func assertStringsEqual(t *testing.T, actual string, expected string) {
	if actual != expected {
		t.Errorf("Queries did not match.\n\"%s\" -- expected\n\"%s\" -- actual", expected, actual)

	}
}
