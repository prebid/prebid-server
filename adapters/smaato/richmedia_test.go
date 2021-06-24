package smaato

import (
	"testing"
)

func TestExtractAdmRichMedia(t *testing.T) {
	type args struct {
		adMarkup string
	}
	tests := []struct {
		testName         string
		args             args
		expectedAdMarkup string
		expectedError    string
	}{
		{"extract richmedia",
			args{"{\"richmedia\":{\"mediadata\":{\"content\":\"<div>hello</div>\"," +
				"" + "\"w\":350," +
				"\"h\":50},\"impressiontrackers\":[\"//prebid-test.smaatolabs.net/track/imp/1\",\"//prebid-test.smaatolabs.net/track/imp/2\"]," +
				"\"clicktrackers\":[\"//prebid-test.smaatolabs.net/track/click/1\",\"//prebid-test.smaatolabs.net/track/click/2\"]}}"},
			`<div onclick="fetch(decodeURIComponent('%2F%2Fprebid-test.smaatolabs.net%2Ftrack%2Fclick%2F1'),` +
				` {cache: 'no-cache'});fetch(decodeURIComponent('%2F%2Fprebid-test.smaatolabs.net%2Ftrack%2Fclick%2F2'),` +
				` {cache: 'no-cache'});"><div>hello</div><img src="//prebid-test.smaatolabs.net/track/imp/1" alt="" width="0" height="0"/>` +
				`<img src="//prebid-test.smaatolabs.net/track/imp/2" alt="" width="0" height="0"/></div>`,
			"",
		},
		{"invalid adMarkup",
			args{"{"},
			"",
			"Invalid ad markup {.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			adMarkup, err := extractAdmRichMedia(tt.args.adMarkup)
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("extractAdmRichMedia() expectedError %v", tt.expectedError)
				} else if err.Error() != tt.expectedError {
					t.Errorf("extractAdmRichMedia() err = %v, expectedError %v", err, tt.expectedError)
				}
			} else if err != nil {
				t.Errorf("extractAdmRichMedia() unexpected err = %v", err)
			}

			if adMarkup != tt.expectedAdMarkup {
				t.Errorf("extractAdmRichMedia() adMarkup = %v, expectedAdMarkup %v", adMarkup, tt.expectedAdMarkup)
			}
		})
	}
}
