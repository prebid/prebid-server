package openrtb_ext

import (
	"testing"
	"github.com/mxmCherry/openrtb"
	"encoding/json"
	"fmt"
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
	dummy1Ext := make(map[string]string)
	dummy1Ext["dummy"] = `{placementId:"5554444"}`
	dummy1Ext["dummy2"] = `{accountId:"abc"}`
	impExt["dummy"] = dummy1Ext
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

	cleanRequests, errList := CleanOpenRTBRequests( &bidRequest, adapters)

	if len(errList) > 0 {
		for _, e := range errList {
			t.Errorf("CleanOpenRTBRequests: %s", e.Error())
		}
	}
	if len(cleanRequests) != 3 {
		t.Errorf("CleanOpenRTBRequests: expected 3 requests, found %d", len(cleanRequests))
	}

	var cleanImpExt map[string]map[string]string
	err = json.Unmarshal(cleanRequests[BidderName("dummy")].Imp[0].Ext, &cleanImpExt)
	fmt.Println(string(cleanRequests[BidderName("dummy")].Imp[0].Ext))
	if err != nil {
		t.Errorf("CleanOpenRTBRequests: %s", err.Error())
	}
	dummymap, ok := cleanImpExt["dummy"]
	if ! ok {
		t.Error("CleanOpenRTBRequests: dummy adapter did not get proper dummy extension")
	}
	if dummymap["placementId"] != "5554444" {
		t.Errorf("CleanOpenRTBRequests: dummy adapter did not get proper placementId, got \"%s\" instead", cleanImpExt["dummy"]["placementId"])
	}
}

