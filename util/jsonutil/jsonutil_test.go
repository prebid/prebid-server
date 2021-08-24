package jsonutil

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestDropElement(t *testing.T) {

	tests := []struct {
		description     string
		input           []byte
		elementToRemove []string
		output          []byte
		errorExpected   bool
		errorContains   string
	}{
		{
			description:     "Drop Single Element After Another Element",
			input:           []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"test": 1,"consented_providers": [1608,765,492]}}`),
			elementToRemove: []string{"consented_providers"},
			output:          []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"test": 1}}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Single Element Before Another Element",
			input:           []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": [1608,765,492],"test": 1}}`),
			elementToRemove: []string{"consented_providers"},
			output:          []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"test": 1}}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Single Element",
			input:           []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": [1545,2563,1411]}}`),
			elementToRemove: []string{"consented_providers"},
			output:          []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {}}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Single Element string",
			input:           []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": "test"}}`),
			elementToRemove: []string{"consented_providers"},
			output:          []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {}}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Parent Element Between Two Elements",
			input:           []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1},"test": 123}`),
			elementToRemove: []string{"consented_providers_settings"},
			output:          []byte(`{"consent": "TESTCONSENT","test": 123}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Parent Element Before Element",
			input:           []byte(`{"consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1},"test": 123}`),
			elementToRemove: []string{"consented_providers_settings"},
			output:          []byte(`{"test": 123}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Parent Element After Element",
			input:           []byte(`{"consent": "TESTCONSENT","consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1}}`),
			elementToRemove: []string{"consented_providers_settings"},
			output:          []byte(`{"consent": "TESTCONSENT"}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Parent Element Only",
			input:           []byte(`{"consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1}}`),
			elementToRemove: []string{"consented_providers_settings"},
			output:          []byte(`{}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Parent Element List",
			input:           []byte(`{"consented_providers_settings":{"consented_providers":[1608,765,492],"test":1},"data": [{"test1":5},{"test2": [1,2,3]}]}`),
			elementToRemove: []string{"data"},
			output:          []byte(`{"consented_providers_settings":{"consented_providers":[1608,765,492],"test":1}}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Element That Doesn't Exist",
			input:           []byte(`{"consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1}}`),
			elementToRemove: []string{"test2"},
			output:          []byte(`{"consented_providers_settings": {"consented_providers": [1608,765,492], "test": 1}}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Nested Element Single Occurrence",
			input:           []byte(`{"consented_providers_settings":{"consented_providers":[1608,765,492],"test":1},"data": [{"test1":5},{"test2": [1,2,3]}]}`),
			elementToRemove: []string{"consented_providers_settings", "test"},
			output:          []byte(`{"consented_providers_settings":{"consented_providers":[1608,765,492]},"data": [{"test1":5},{"test2": [1,2,3]}]}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Nested Element Multiple Occurrence",
			input:           []byte(`{"consented_providers_settings":{"consented_providers":[1608,765,492],"test":1},"data": [{"test":5},{"test": [1,2,3]}]}`),
			elementToRemove: []string{"consented_providers_settings", "test"},
			output:          []byte(`{"consented_providers_settings":{"consented_providers":[1608,765,492]},"data": [{"test":5},{"test": [1,2,3]}]}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Nested Structure Single Occurrence",
			input:           []byte(`{"consented_providers":{"providers":[1608,765,492],"test":{"nested":true}},"data": [{"test":5},{"test": [1,2,3]}]}`),
			elementToRemove: []string{"consented_providers", "test"},
			output:          []byte(`{"consented_providers":{"providers":[1608,765,492]},"data": [{"test":5},{"test": [1,2,3]}]}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Nested Structure Single Occurrence Deep Nested",
			input:           []byte(`{"consented_providers":{"providers":[1608,765,492],"test":{"nested":true, "nested2": {"test6": 123}}},"data": [{"test":5},{"test": [1,2,3]}]}`),
			elementToRemove: []string{"consented_providers", "test6"},
			output:          []byte(`{"consented_providers":{"providers":[1608,765,492],"test":{"nested":true, "nested2": {}}},"data": [{"test":5},{"test": [1,2,3]}]}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Nested Structure Single Occurrence Deep Nested Full Path",
			input:           []byte(`{"consented_providers":{"providers":[1608,765,492],"test":{"nested":true,"nested2": {"test6": 123}}},"data": [{"test":5},{"test": [1,2,3]}]}`),
			elementToRemove: []string{"consented_providers", "test", "nested"},
			output:          []byte(`{"consented_providers":{"providers":[1608,765,492],"test":{"nested2": {"test6": 123}}},"data": [{"test":5},{"test": [1,2,3]}]}`),
			errorExpected:   false,
			errorContains:   "",
		},
		{
			description:     "Drop Nested Structure Doesn't Exist",
			input:           []byte(`{"consented_providers":{"providers":[1608,765,492]},"test":{"nested":true}},"data": [{"test":5},{"test": [1,2,3]}]}`),
			elementToRemove: []string{"consented_providers", "test2"},
			output:          []byte(`{"consented_providers":{"providers":[1608,765,492]},"test":{"nested":true}},"data": [{"test":5},{"test": [1,2,3]}]}`),
			errorExpected:   false,
			errorContains:   "",
		},
		//Errors
		{
			description:     "Error Decode",
			input:           []byte(`{"consented_providers_settings": {"consented_providers": ["123",1,,1365,5678,1545,2563,1411], "test": 1}}`),
			elementToRemove: []string{"consented_providers"},
			output:          []byte(``),
			errorExpected:   true,
			errorContains:   "looking for beginning of value",
		},
		{
			description:     "Error Malformed",
			input:           []byte(`{consented_providers_settings: {"consented_providers": [1365,5678,1545,2563,1411], "test": 1}}`),
			elementToRemove: []string{"consented_providers"},
			output:          []byte(``),
			errorExpected:   true,
			errorContains:   "invalid character",
		},
	}

	for _, tt := range tests {
		res, err := DropElement(tt.input, tt.elementToRemove...)

		if tt.errorExpected {
			assert.Error(t, err, "Error should not be nil")
			assert.True(t, strings.Contains(err.Error(), tt.errorContains))
		} else {
			assert.NoError(t, err, "Error should be nil")
			assert.Equal(t, tt.output, res, "Result is incorrect")
		}
	}
}

func TestSetElement(t *testing.T) {

	tests := []struct {
		description   string
		input         []byte
		setValue      []byte
		setTo         []string
		output        []byte
		errorExpected bool
		errorContains string
	}{
		{
			description:   "Set Element Nested Exists",
			input:         []byte(`{"data":{"sitedata":"mysitedata"}}`),
			setValue:      []byte(`{"somefpd":"fpdDataTest"}`),
			setTo:         []string{"data"},
			output:        []byte(`{"data":{"sitedata":"mysitedata","somefpd":"fpdDataTest"}}`),
			errorExpected: false,
			errorContains: "",
		},
		{
			description:   "Set Element Nested Doesn't Exists",
			input:         []byte(`{"data":{"sitedata":"mysitedata"}}`),
			setValue:      []byte(`{"somefpd":"fpdDataTest"}`),
			setTo:         []string{"providers"},
			output:        []byte(`{"data":{"sitedata":"mysitedata"},"providers":{"somefpd":"fpdDataTest"}}`),
			errorExpected: false,
			errorContains: "",
		},
	}
	for _, tt := range tests {
		res, err := SetElement(tt.input, tt.setValue, tt.setTo...)
		if tt.errorExpected {
			assert.Error(t, err, "Error should not be nil")
			assert.True(t, strings.Contains(err.Error(), tt.errorContains))
		} else {
			assert.NoError(t, err, "Error should be nil")
			assert.Equal(t, tt.output, res, "Result is incorrect")
		}
	}

}

func TestFindAndDropElement(t *testing.T) {
	testCases := []struct {
		description   string
		input         []byte
		dataPath      []string
		output        []byte
		foundData     []byte
		errorExpected bool
		errorContains string
	}{
		{
			description: "Element exists",
			input: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "publisher": {
  				    "id": "1"
  				  },
  				  "data": {"somesitefpd": "sitefpdDataTest"}
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			dataPath: []string{"site", "data"},
			output: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "publisher": {
  				    "id": "1"
  				  }
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			foundData:     []byte(`{"somesitefpd": "sitefpdDataTest"}`),
			errorExpected: false,
			errorContains: "",
		},
		{
			description: "Element doesn't exists",
			input: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "publisher": {
  				    "id": "1"
  				  },
  				  "data": {"somesitefpd": "sitefpdDataTest"}
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			dataPath: []string{"site", "test"},
			output: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "publisher": {
  				    "id": "1"
  				  },
  				  "data": {"somesitefpd": "sitefpdDataTest"}
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			foundData:     []byte{},
			errorExpected: false,
			errorContains: "",
		},
		{
			description: "Non object element exists",
			input: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "publisher": {
  				    "id": "1"
  				  },
  				  "data": {"somesitefpd": "sitefpdDataTest"}
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			dataPath: []string{"site", "id"},
			output: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "page": "http://www.foobar.com/1234.html",
  				  "publisher": {
  				    "id": "1"
  				  },
  				  "data": {"somesitefpd": "sitefpdDataTest"}
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			foundData:     []byte(`reqSiteId`),
			errorExpected: false,
			errorContains: "",
		},
		{
			description: "Malformed input",
			input: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "publisher": {
  				    "id": "1"
  				  },
  				  "data":  sitefpdDataTest"}
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			dataPath: []string{"site", "data"},
			output: []byte(`{
  				"id": "bid_id",
  				"site": {
  				  "id":"reqSiteId",
  				  "page": "http://www.foobar.com/1234.html",
  				  "publisher": {
  				    "id": "1"
  				  },
  				  "data":  sitefpdDataTest"}
  				},
  				"tmax": 5000,
  				"source": {
  				  "tid": "ad839de0-5ae6-40bb-92b2-af8bad6439b3"
  				}
			}`),
			foundData:     []byte(nil),
			errorExpected: true,
			errorContains: "Unknown value type",
		},
	}
	for _, test := range testCases {
		res, data, err := FindAndDropElement(test.input, test.dataPath...)

		if test.errorExpected {
			assert.Error(t, err, "Error should not be nil")
			assert.Equal(t, test.output, res, "Result should be still returned")
			assert.True(t, strings.Contains(err.Error(), test.errorContains))
		} else {
			assert.NoError(t, err, "Error should be nil")
			assert.Equal(t, test.output, res, "Result is incorrect")
			assert.Equal(t, test.foundData, data, "FPD is incorrect")
		}
	}

}
