package db_provider

import (
	"testing"

	"github.com/prebid/prebid-server/config"
)

func TestConnStringMySql(t *testing.T) {
	driver := "mysql"
	db := "TestDB"
	host := "somehost.com"
	port := 20
	username := "someuser"
	password := "somepassword"

	cfg := config.DatabaseConnection{
		Driver:   driver,
		Database: db,
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}
	provider := MySqlDbProvider{
		cfg: cfg,
	}

	dataSourceName := provider.ConnString()
	assertStringsEqual(t, dataSourceName, "someuser:somepassword@tcp(somehost.com:20)/TestDB")
}

func assertStringsEqual(t *testing.T, actual string, expected string) {
	if actual != expected {
		t.Errorf("Strings did not match.\n\"%s\" -- expected\n\"%s\" -- actual", expected, actual)
	}
}
