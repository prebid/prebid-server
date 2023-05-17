package endpoints

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	var testCases = []struct {
		description string
		version     string
		revision    string
		expected    string
	}{
		{
			description: "Empty",
			version:     "",
			revision:    "",
			expected:    `{"revision":"not-set","version":"not-set"}`,
		},
		{
			description: "Version Only",
			version:     "1.2.3",
			revision:    "",
			expected:    `{"revision":"not-set","version":"1.2.3"}`,
		},
		{
			description: "Revision Only",
			version:     "",
			revision:    "d6cd1e2bd19e03a81132a23b2025920577f84e37",
			expected:    `{"revision":"d6cd1e2bd19e03a81132a23b2025920577f84e37","version":"not-set"}`,
		},
		{
			description: "Fully Populated",
			version:     "1.2.3",
			revision:    "d6cd1e2bd19e03a81132a23b2025920577f84e37",
			expected:    `{"revision":"d6cd1e2bd19e03a81132a23b2025920577f84e37","version":"1.2.3"}`,
		},
	}

	for _, test := range testCases {
		handler := NewVersionEndpoint(test.version, test.revision)
		w := httptest.NewRecorder()

		handler(w, nil)

		response, err := io.ReadAll(w.Result().Body)
		if assert.NoError(t, err, test.description+":read") {
			assert.JSONEq(t, test.expected, string(response), test.description+":response")
		}
	}
}
