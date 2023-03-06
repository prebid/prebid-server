package jsonutil

import (
	"encoding/json"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
)

func TestStringIntUnmarshalJSON(t *testing.T) {
	type Item struct {
		ItemId StringInt `json:"item_id"`
	}

	t.Run("string", func(t *testing.T) {
		jsonData := []byte(`{"item_id":"30"}`)
		var item Item
		assert.NoError(t, json.Unmarshal(jsonData, &item))
		assert.Equal(t, 30, int(item.ItemId))
	})

	t.Run("int", func(t *testing.T) {
		jsonData := []byte(`{"item_id":30}`)
		var item Item
		assert.NoError(t, json.Unmarshal(jsonData, &item))
		assert.Equal(t, 30, int(item.ItemId))
	})

	t.Run("empty_id", func(t *testing.T) {
		jsonData := []byte(`{"item_id": ""}`)
		var item Item
		assert.NoError(t, json.Unmarshal(jsonData, &item))
	})

	t.Run("invalid_input", func(t *testing.T) {
		jsonData := []byte(`{"item_id":true}`)
		var item Item
		err := json.Unmarshal(jsonData, &item)
		assert.Equal(t, jsonparser.MalformedValueError, err)
	})
}
