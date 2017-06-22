package postgrescache

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/coocood/freecache"
	"github.com/erikstmartin/go-testdb"
	"github.com/golang/glog"
	"github.com/stretchr/testify/assert"
)

func TestPostgresConfig(t *testing.T) {
	conf := PostgresConfig{
		Host:     "host",
		Port:     1234,
		Dbname:   "dbname",
		User:     "user",
		Password: "password",
		TTL:      3434,
		Size:     100,
	}

	u := conf.uri()
	assert.True(t, strings.Contains(u, "host=host"))
	assert.True(t, strings.Contains(u, "port=1234"))
	assert.True(t, strings.Contains(u, "dbname=dbname"))
	assert.True(t, strings.Contains(u, "user=user"))
	assert.True(t, strings.Contains(u, "password=password"))
}

type StubCache struct {
	shared *shared

	accounts *accountService
	domains  *domainService
	apps     *appsService
	config   *configService
}

// New creates new postgres.Cache
func StubNew(cfg PostgresConfig) (*Cache) {
	shared := stubnewShared(cfg)
	return &Cache{
		shared:   shared,
		accounts: &accountService{shared: shared},
		domains:  &domainService{shared: shared},
		apps:     &appsService{shared: shared},
		config:   &configService{shared: shared},
	}
}

func stubnewShared(conf PostgresConfig) (*shared) {
	db, _ := sql.Open("testdb", "")

	s := &shared{
		db:         db,
		lru:        freecache.NewCache(conf.Size),
		ttlSeconds: 0,
	}
	return s
}

func TestPostgresDbPriceGranularity(t *testing.T) {
	defer testdb.Reset()

	sql := "SELECT uuid, price_granularity FROM accounts_account where uuid = $1 LIMIT 1"
	columns := []string{"uuid", "price_granularity"}
	result := `
	  bdc928ef-f725-4688-8171-c104cc715bdf,med
	  `
	testdb.StubQuery(sql, testdb.RowsFromCSVString(columns, result))

	conf := PostgresConfig{
		Host:     "host",
		Port:     1234,
		Dbname:   "dbname",
		User:     "user",
		Password: "password",
		TTL:      3434,
		Size:     100,
	}
	dataCache := StubNew(conf)

	account, err := dataCache.Accounts().Get("bdc928ef-f725-4688-8171-c104cc715bdf")
	if err != nil {
		glog.Errorf("test postgres db errored: %v", err)
	}

	if account.ID != "bdc928ef-f725-4688-8171-c104cc715bdf" {
		t.Error("Expected bdc928ef-f725-4688-8171-c104cc715bdf")
	}
	if account.PriceGranularity != "med" {
		t.Error("Expected med")
	}
}

func TestPostgresDbNullPriceGranularity(t *testing.T) {
	defer testdb.Reset()

	sql := "SELECT uuid, price_granularity FROM accounts_account where uuid = $1 LIMIT 1"
	columns := []string{"uuid", "price_granularity"}
	result := `
	  bdc928ef-f725-4688-8171-c104cc715bdf
	  `
	testdb.StubQuery(sql, testdb.RowsFromCSVString(columns, result))

	conf := PostgresConfig{
		Host:     "host",
		Port:     1234,
		Dbname:   "dbname",
		User:     "user",
		Password: "password",
		TTL:      3434,
		Size:     100,
	}
	dataCache := StubNew(conf)

	account, err := dataCache.Accounts().Get("bdc928ef-f725-4688-8171-c104cc715bdf")
	if err != nil {
		glog.Errorf("test postgres db errored: %v", err)
	}

	if account.ID != "bdc928ef-f725-4688-8171-c104cc715bdf" {
		t.Error("Expected bdc928ef-f725-4688-8171-c104cc715bdf")
	}
	if account.PriceGranularity != "" {
		t.Error("Expected null string")
	}
}
