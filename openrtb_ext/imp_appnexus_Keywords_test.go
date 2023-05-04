package openrtb_ext

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeywordsUnmarshalJSON(t *testing.T) {
	type keywords struct {
		Keywords Keywords `json:"keywords"`
	}

	type validTest struct {
		input    []byte
		expected string
	}

	validTests := []validTest{
		{input: []byte(`{"keywords" : { "pets": ["dog"] }}`), expected: "pets=dog"},
		{input: []byte(`{"keywords" : { "foo":[] }}`), expected: "foo"},
		{input: []byte(`{"keywords" : [{"key": "foo", "value": ["bar","baz"]},{"key": "valueless"}]}`), expected: "foo=bar,foo=baz,valueless"},
		{input: []byte(`{"keywords" : "foo=bar,foo=baz,valueless"}`), expected: "foo=bar,foo=baz,valueless"},
		{input: []byte(`{"keywords" : ""}`), expected: ""},
	}

	for _, test := range validTests {
		var keywords keywords
		assert.NoError(t, json.Unmarshal(test.input, &keywords))
		assert.Equal(t, test.expected, keywords.Keywords.String())
	}
}
