package firstpartydata

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtMerger(t *testing.T) {
	testCases := []struct {
		name        string
		givenBefore json.RawMessage
		givenAfter  json.RawMessage
		expectedExt json.RawMessage
		expectedErr string
	}{
		{
			name:        "both-populated",
			givenBefore: json.RawMessage(`{"a":1,"b":2}`),
			givenAfter:  json.RawMessage(`{"b":200,"c":3}`),
			expectedExt: json.RawMessage(`{"a":1,"b":200,"c":3}`),
		},
		{
			name:        "both-nil",
			givenAfter:  nil,
			givenBefore: nil,
			expectedExt: nil,
		},
		{
			name:        "both-empty",
			givenBefore: json.RawMessage(`{}`),
			givenAfter:  json.RawMessage(`{}`),
			expectedExt: json.RawMessage(`{}`),
		},
		{
			name:        "ext-nil",
			givenBefore: json.RawMessage(`{"b":2}`),
			givenAfter:  nil,
			expectedExt: json.RawMessage(`{"b":2}`),
		},
		{
			name:        "ext-empty",
			givenBefore: json.RawMessage(`{"b":2}`),
			givenAfter:  json.RawMessage(`{}`),
			expectedExt: json.RawMessage(`{"b":2}`),
		},
		{
			name:        "ext-malformed",
			givenBefore: json.RawMessage(`{"b":2}`),
			givenAfter:  json.RawMessage(`malformed`),
			expectedErr: "Invalid JSON Patch", //todo: better error message
		},
		{
			name:        "snapshot-nil",
			givenBefore: nil,
			givenAfter:  json.RawMessage(`{"a":1}`),
			expectedExt: json.RawMessage(`{"a":1}`),
		},
		{
			name:        "snapshot-empty",
			givenBefore: json.RawMessage(`{}`),
			givenAfter:  json.RawMessage(`{"a":1}`),
			expectedExt: json.RawMessage(`{"a":1}`),
		},
		{
			name:        "snapshot-malformed",
			givenBefore: json.RawMessage(`malformed`),
			givenAfter:  json.RawMessage(`{"a":1}`),
			expectedErr: "Invalid JSON Document",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ext := &test.givenBefore // best to use a sub-item to properly simulate

			// Begin Tracking
			var merger extMerger
			merger.Track(ext)

			// Unmarshal (let's do it for real)
			json.Unmarshal("data", ext)

			// Merge
			actualErr := merger.Merge()

			if test.expectedErr == "" {
				assert.NoError(t, actualErr, "error")

				if test.expectedExt == nil {
					assert.Equal(t, test.expectedExt, test.givenAfter, "json")
				} else {
					assert.JSONEq(t, string(test.expectedExt), string(test.givenAfter), "json")
				}
			} else {
				assert.EqualError(t, actualErr, test.expectedErr, "error")
			}
		})
	}
}
