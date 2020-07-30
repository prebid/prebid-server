package smaato

import (
	"testing"
)

func TestExtractAdmRichMedia(t *testing.T) {
	type args struct {
		adType             adMarkupType
		adapterResponseAdm string
	}
	expectedResult := `<div onclick="fetch(decodeURIComponent('%2F%2Fprebid-test.smaatolabs.net%2Ftrack%2Fclick%2F1'),` +
		` {cache: 'no-cache'});fetch(decodeURIComponent('%2F%2Fprebid-test.smaatolabs.net%2Ftrack%2Fclick%2F2'),` +
		` {cache: 'no-cache'});"><div>hello</div><img src="//prebid-test.smaatolabs.net/track/imp/1" alt="" width="0" height="0"/>` +
		`<img src="//prebid-test.smaatolabs.net/track/imp/2" alt="" width="0" height="0"/></div>`
	tests := []struct {
		testName string
		args     args
		result   string
	}{
		{"richmediaTest", args{"Richmedia", "{\"richmedia\":{\"mediadata\":{\"content\":\"<div>hello</div>\"," +
			"" + "\"w\":350," +
			"\"h\":50},\"impressiontrackers\":[\"//prebid-test.smaatolabs.net/track/imp/1\",\"//prebid-test.smaatolabs.net/track/imp/2\"]," +
			"\"clicktrackers\":[\"//prebid-test.smaatolabs.net/track/click/1\",\"//prebid-test.smaatolabs.net/track/click/2\"]}}"},
			expectedResult,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			got, err := renderAdMarkup(tt.args.adType, tt.args.adapterResponseAdm)
			if err != nil {
				t.Errorf("error rendering ad markup: %v", err)
			}
			if got != tt.result {
				t.Errorf("renderAdMarkup() got = %v, result %v", got, tt.result)
			}
		})
	}
}
