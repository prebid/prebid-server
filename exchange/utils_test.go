package exchange

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

func TestRandomizeList(t *testing.T) {
	adapters := make([]openrtb_ext.BidderName, 3)
	adapters[0] = openrtb_ext.BidderName("dummy")
	adapters[1] = openrtb_ext.BidderName("dummy2")
	adapters[2] = openrtb_ext.BidderName("dummy3")

	randomizeList(adapters)

	if len(adapters) != 3 {
		t.Errorf("RondomizeList, expected a list of 3, found %d", len(adapters))
	}

	adapters = adapters[0:1]
	randomizeList(adapters)

	if len(adapters) != 1 {
		t.Errorf("RondomizeList, expected a list of 1, found %d", len(adapters))
	}

}

func TestCleanOpenRTBRequests(t *testing.T) {
	// Very simple Bid request. The dummy bidders know what to do.
	bidRequest := openrtb.BidRequest{
		ID:  "This Bid",
		Imp: make([]openrtb.Imp, 2),
	}
	// Need extensions for all the bidders so we know to hold auctions for them.
	impExt := make(map[string]interface{})
	dummy1Ext := make(map[string]string)
	dummy2Ext := make(map[string]string)
	dummy3Ext := make(map[string]string)
	dummy1Ext["placementId"] = "5554444"
	dummy2Ext["accountID"] = "abc"
	dummy3Ext["placementId"] = "1234567"
	impExt["dummy"] = dummy1Ext
	impExt["dummy2"] = dummy2Ext
	impExt["dummy3"] = dummy3Ext

	b, err := json.Marshal(impExt)
	if err != nil {
		t.Errorf("Error Mashalling bidRequest Extants: %s", err.Error())
	}
	bidRequest.Imp[0].Ext = b
	bidRequest.Imp[1].Ext = b

	adapters := make([]openrtb_ext.BidderName, 3)
	adapters[0] = openrtb_ext.BidderName("dummy")
	adapters[1] = openrtb_ext.BidderName("dummy2")
	adapters[2] = openrtb_ext.BidderName("dummy3")

	cleanRequests, errList := cleanOpenRTBRequests(&bidRequest, adapters, &emptyUsersync{})

	if len(errList) > 0 {
		for _, e := range errList {
			t.Errorf("CleanOpenRTBRequests: %s", e.Error())
		}
	}
	if len(cleanRequests) != 3 {
		t.Errorf("CleanOpenRTBRequests: expected 3 requests, found %d", len(cleanRequests))
	}

	var cleanImpExt map[string]map[string]string
	err = json.Unmarshal(cleanRequests[openrtb_ext.BidderName("dummy")].Imp[0].Ext, &cleanImpExt)
	if err != nil {
		t.Errorf("CleanOpenRTBRequests: %s", err.Error())
	}
	dummymap, ok := cleanImpExt["bidder"]
	if !ok {
		t.Error("CleanOpenRTBRequests: dummy adapter did not get proper bidder extension")
	}
	if dummymap["placementId"] != "5554444" {
		t.Errorf("CleanOpenRTBRequests: dummy adapter did not get proper placementId, got \"%s\" instead", cleanImpExt["dummy"]["placementId"])
	}
	_, ok = dummymap["accountID"]
	if ok {
		t.Error("CleanOpenRTBRequests: dummy adapter got dummy2 parameter")
	}
	err = json.Unmarshal(cleanRequests[openrtb_ext.BidderName("dummy3")].Imp[0].Ext, &cleanImpExt)
	if err != nil {
		t.Errorf("CleanOpenRTBRequests: %s", err.Error())
	}
	dummymap, ok = cleanImpExt["bidder"]
	if !ok {
		t.Error("CleanOpenRTBRequests: dummy3 adapter did not get proper bidder extension")
	}
	if dummymap["placementId"] != "1234567" {
		t.Errorf("CleanOpenRTBRequests: dummy3 adapter did not get proper placementId, got \"%s\" instead", cleanImpExt["dummy"]["placementId"])
	}

}

type emptyUsersync struct{}

func (e *emptyUsersync) GetId(bidder openrtb_ext.BidderName) (string, bool) {
	return "", false
}
