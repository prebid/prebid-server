package superedge

import (
	"encoding/json"
	"testing"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderSuperEdge, config.Adapter{
		Endpoint: "https://rtb-us.superedge.co.jp/bid?sk={{.sk}}"}, config.Server{ExternalUrl: "http://hosturl.com"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "superedgetest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderSuperEdge, config.Adapter{Endpoint: "{{Malformed}}"}, config.Server{ExternalUrl: "http://hosturl.com"})

	assert.Error(t, buildErr)
}

func TestGetSuperEdgeExtEmptyImp(t *testing.T) {
	request := &openrtb2.BidRequest{}
	_, err := getSuperEdgeExt(request)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sk not found")
}

func TestMakeRequestGetEndPointError(t *testing.T) {
	badTmpl, _ := template.New("").Parse("http://example.com/bid?sk={{.sk}}{{template \"nonexistent\" .}}")
	a := &adapter{EndpointTemplate: badTmpl}

	request := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{
				ID:  "test-imp-id",
				Ext: json.RawMessage(`{"bidder":{"sk":"test-sk"}}`),
			},
		},
	}

	_, err := a.makeRequest(request)
	assert.Error(t, err)
}

func TestGetBidTypeAmbiguous(t *testing.T) {
	bid := openrtb2.Bid{ImpID: "imp-1"}
	imps := []openrtb2.Imp{
		{
			ID:     "imp-1",
			Banner: &openrtb2.Banner{},
			Native: &openrtb2.Native{},
		},
	}
	_, err := getBidType(bid, imps)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported MType")
}

func TestGetBidTypeNoMatchingImp(t *testing.T) {
	bid := openrtb2.Bid{ImpID: "imp-1"}
	imps := []openrtb2.Imp{
		{ID: "imp-2", Banner: &openrtb2.Banner{}},
	}
	_, err := getBidType(bid, imps)
	assert.Error(t, err)
}

func TestGetBidTypeNoMediaType(t *testing.T) {
	bid := openrtb2.Bid{ImpID: "imp-1"}
	imps := []openrtb2.Imp{
		{ID: "imp-1"},
	}
	_, err := getBidType(bid, imps)
	assert.Error(t, err)
}
