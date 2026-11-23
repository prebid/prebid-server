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

	response, errs := bidder.MakeBids(&openrtb2.BidRequest{}, &adapters.RequestData{Uri: "http://test.endpoint/sspRTB2?tagid=chan-a", Body: []byte("not json"), ImpIDs: []string{"imp-1"}},
		&adapters.ResponseData{StatusCode: http.StatusOK, Body: []byte(`{"id":"req-1"}`)})

	assert.Nil(t, response)
	assert.Len(t, errs, 1)
}

// A stored bid response arrives with an empty URI, no ImpIDs and a text body, and its imp is not in the request.
func TestMakeBidsStoredResponse(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBid9, config.Adapter{Endpoint: "http://test.endpoint/sspRTB2"}, config.Server{})
	require.NoError(t, buildErr)

	response, errs := bidder.MakeBids(&openrtb2.BidRequest{ID: "auction-1"},
		&adapters.RequestData{Body: []byte("Stored bid response for impression id: imp-1")},
		&adapters.ResponseData{StatusCode: http.StatusOK, Body: []byte(`{"id":"auction-1","seatbid":[{"bid":[
			{"id":"b1","impid":"imp-1","price":1.5,"adm":"<div>ad</div>","nurl":"https://t.test/win?a=${AUCTION_ID}&s=${AUCTION_SEAT_ID}","mtype":1},
			{"id":"b2","impid":"imp-1","price":2,"adm":"<VAST version=\"4.0\"></VAST>"},
			{"id":"b3","impid":"imp-1","price":1,"adm":"<div>ad</div>"}]}]}`)})

	assert.Empty(t, errs)
	require.Len(t, response.Bids, 3)
	assert.Equal(t, openrtb_ext.BidTypeBanner, response.Bids[0].BidType)
	assert.Equal(t, "https://t.test/win?a=auction-1", response.Bids[0].Bid.BURL)
	assert.Equal(t, openrtb_ext.BidTypeVideo, response.Bids[1].BidType)
	assert.Equal(t, openrtb_ext.BidTypeBanner, response.Bids[2].BidType)
}

func TestRemoveUnresolvedMacros(t *testing.T) {
	cases := map[string]string{
		"https://t.test/w?s=${AUCTION_SEAT_ID}&p=1":                        "https://t.test/w?p=1",
		"https://t.test/w?p=1&s=${AUCTION_SEAT_ID}":                        "https://t.test/w?p=1",
		"https://t.test/w?p=1&s=${AUCTION_SEAT_ID}&x=2":                    "https://t.test/w?p=1&x=2",
		"https://t.test/w?s=${AUCTION_SEAT_ID}":                            "https://t.test/w",
		"<a href='https://t.test/c?s=${AUCTION_SEAT_ID}&amp;b=2'>":         "<a href='https://t.test/c?b=2'>",
		"<a href='https://t.test/c?a=1&amp;s=${AUCTION_SEAT_ID}&amp;b=2'>": "<a href='https://t.test/c?a=1&amp;b=2'>",
		"<a href='https://t.test/c?a=1&amp;s=${AUCTION_SEAT_ID}'>":         "<a href='https://t.test/c?a=1'>",
		"https://t.test/w?s=${AUCTION_SEAT_ID}&d=${AUCTION_AD_ID}&p=1":     "https://t.test/w?p=1",
		"https://t.test/${AUCTION_LOSS}/x?p=1":                             "https://t.test//x?p=1",
		"no macros here":                                                   "no macros here",
	}
	for in, want := range cases {
		assert.Equal(t, want, removeUnresolvedMacros(in), in)
	}
}

func TestDeviceHeadersForwarded(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBid9, config.Adapter{Endpoint: "http://test.endpoint/sspRTB2"}, config.Server{})
	require.NoError(t, buildErr)

	requests, errs := bidder.MakeRequests(&openrtb2.BidRequest{
		ID:     "req-1",
		App:    &openrtb2.App{Bundle: "com.publisher.app"},
		Device: &openrtb2.Device{UA: "test-ua", IP: "203.0.113.7", IPv6: "2001:db8::1"},
		Imp:    []openrtb2.Imp{{ID: "imp-1", Video: &openrtb2.Video{MIMEs: []string{"video/mp4"}}, Ext: []byte(`{"bidder":{"channelId":"chan-a"}}`)}},
	}, nil)

	assert.Empty(t, errs)
	require.Len(t, requests, 1)
	assert.Equal(t, "test-ua", requests[0].Headers.Get("User-Agent"))
	assert.Equal(t, []string{"2001:db8::1", "203.0.113.7"}, requests[0].Headers.Values("X-Forwarded-For"))
}
