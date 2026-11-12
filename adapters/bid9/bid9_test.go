package bid9

import (
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBid9, config.Adapter{
		Endpoint: "http://test.endpoint/sspRTB2"}, config.Server{ExternalUrl: "http://hosturl.com"})
	require.NoError(t, buildErr)

	adapterstest.RunJSONBidderTest(t, "bid9test", bidder)
}

func TestBuilderInvalidEndpoint(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderBid9, config.Adapter{Endpoint: "http://bad host/%zz"}, config.Server{})

	assert.Error(t, buildErr)
}

func TestEndpointQueryKept(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBid9, config.Adapter{
		Endpoint: "http://localhost:8099/openrtb2/auction?region=eu"}, config.Server{})
	require.NoError(t, buildErr)

	requests, errs := bidder.MakeRequests(&openrtb2.BidRequest{
		ID:  "req-1",
		Imp: []openrtb2.Imp{{ID: "imp-1", Ext: []byte(`{"bidder":{"channelId":"chan-a"}}`)}},
	}, nil)

	assert.Empty(t, errs)
	require.Len(t, requests, 1)
	assert.Equal(t, "http://localhost:8099/openrtb2/auction?region=eu&tagid=chan-a", requests[0].Uri)
}

func TestPublisherIDNotLeakedToSharedRequest(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBid9, config.Adapter{Endpoint: "http://test.endpoint/sspRTB2"}, config.Server{})
	require.NoError(t, buildErr)

	publisher := &openrtb2.Publisher{Name: "pub name"}
	site := &openrtb2.Site{Domain: "publisher.test", Publisher: publisher}
	imp := openrtb2.Imp{ID: "imp-1", Ext: []byte(`{"bidder":{"channelId":"chan-a","publisherId":"pub-1"}}`)}
	request := &openrtb2.BidRequest{ID: "req-1", Site: site, Imp: []openrtb2.Imp{imp}}

	requests, errs := bidder.MakeRequests(request, nil)

	assert.Empty(t, errs)
	require.Len(t, requests, 1)
	assert.Contains(t, string(requests[0].Body), `"id":"pub-1"`)
	assert.Same(t, site, request.Site)
	assert.Same(t, publisher, request.Site.Publisher)
	assert.Empty(t, publisher.ID)
	assert.Empty(t, request.Imp[0].TagID)
	assert.JSONEq(t, `{"bidder":{"channelId":"chan-a","publisherId":"pub-1"}}`, string(request.Imp[0].Ext))
}

func TestMakeBidsInvalidRequestBody(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBid9, config.Adapter{Endpoint: "http://test.endpoint/sspRTB2"}, config.Server{})
	require.NoError(t, buildErr)

	response, errs := bidder.MakeBids(&openrtb2.BidRequest{}, &adapters.RequestData{Body: []byte("not json")},
		&adapters.ResponseData{StatusCode: http.StatusOK, Body: []byte(`{"id":"req-1"}`)})

	assert.Nil(t, response)
	assert.Len(t, errs, 1)
}
