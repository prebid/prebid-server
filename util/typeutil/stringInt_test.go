package typeutil

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalJSON(t *testing.T) {
	type Item struct {
		ItemId StringInt `json:"item_id"`
	}

	t.Run("happy path - convert string to int", func(t *testing.T) {
		jsonData := []byte(`{"item_id":"30"}`)
		var item Item
		assert.NoError(t, json.Unmarshal(jsonData, &item))
		assert.Equal(t, 30, int(item.ItemId))
	})

	t.Run("happy path - make sure integer still works", func(t *testing.T) {
		jsonData := []byte(`{"item_id":30}`)
		var item Item
		assert.NoError(t, json.Unmarshal(jsonData, &item))
		assert.Equal(t, 30, int(item.ItemId))
	})
}
