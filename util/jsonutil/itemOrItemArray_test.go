package jsonutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestItemOrItemArrayUnmarshalJSON_String(t *testing.T) {
	type Data struct {
		Item ItemOrItemArray[string] `json:"item"`
	}

	t.Run("string", func(t *testing.T) {
		jsonData := []byte(`{"item":"hello"}`)
		var data Data
		assert.NoError(t, UnmarshalValid(jsonData, &data))
		assert.Equal(t, 1, len(data.Item))
		assert.Equal(t, "hello", data.Item[0])
	})

	t.Run("string_array", func(t *testing.T) {
		jsonData := []byte(`{"item":["hello","world"]}`)
		var data Data
		assert.NoError(t, UnmarshalValid(jsonData, &data))
		assert.Equal(t, 2, len(data.Item))
		assert.Equal(t, "hello", data.Item[0])
		assert.Equal(t, "world", data.Item[1])
	})

	t.Run("empty_array", func(t *testing.T) {
		jsonData := []byte(`{"item": []}`)
		var data Data
		assert.NoError(t, UnmarshalValid(jsonData, &data))
		assert.Empty(t, data.Item)
	})

	t.Run("invalid_input", func(t *testing.T) {
		jsonData := []byte(`{"item":true}`)
		var item Data
		err := UnmarshalValid(jsonData, &item)
		assert.EqualError(t, err, "cannot unmarshal jsonutil.Data.Item: value should be of type string or []string")
	})
}

func TestItemOrItemArrayUnmarshallJSON_Struct(t *testing.T) {
	type Item struct {
		Name string `json:"name"`
	}

	type Data struct {
		Item ItemOrItemArray[Item] `json:"item"`
	}

	t.Run("struct", func(t *testing.T) {
		jsonData := []byte(`{"item":{"name":"test"}}`)
		var data Data
		assert.NoError(t, UnmarshalValid(jsonData, &data))
		assert.Equal(t, "test", data.Item[0].Name)
	})

	t.Run("struct_array", func(t *testing.T) {
		jsonData := []byte(`{"item":[{"name":"test1"},{"name":"test2"}]}`)
		var data Data
		assert.NoError(t, UnmarshalValid(jsonData, &data))
		assert.Equal(t, "test1", data.Item[0].Name)
		assert.Equal(t, "test2", data.Item[1].Name)
	})

	t.Run("empty_array", func(t *testing.T) {
		jsonData := []byte(`{"item":[]}`)
		var data Data
		assert.NoError(t, UnmarshalValid(jsonData, &data))
		assert.Empty(t, data.Item)
	})

	t.Run("invalid_input", func(t *testing.T) {
		jsonData := []byte(`{"item":true}`)
		var item Data
		err := UnmarshalValid(jsonData, &item)
		assert.EqualError(t, err, "cannot unmarshal jsonutil.Data.Item: value should be of type jsonutil.Item or []jsonutil.Item")
	})
}
