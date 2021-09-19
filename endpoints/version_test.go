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
		version  string
		revision string
		expected string
	}{
		{"", "", `{"version":"not-set","revision":"not-set"}`},
		{"abc", "def", `{"version":"abc","revision"	:"def"}`},
		{"1.2.3", "d6cd1e2bd19e03a81132a23b2025920577f84e37", `{"version":"1.2.3","revision":"d6cd1e2bd19e03a81132a23b2025920577f84e37"}`},
	}

	for _, tc := range testCases {

		handler := NewVersionEndpoint(tc.version, tc.revision)
		w := httptest.NewRecorder()

		// Execute:
		handler(w, nil)

		// Verify:
		var result, expected versionModel
		responseBodyBytes, err := ioutil.ReadAll(w.Body)
		if err != nil {
			t.Errorf("Error reading response body bytes: %s", err)
		}
		err = json.Unmarshal(responseBodyBytes, &result)
		if err != nil {
			t.Errorf("Bad response body. Expected: %s, got an error %s", tc.expected, err)
		}

		err = json.Unmarshal([]byte(tc.expected), &expected)
		if err != nil {
			t.Errorf("Error while trying to unmarshal expected result JSON")
		}

		if !reflect.DeepEqual(expected, result) {
			responseBodyString := string(responseBodyBytes)
			t.Errorf("Bad response body. Expected: %s, got %s", tc.expected, responseBodyString)
		}
	}
}
