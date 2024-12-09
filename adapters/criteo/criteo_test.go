package criteo

import (
	"fmt"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {

	bidder, buildErr := Builder(openrtb_ext.BidderCriteo, config.Adapter{
		Endpoint: "https://ssp-bidder.criteo.com/openrtb/pbs/auction/request?profile=230"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Execute & Verify:
	adapterstest.RunJSONBidderTest(t, "criteotest", bidder)
}

func TestParseFledgeAuctionConfigs_Nil(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderCriteo, config.Adapter{
		Endpoint: "https://ssp-bidder.criteo.com/openrtb/pbs/auction/request?profile=230"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	var tests = []struct {
		name        string
		bidResponse openrtb2.BidResponse
	}{
		{"no ext", openrtb2.BidResponse{Ext: nil}},
		{"no igi", openrtb2.BidResponse{Ext: []byte("{}")}},
		{"igi empty", openrtb2.BidResponse{Ext: []byte(`{"igi":[]}`)}},
		{"no igs", openrtb2.BidResponse{Ext: []byte(`{"igi":[{}]}`)}},
		{"igs empty", openrtb2.BidResponse{Ext: []byte(`{"igi":[{"impid": "1", "igs": []}]}`)}},
		{"no config", openrtb2.BidResponse{Ext: []byte(`{"igi":[{"impid": "1", "igs": [{}]}]}`)}},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("%s", tt.name)
		t.Run(testname, func(t *testing.T) {
			fledgeAuctionConfigs := bidder.(*adapter).ParseFledgeAuctionConfigs(tt.bidResponse)

			assert.Nil(t, fledgeAuctionConfigs)
		})
	}
}
