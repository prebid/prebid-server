package clickio

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderClickio, config.Adapter{
		Endpoint: "https://ssp.clickio.example/openrtb2/auction",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "clickiotest", bidder)
}

func TestMakeRequestsDoesNotMutateInputRequest(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderClickio, config.Adapter{
		Endpoint: "https://ssp.clickio.example/openrtb2/auction",
	}, config.Server{})
	if err != nil {
		t.Fatalf("Builder returned unexpected error %v", err)
	}

	req := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{
				ID: "imp-1",
				Ext: json.RawMessage(`{"bidder":{"said":"auction-1"}}`),
			},
		},
	}

	_, errs := bidder.MakeRequests(req, nil)
	if len(errs) > 0 {
		t.Fatalf("MakeRequests returned unexpected errors %v", errs)
	}

	if string(req.Imp[0].Ext) != `{"bidder":{"said":"auction-1"}}` {
		t.Fatalf("input request was mutated: %s", string(req.Imp[0].Ext))
	}
}

func TestUpdateImpExtWithParamsDoesNotUseOtherBidderFromPrebid(t *testing.T) {
	imp := &openrtb2.Imp{
		ID:  "imp-1",
		Ext: json.RawMessage(`{"prebid":{"bidder":{"other":{"said":"auction-1"}}}}`),
	}

	if err := updateImpExtWithParams(imp); err != nil {
		t.Fatalf("updateImpExtWithParams returned unexpected error: %v", err)
	}

	var ext map[string]json.RawMessage
	if err := json.Unmarshal(imp.Ext, &ext); err != nil {
		t.Fatalf("failed to unmarshal imp ext: %v", err)
	}
	if _, ok := ext["params"]; ok {
		t.Fatalf("unexpected params copied from non-clickio bidder: %s", string(imp.Ext))
	}
}
