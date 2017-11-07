package openrtb_ext

import (
	"testing"
	"github.com/mxmCherry/openrtb"
	"encoding/json"
)

func TestRandomizeList(t *testing.T) {
	adapters := make([]BidderName, 3)
	adapters[0] = BidderName("dummy")
	adapters[1] = BidderName("dummy2")
	adapters[2] = BidderName("dummy3")

	RandomizeList(adapters)

	if len(adapters) != 3 {
		t.Errorf("RondomizeList, expected a list of 3, found %d", len(adapters))
	}

	adapters = adapters[0:1]
	RandomizeList(adapters)

	if len(adapters) != 1 {
		t.Errorf("RondomizeList, expected a list of 1, found %d", len(adapters))
	}

}

func TestCleanOpenRTBRequests(t *testing.T) {
	// Very simple Bid request. The dummy bidders know what to do.
	bidRequest := openrtb.BidRequest{
		ID: "This Bid",
		Imp: make([]openrtb.Imp, 2),
	}
	// Need extensions for all the bidders so we know to hold auctions for them.
	impExt := make(map[string]interface{})
	impExt["dummy"] = make(map[string]string)
	impExt["dummy2"] = make(map[string]string)
	impExt["dummy3"] = make(map[string]string)
	b, err := json.Marshal(impExt)
	if err != nil {
		t.Errorf("Error Mashalling bidRequest Extants: %s", err.Error())
	}
	bidRequest.Imp[0].Ext = b
	bidRequest.Imp[1].Ext = b

	adapters := make([]BidderName, 3)
	adapters[0] = BidderName("dummy")
	adapters[1] = BidderName("dummy2")
	adapters[2] = BidderName("dummy3")

	cleanRequests := CleanOpenRTBRequests( &bidRequest, adapters)

	if len(cleanRequests) != 3 {
		t.Errorf("CleanOpenRTBRequests: expected 3 requests, found %d", len(cleanRequests))
	}
}

