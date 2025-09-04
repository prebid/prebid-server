package jsonutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringOrStringArrayUnmarshalJSON(t *testing.T) {
	type Item struct {
		Item StringOrStringArray `json:"item"`
	}

	t.Run("string", func(t *testing.T) {
		jsonData := []byte(`{"item":"hello"}`)
		var item Item
		assert.NoError(t, UnmarshalValid(jsonData, &item))
		assert.Equal(t, "hello", item.Item[0])
	})

	t.Run("string_array", func(t *testing.T) {
		jsonData := []byte(`{"item":["hello","world"]}`)
		var item Item
		assert.NoError(t, UnmarshalValid(jsonData, &item))
		assert.Equal(t, "hello", item.Item[0])
		assert.Equal(t, "world", item.Item[1])
	})

	t.Run("empty_array", func(t *testing.T) {
		jsonData := []byte(`{"item": []}`)
		var item Item
		assert.NoError(t, UnmarshalValid(jsonData, &item))
		assert.Empty(t, item.Item)
	})

	t.Run("invalid_input", func(t *testing.T) {
		jsonData := []byte(`{"item":true}`)
		var item Item
		err := UnmarshalValid(jsonData, &item)
		assert.EqualError(t, err, "cannot unmarshal jsonutil.Item.Item: value should be of type string or []string")
	})
}
