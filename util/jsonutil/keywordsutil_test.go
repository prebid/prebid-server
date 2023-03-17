package jsonutil

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeywordsUtilUnmarshalJSON(t *testing.T) {
	type Keywords struct {
		Keywords KeywordsString `json:"keywords"`
	}

	t.Run("dynamic-json", func(t *testing.T) {
		jsonData := []byte(`{"keywords" : { "genre": ["rock", "pop"], "pets": ["dog"] }}`)
		var keywords Keywords
		assert.NoError(t, json.Unmarshal(jsonData, &keywords))
	})

	t.Run("json-array", func(t *testing.T) {
		jsonData := []byte(`{"keywords" : [{"key": "foo", "value": ["bar","baz"]},{"key": "valueless"}]}`)
		var keywords Keywords
		assert.NoError(t, json.Unmarshal(jsonData, &keywords))
		assert.Equal(t, "foo=bar,foo=baz,valueless", string(keywords.Keywords))
	})

	t.Run("string", func(t *testing.T) {
		jsonData := []byte(`{"keywords" : "foo=bar,foo=baz,valueless"}`)
		var keywords Keywords
		assert.NoError(t, json.Unmarshal(jsonData, &keywords))
		assert.Equal(t, "foo=bar,foo=baz,valueless", string(keywords.Keywords))
	})
}
