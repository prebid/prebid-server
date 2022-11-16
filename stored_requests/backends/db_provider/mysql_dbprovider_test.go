package db_provider

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/stretchr/testify/assert"
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

	connString := provider.ConnString()
	expectedConnString := "someuser:somepassword@tcp(somehost.com:20)/TestDB"
	assert.Equal(t, expectedConnString, connString, "Strings did not match")
}
