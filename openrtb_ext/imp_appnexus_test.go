package openrtb_ext

import (
	"testing"

	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
)

func TestKeywordsUnmarshalJSON(t *testing.T) {
	type keywords struct {
		Keywords ExtImpAppnexusKeywords `json:"keywords"`
	}

	type testCase struct {
		input    []byte
		expected string
		desc     string
	}

	validTestCases := []testCase{
		{input: []byte(`{"keywords" : { "pets": ["dog"] }}`), expected: "pets=dog", desc: "dynamic json object"},
		{input: []byte(`{"keywords" : { "foo":[] }}`), expected: "foo", desc: "dynamic json object with empty value array"},
		{input: []byte(`{"keywords" : [{"key": "foo", "value": ["bar","baz"]},{"key": "valueless"}]}`), expected: "foo=bar,foo=baz,valueless", desc: "array of objects"},
		{input: []byte(`{"keywords" : "foo=bar,foo=baz,valueless"}`), expected: "foo=bar,foo=baz,valueless", desc: "string keywords"},
		{input: []byte(`{"keywords" : ""}`), expected: "", desc: "empty string"},
		{input: []byte(`{"keywords" : {}}`), expected: "", desc: "empty keywords object"},
		{input: []byte(`{"keywords" : [{}]}`), expected: "", desc: "empty keywords object array"},
		{input: []byte(`{"keywords": []}`), expected: "", desc: "empty keywords array"},
	}

	for _, test := range validTestCases {
		var keywords keywords
		assert.NoError(t, jsonutil.UnmarshalValid(test.input, &keywords), test.desc)
		assert.Equal(t, test.expected, keywords.Keywords.String())
	}

	invalidTestCases := []testCase{
		{input: []byte(`{"keywords": [{]}`), desc: "invalid keywords array"},
		{input: []byte(`{"keywords" : {"}}`), desc: "invalid keywords object"},
	}

	for _, test := range invalidTestCases {
		var keywords keywords
		assert.Error(t, jsonutil.UnmarshalValid(test.input, &keywords), test.desc)
	}
}
