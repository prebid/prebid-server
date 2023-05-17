package smaato

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractAdmRichMedia(t *testing.T) {
	tests := []struct {
		testName         string
		adMarkup         string
		expectedAdMarkup string
		expectedError    string
	}{
		{
			testName: "extract richmedia",
			adMarkup: "{\"richmedia\":{\"mediadata\":{\"content\":\"<div>hello</div>\"," +
				"" + "\"w\":350," +
				"\"h\":50},\"impressiontrackers\":[\"//prebid-test.smaatolabs.net/track/imp/1\",\"//prebid-test.smaatolabs.net/track/imp/2\"]," +
				"\"clicktrackers\":[\"//prebid-test.smaatolabs.net/track/click/1\",\"//prebid-test.smaatolabs.net/track/click/2\"]}}",
			expectedAdMarkup: `<div onclick="fetch(decodeURIComponent('%2F%2Fprebid-test.smaatolabs.net%2Ftrack%2Fclick%2F1'),` +
				` {cache: 'no-cache'});fetch(decodeURIComponent('%2F%2Fprebid-test.smaatolabs.net%2Ftrack%2Fclick%2F2'),` +
				` {cache: 'no-cache'});"><div>hello</div><img src="//prebid-test.smaatolabs.net/track/imp/1" alt="" width="0" height="0"/>` +
				`<img src="//prebid-test.smaatolabs.net/track/imp/2" alt="" width="0" height="0"/></div>`,
			expectedError: "",
		},
		{
			testName:         "invalid adMarkup",
			adMarkup:         "{",
			expectedAdMarkup: "",
			expectedError:    "Invalid ad markup {.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			adMarkup, err := extractAdmRichMedia(tt.adMarkup)

			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedAdMarkup, adMarkup)
		})
	}
}
