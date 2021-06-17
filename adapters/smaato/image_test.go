package smaato

import (
	"testing"
)

func TestExtractAdmImage(t *testing.T) {
	type args struct {
		adMarkup string
	}
	tests := []struct {
		testName         string
		args             args
		expectedAdMarkup string
		expectedError    string
	}{
		{"extract image",
			args{"{\"image\":{\"img\":{\"url\":\"//prebid-test.smaatolabs.net/img/320x50.jpg\"," +
				"\"w\":350,\"h\":50,\"ctaurl\":\"//prebid-test.smaatolabs.net/track/ctaurl/1\"}," +
				"\"impressiontrackers\":[\"//prebid-test.smaatolabs.net/track/imp/1\",\"//prebid-test.smaatolabs.net/track/imp/2\"]," +
				"\"clicktrackers\":[\"//prebid-test.smaatolabs.net/track/click/1\",\"//prebid-test.smaatolabs.net/track/click/2\"]}}"},
			`<div style="cursor:pointer"` +
				` onclick="fetch(decodeURIComponent('%2F%2Fprebid-test.smaatolabs.net%2Ftrack%2Fclick%2F1'.replace(/\+/g, ' ')),` +
				` {cache: 'no-cache'});fetch(decodeURIComponent('%2F%2Fprebid-test.smaatolabs.net%2Ftrack%2Fclick%2F2'.replace(/\+/g, ' ')),` +
				` {cache: 'no-cache'});;window.open(decodeURIComponent('%2F%2Fprebid-test.smaatolabs.net%2Ftrack%2Fctaurl%2F1'.replace(/\+/g, ' ')));">` +
				`<img src="//prebid-test.smaatolabs.net/img/320x50.jpg" width="350" height="50"/>` +
				`<img src="//prebid-test.smaatolabs.net/track/imp/1" alt="" width="0" height="0"/>` +
				`<img src="//prebid-test.smaatolabs.net/track/imp/2" alt="" width="0" height="0"/></div>`,
			"",
		},
		{"invalid adMarkup",
			args{"{"},
			"",
			"unexpected end of JSON input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			adMarkup, err := extractAdmImage(tt.args.adMarkup)
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("extractAdmImage() expectedError %v", tt.expectedError)
				} else if err.Error() != tt.expectedError {
					t.Errorf("extractAdmImage() err = %v, expectedError %v", err, tt.expectedError)
				}
			} else if err != nil {
				t.Errorf("extractAdmImage() unexpected err = %v", err)
			}

			if adMarkup != tt.expectedAdMarkup {
				t.Errorf("extractAdmImage() adMarkup = %v, expectedAdMarkup %v", adMarkup, tt.expectedAdMarkup)
			}
		})
	}
}
