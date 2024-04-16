package merge

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v2/util/sliceutil"
	"github.com/stretchr/testify/assert"
)

func TestExtMerger(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		merger := extMerger{ext: nil, snapshot: json.RawMessage(`{"a":1}`)}
		assert.NoError(t, merger.Merge())
		assert.Nil(t, merger.ext)
	})

	testCases := []struct {
		name          string
		givenOriginal json.RawMessage
		givenJson     json.RawMessage
		expectedExt   json.RawMessage
		expectedErr   string
	}{
		{
			name:          "both-populated",
			givenOriginal: json.RawMessage(`{"a":1,"b":2}`),
			givenJson:     json.RawMessage(`{"b":200,"c":3}`),
			expectedExt:   json.RawMessage(`{"a":1,"b":200,"c":3}`),
		},
		{
			name:          "both-nil",
			givenJson:     nil,
			givenOriginal: nil,
			expectedExt:   nil,
		},
		{
			name:          "both-empty",
			givenOriginal: json.RawMessage(`{}`),
			givenJson:     json.RawMessage(`{}`),
			expectedExt:   json.RawMessage(`{}`),
		},
		{
			name:          "ext-nil",
			givenOriginal: json.RawMessage(`{"b":2}`),
			givenJson:     nil,
			expectedExt:   json.RawMessage(`{"b":2}`),
		},
		{
			name:          "ext-empty",
			givenOriginal: json.RawMessage(`{"b":2}`),
			givenJson:     json.RawMessage(`{}`),
			expectedExt:   json.RawMessage(`{"b":2}`),
		},
		{
			name:          "ext-malformed",
			givenOriginal: json.RawMessage(`{"b":2}`),
			givenJson:     json.RawMessage(`malformed`),
			expectedErr:   "invalid override ext",
		},
		{
			name:          "snapshot-nil",
			givenOriginal: nil,
			givenJson:     json.RawMessage(`{"a":1}`),
			expectedExt:   json.RawMessage(`{"a":1}`),
		},
		{
			name:          "snapshot-empty",
			givenOriginal: json.RawMessage(`{}`),
			givenJson:     json.RawMessage(`{"a":1}`),
			expectedExt:   json.RawMessage(`{"a":1}`),
		},
		{
			name:          "snapshot-malformed",
			givenOriginal: json.RawMessage(`malformed`),
			givenJson:     json.RawMessage(`{"a":1}`),
			expectedErr:   "invalid request ext",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// Initialize A Ext Raw Message For Testing
			simulatedExt := json.RawMessage(sliceutil.Clone(test.givenOriginal))

			// Begin Tracking
			var merger extMerger
			merger.Track(&simulatedExt)

			// Unmarshal
			simulatedExt.UnmarshalJSON(test.givenJson)

			// Merge
			actualErr := merger.Merge()

			if test.expectedErr == "" {
				assert.NoError(t, actualErr, "error")

				if test.expectedExt == nil {
					assert.Nil(t, simulatedExt, "json")
				} else {
					assert.JSONEq(t, string(test.expectedExt), string(simulatedExt), "json")
				}
			} else {
				assert.EqualError(t, actualErr, test.expectedErr, "error")
			}
		})
	}
}
