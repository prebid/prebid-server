package openrtb_ext

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xeipuuv/gojsonschema"
)

func TestBidderParamValidatorValidate(t *testing.T) {
	testSchemaLoader := gojsonschema.NewStringLoader(`{
		"$schema": "http://json-schema.org/draft-04/schema#",
		"title": "Test Params",
		"description": "Test Description",
		"type": "object",
		"properties": {
		  "placementId": {
			"type": "integer",
			"description": "An ID which identifies this placement of the impression."
		  },
		  "optionalText": {
			"type": "string",
			"description": "Optional text for testing."
		  }
		},
		"required": ["placementId"]
	}`)
	testSchema, err := gojsonschema.NewSchema(testSchemaLoader)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	testBidderName := BidderName("foo")
	testValidator := bidderParamValidator{
		parsedSchemas: map[BidderName]*gojsonschema.Schema{
			testBidderName: testSchema,
		},
	}

	testCases := []struct {
		description   string
		ext           json.RawMessage
		expectedError string
	}{
		{
			description:   "Valid",
			ext:           json.RawMessage(`{"placementId":123}`),
			expectedError: "",
		},
		{
			description:   "Invalid - Wrong Type",
			ext:           json.RawMessage(`{"placementId":"stringInsteadOfInt"}`),
			expectedError: "placementId: Invalid type. Expected: integer, given: string",
		},
		{
			description:   "Invalid - Empty Object",
			ext:           json.RawMessage(`{}`),
			expectedError: "placementId: placementId is required",
		},
		{
			description:   "Malformed",
			ext:           json.RawMessage(`malformedJSON`),
			expectedError: "invalid character 'm' looking for beginning of value",
		},
	}

	for _, test := range testCases {
		err := testValidator.Validate(testBidderName, test.ext)
		if test.expectedError == "" {
			assert.NoError(t, err, test.description)
		} else {
			assert.EqualError(t, err, test.expectedError, test.description)
		}
	}
}

func TestBidderParamValidatorSchema(t *testing.T) {
	testValidator := bidderParamValidator{
		schemaContents: map[BidderName]string{
			BidderName("foo"): "foo content",
			BidderName("bar"): "bar content",
		},
	}

	result := testValidator.Schema(BidderName("bar"))

	assert.Equal(t, "bar content", result)
}

func TestIsBidderNameReserved(t *testing.T) {
	testCases := []struct {
		bidder   string
		expected bool
	}{
		{"all", true},
		{"aLl", true},
		{"ALL", true},
		{"context", true},
		{"CONTEXT", true},
		{"conTExt", true},
		{"data", true},
		{"DATA", true},
		{"DaTa", true},
		{"general", true},
		{"gEnErAl", true},
		{"GENERAL", true},
		{"skadn", true},
		{"skADN", true},
		{"SKADN", true},
		{"prebid", true},
		{"PREbid", true},
		{"PREBID", true},
		{"notreserved", false},
	}

	for _, test := range testCases {
		result := IsBidderNameReserved(test.bidder)
		assert.Equal(t, test.expected, result, test.bidder)
	}
}
