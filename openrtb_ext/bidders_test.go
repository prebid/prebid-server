package openrtb_ext

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"testing/fstest"

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
			expectedError: "(root): placementId is required",
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
		{"gpid", true},
		{"GPID", true},
		{"GPid", true},
		{"prebid", true},
		{"PREbid", true},
		{"PREBID", true},
		{"skadn", true},
		{"skADN", true},
		{"SKADN", true},
		{"tid", true},
		{"TId", true},
		{"Tid", true},
		{"TiD", true},
		{"notreserved", false},
	}

	for _, test := range testCases {
		result := IsBidderNameReserved(test.bidder)
		assert.Equal(t, test.expected, result, test.bidder)
	}
}

func TestSetAliasBidderName(t *testing.T) {
	parentBidder := BidderName("pBidder")
	existingCoreBidderNames := coreBidderNames

	testCases := []struct {
		aliasBidderName string
		err             error
	}{
		{"aBidder", nil},
		{"all", errors.New("alias all is a reserved bidder name and cannot be used")},
	}

	for _, test := range testCases {
		err := SetAliasBidderName(test.aliasBidderName, parentBidder)
		if err != nil {
			assert.Equal(t, test.err, err)
		} else {
			assert.Contains(t, CoreBidderNames(), BidderName(test.aliasBidderName))
			assert.Contains(t, aliasBidderToParent, BidderName(test.aliasBidderName))
			assert.Contains(t, bidderNameLookup, strings.ToLower(test.aliasBidderName))
		}
	}

	//reset package variables to not interfere with other test cases. Example - TestBidderParamSchemas
	coreBidderNames = existingCoreBidderNames
	aliasBidderToParent = map[BidderName]BidderName{}
}

type mockParamsHelper struct {
	fs              fstest.MapFS
	absFilePath     string
	absPathErr      error
	schemaLoaderErr error
	readFileErr     error
}

func (m *mockParamsHelper) readDir(name string) ([]os.DirEntry, error) {
	return m.fs.ReadDir(name)
}

func (m *mockParamsHelper) readFile(name string) ([]byte, error) {
	if m.readFileErr != nil {
		return nil, m.readFileErr
	}
	return m.fs.ReadFile(name)
}

func (m *mockParamsHelper) newReferenceLoader(source string) gojsonschema.JSONLoader {
	return nil
}

func (m *mockParamsHelper) newSchema(l gojsonschema.JSONLoader) (*gojsonschema.Schema, error) {
	return nil, m.schemaLoaderErr
}

func (m *mockParamsHelper) abs(path string) (string, error) {
	return m.absFilePath, m.absPathErr
}

func TestNewBidderParamsValidator(t *testing.T) {
	testCases := []struct {
		description     string
		paramsValidator mockParamsHelper
		dir             string
		expectedErr     error
	}{
		{
			description: "Valid case",
			paramsValidator: mockParamsHelper{
				fs: fstest.MapFS{
					"test/appnexus.json": {
						Data: []byte("{}"),
					},
				},
			},
			dir: "test",
		},
		{
			description:     "failed to read directory",
			paramsValidator: mockParamsHelper{},
			dir:             "t",
			expectedErr:     errors.New("Failed to read JSON schemas from directory t. open t: file does not exist"),
		},
		{
			description: "file name does not match the bidder name",
			paramsValidator: mockParamsHelper{
				fs: fstest.MapFS{
					"test/anyBidder.json": {
						Data: []byte("{}"),
					},
				},
			},
			dir:         "test",
			expectedErr: errors.New("File test/anyBidder.json does not match a valid BidderName."),
		},
		{
			description: "abs file path error",
			paramsValidator: mockParamsHelper{
				fs: fstest.MapFS{
					"test/appnexus.json": {
						Data: []byte("{}"),
					},
				},
				absFilePath: "test/app.json",
				absPathErr:  errors.New("any abs error"),
			},
			dir:         "test",
			expectedErr: errors.New("Failed to get an absolute representation of the path: test/app.json, any abs error"),
		},
		{
			description: "schema loader error",
			paramsValidator: mockParamsHelper{
				fs: fstest.MapFS{
					"test/appnexus.json": {
						Data: []byte("{}"),
					},
				},
				schemaLoaderErr: errors.New("any schema loader error"),
			},
			dir:         "test",
			expectedErr: errors.New("Failed to load json schema at : any schema loader error"),
		},
		{
			description: "read file error",
			paramsValidator: mockParamsHelper{
				fs: fstest.MapFS{
					"test/appnexus.json": {
						Data: []byte("{}"),
					},
				},
				readFileErr: errors.New("any read file error"),
			},
			dir:         "test",
			expectedErr: errors.New("Failed to read file test/appnexus.json: any read file error"),
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			aliasBidderToParent = map[BidderName]BidderName{"rubicon": "appnexus"}
			paramsValidator = &test.paramsValidator
			bidderValidator, err := NewBidderParamsValidator(test.dir)
			if test.expectedErr == nil {
				assert.NoError(t, err)
				assert.Contains(t, bidderValidator.Schema("appnexus"), "{}")
				assert.Contains(t, bidderValidator.Schema("rubicon"), "{}")
			} else {
				assert.Equal(t, err, test.expectedErr)
			}
		})
	}
}
