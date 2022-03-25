package smaato

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractAdmImage(t *testing.T) {
	tests := []struct {
		testName         string
		adMarkup         string
		expectedAdMarkup string
		expectedError    string
	}{
		{
			testName: "extract image",
			adMarkup: "{\"image\":{\"img\":{\"url\":\"//prebid-test.smaatolabs.net/img/320x50.jpg\"," +
				"\"w\":350,\"h\":50,\"ctaurl\":\"//prebid-test.smaatolabs.net/track/ctaurl/1\"}," +
				"\"impressiontrackers\":[\"//prebid-test.smaatolabs.net/track/imp/1\",\"//prebid-test.smaatolabs.net/track/imp/2\"]," +
				"\"clicktrackers\":[\"//prebid-test.smaatolabs.net/track/click/1\",\"//prebid-test.smaatolabs.net/track/click/2\"]}}",
			expectedAdMarkup: `<div style="cursor:pointer"` +
				` onclick="fetch(decodeURIComponent('%2F%2Fprebid-test.smaatolabs.net%2Ftrack%2Fclick%2F1'.replace(/\+/g, ' ')),` +
				` {cache: 'no-cache'});fetch(decodeURIComponent('%2F%2Fprebid-test.smaatolabs.net%2Ftrack%2Fclick%2F2'.replace(/\+/g, ' ')),` +
				` {cache: 'no-cache'});;window.open(decodeURIComponent('%2F%2Fprebid-test.smaatolabs.net%2Ftrack%2Fctaurl%2F1'.replace(/\+/g, ' ')));">` +
				`<img src="//prebid-test.smaatolabs.net/img/320x50.jpg" width="350" height="50"/>` +
				`<img src="//prebid-test.smaatolabs.net/track/imp/1" alt="" width="0" height="0"/>` +
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
			adMarkup, err := extractAdmImage(tt.adMarkup)

			if tt.expectedError != "" {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedAdMarkup, adMarkup)
		})
	}
}
