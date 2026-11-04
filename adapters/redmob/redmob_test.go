package redmob

import (
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderRedmob, config.Adapter{
		Endpoint: "https://test.redmob.example/prebid"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "redmobtest", bidder)
}

func TestMakeBidsSetsAdvertiserDomains(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderRedmob, config.Adapter{
		Endpoint: "https://test.redmob.example/prebid"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	require.NoError(t, buildErr)

	request := &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "1", Banner: &openrtb2.Banner{}}}}
	body := `{"id":"req-1","cur":"USD","seatbid":[{"seat":"41","bid":[
		{"id":"bid-1","impid":"1","price":0.5,"crid":"cr-1","adomain":["brand.example.com"]},
		{"id":"bid-2","impid":"1","price":0.4,"crid":"cr-2"}]}]}`

	bidResponse, errs := bidder.MakeBids(request, &adapters.RequestData{ImpIDs: []string{"1"}},
		&adapters.ResponseData{StatusCode: http.StatusOK, Body: []byte(body)})

	assert.Empty(t, errs)
	require.Len(t, bidResponse.Bids, 2)
	require.NotNil(t, bidResponse.Bids[0].BidMeta)
	assert.Equal(t, []string{"brand.example.com"}, bidResponse.Bids[0].BidMeta.AdvertiserDomains)
	assert.Nil(t, bidResponse.Bids[1].BidMeta, "no adomain means no meta")
}
