package smaato

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "smaatotest", NewSmaatoBidder(nil, "https://prebid-test.smaatolabs.net/bidder"))
}

func Test_getADM(t *testing.T) {
	type args struct {
		adType             string
		adapterResponseAdm string
	}
	tests := []struct {
		testName string
		args     args
		result   string
	}{
		{"nonImageTest", args{" ", "<div>mytestadd</div>"}, "<div>mytestadd</div>"},
		{"imageTest", args{"img", "{\"image\":{\"img\":{\"url\":\"//prebid-test.smaatolabs.net/img/320x50.jpg\"," +
			"\"w\":350,\"h\":50,\"ctaurl\":\"//prebid-test.smaatolabs.net/track/ctaurl/1\"},\"impressiontrackers\":[\"//prebid-test.smaatolabs.net/track/imp/1\",\"//prebid-test.smaatolabs.net/track/imp/2\"],\"clicktrackers\":[\"//prebid-test.smaatolabs.net/track/click/1\",\"//prebid-test.smaatolabs.net/track/click/2\"]}}"},
			"<div onclick=\"fetch(decodeURIComponent(%2F%2Fprebid-test." +
				"smaatolabs.net%2Ftrack%2Fclick%2F1), {cache: 'no-cache'});fetch(decodeURIComponent(%2F%2Fprebid-test." +
				"smaatolabs.net%2Ftrack%2Fclick%2F2), {cache: 'no-cache'});\"><a href=//prebid-test.smaatolabs." +
				"net/track/ctaurl/1><img src=//prebid-test.smaatolabs.net/img/320x50." +
				"jpg width=350 height=50/></a></div>"},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			got, _ := getADM(tt.args.adType, tt.args.adapterResponseAdm)
			if got != tt.result {
				t.Errorf("getADM() got = %v, result %v", got, tt.result)
			}
		})
	}
}
