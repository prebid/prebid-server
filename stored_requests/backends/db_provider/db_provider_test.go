package db_provider

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrepareQuery(t *testing.T) {
	tests := []struct {
		description string

		template      string
		params        []QueryParam
		mySqlQuery    string
		mySqlArgs     []interface{}
		postgresQuery string
		postgresArgs  []interface{}
	}{
		{
			description:   "Np parameters",
			template:      "SELECT * FROM table",
			params:        []QueryParam{},
			mySqlQuery:    "SELECT * FROM table",
			mySqlArgs:     []interface{}{},
			postgresQuery: "SELECT * FROM table",
			postgresArgs:  []interface{}{},
		},
		{
			description:   "One simple parameter",
			template:      "SELECT * FROM table WHERE id = $ID",
			params:        []QueryParam{{Name: "ID", Value: "1001"}},
			mySqlQuery:    "SELECT * FROM table WHERE id = ?",
			mySqlArgs:     []interface{}{"1001"},
			postgresQuery: "SELECT * FROM table WHERE id = $1",
			postgresArgs:  []interface{}{"1001"},
		},
		{
			description: "Two simple parameters",
			template:    "SELECT * FROM table WHERE id = $ID AND name = $NAME",
			params: []QueryParam{
				{Name: "ID", Value: "1001"},
				{Name: "NAME", Value: "Alice"},
			},
			mySqlQuery:    "SELECT * FROM table WHERE id = ? AND name = ?",
			mySqlArgs:     []interface{}{"1001", "Alice"},
			postgresQuery: "SELECT * FROM table WHERE id = $1 AND name = $2",
			postgresArgs:  []interface{}{"1001", "Alice"},
		},
		{
			description: "Two simple parameters, used several times",
			template:    "SELECT $ID, $NAME, * FROM table WHERE id = $ID AND name = $NAME",
			params: []QueryParam{
				{Name: "ID", Value: "1001"},
				{Name: "NAME", Value: "Alice"},
			},
			mySqlQuery:    "SELECT ?, ?, * FROM table WHERE id = ? AND name = ?",
			mySqlArgs:     []interface{}{"1001", "Alice", "1001", "Alice"},
			postgresQuery: "SELECT $1, $2, * FROM table WHERE id = $1 AND name = $2",
			postgresArgs:  []interface{}{"1001", "Alice"},
		},
		{
			description:   "Empty list parameter",
			template:      "SELECT * FROM table WHERE id IN $IDS",
			params:        []QueryParam{{Name: "IDS", Value: []interface{}{}}},
			mySqlQuery:    "SELECT * FROM table WHERE id IN (NULL)",
			mySqlArgs:     []interface{}{},
			postgresQuery: "SELECT * FROM table WHERE id IN (NULL)",
			postgresArgs:  []interface{}{},
		},
		{
			description:   "One list parameter",
			template:      "SELECT * FROM table WHERE id IN $IDS",
			params:        []QueryParam{{Name: "IDS", Value: []interface{}{"1001", "1002"}}},
			mySqlQuery:    "SELECT * FROM table WHERE id IN (?, ?)",
			mySqlArgs:     []interface{}{"1001", "1002"},
			postgresQuery: "SELECT * FROM table WHERE id IN ($1, $2)",
			postgresArgs:  []interface{}{"1001", "1002"},
		},
		{
			description: "Two list parameters",
			template:    "SELECT * FROM table WHERE id IN $IDS OR name in $NAMES",
			params: []QueryParam{
				{Name: "IDS", Value: []interface{}{"1001"}},
				{Name: "NAMES", Value: []interface{}{"Bob", "Nancy"}},
			},
			mySqlQuery:    "SELECT * FROM table WHERE id IN (?) OR name in (?, ?)",
			mySqlArgs:     []interface{}{"1001", "Bob", "Nancy"},
			postgresQuery: "SELECT * FROM table WHERE id IN ($1) OR name in ($2, $3)",
			postgresArgs:  []interface{}{"1001", "Bob", "Nancy"},
		},
		{
			description: "Mix of simple and list parameters",
			template: `
				SELECT * FROM table1
				WHERE last_updated > $LAST_UPDATED
				AND (id IN $IDS OR name in $NAMES)
				UNION ALL
				SELECT * FROM table1
				WHERE last_updated > $LAST_UPDATED
				AND (id IN $IDS OR name in $NAMES)
				`,
			params: []QueryParam{
				{Name: "LAST_UPDATED", Value: "1970-01-01"},
				{Name: "IDS", Value: []interface{}{"1001"}},
				{Name: "NAMES", Value: []interface{}{"Bob", "Nancy"}},
			},
			mySqlQuery: `
				SELECT * FROM table1
				WHERE last_updated > ?
				AND (id IN (?) OR name in (?, ?))
				UNION ALL
				SELECT * FROM table1
				WHERE last_updated > ?
				AND (id IN (?) OR name in (?, ?))
				`,
			mySqlArgs: []interface{}{
				"1970-01-01",
				"1001",
				"Bob", "Nancy",
				"1970-01-01",
				"1001",
				"Bob", "Nancy",
			},
			postgresQuery: `
				SELECT * FROM table1
				WHERE last_updated > $1
				AND (id IN ($2) OR name in ($3, $4))
				UNION ALL
				SELECT * FROM table1
				WHERE last_updated > $1
				AND (id IN ($2) OR name in ($3, $4))
				`,
			postgresArgs: []interface{}{
				"1970-01-01",
				"1001",
				"Bob", "Nancy",
			},
		},
	}

	for _, tt := range tests {
		mySqlDbProvider := MySqlDbProvider{}
		mySqlQuery, mySqlArgs := mySqlDbProvider.PrepareQuery(tt.template, tt.params...)
		assert.Equal(t, tt.mySqlQuery, mySqlQuery, fmt.Sprintf("MySql: %s", tt.description))
		assert.Equal(t, tt.mySqlArgs, mySqlArgs, fmt.Sprintf("MySql: %s", tt.description))

		postgresDbProvider := PostgresDbProvider{}
		postgresQuery, postgresArgs := postgresDbProvider.PrepareQuery(tt.template, tt.params...)
		assert.Equal(t, tt.postgresQuery, postgresQuery, fmt.Sprintf("Postgres: %s", tt.description))
		assert.Equal(t, tt.postgresArgs, postgresArgs, fmt.Sprintf("Postgres: %s", tt.description))
	}
}
