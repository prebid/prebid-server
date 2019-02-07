package endpoints

import (
	"encoding/json"
	"io/ioutil"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestVersion(t *testing.T) {
	// Setup:
	var testCases = []struct {
		input    string
		expected string
	}{
		{"", `{"revision": "not-set"}`},
		{"abc", `{"revision": "abc"}`},
		{"d6cd1e2bd19e03a81132a23b2025920577f84e37", `{"revision": "d6cd1e2bd19e03a81132a23b2025920577f84e37"}`},
	}

	for _, tc := range testCases {

		handler := NewVersionEndpoint(tc.input)
		w := httptest.NewRecorder()

		// Execute:
		handler(w, nil)

		// Verify:
		var result, expected versionModel
		err := json.NewDecoder(w.Body).Decode(&result)
		if err != nil {
			t.Errorf("Bad response body. Expected: %s, got an error %s", tc.expected, err)
		}

		err = json.Unmarshal([]byte(tc.expected), &expected)
		if err != nil {
			t.Errorf("Error while trying to unmarshal expected result JSON")
		}

		if !reflect.DeepEqual(expected, result) {
			responseBodyBytes, _ := ioutil.ReadAll(w.Body)
			responseBodyString := string(responseBodyBytes)
			t.Errorf("Bad response body. Expected: %s, got %s", tc.expected, responseBodyString)
		}
	}
}
