package db_fetcher

import (
	"github.com/prebid/prebid-server/config"
	"strconv"
	"strings"
	"testing"
)

// TestDSNCreation makes sure we turn the config into a string expected by the Postgres driver library.
func TestDSNCreation(t *testing.T) {
	db := "TestDB"
	host := "somehost.com"
	port := 20
	username := "someuser"
	password := "somepassword"
	query := "SELECT id, config FROM table WHERE id in %ID_LIST%"

	cfg := &config.PostgresConfig{
		Database:      db,
		Host:          host,
		Port:          port,
		Username:      username,
		Password:      password,
		QueryTemplate: query,
	}

	dataSourceName := confToPostgresDSN(cfg)
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
