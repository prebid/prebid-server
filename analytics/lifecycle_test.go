package analytics_test

import (
	"testing"

	"github.com/prebid/prebid-server/v4/analytics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuctionLifecycleWithoutOptions(t *testing.T) {
	assert.Nil(t, analytics.NewAuctionLifecycle(nil))
	assert.Nil(t, analytics.NewAuctionLifecycle(map[string]*analytics.LifecycleOptions{}))
	assert.Nil(t, analytics.NewAuctionLifecycle(map[string]*analytics.LifecycleOptions{"a": nil}))
}

func TestAuctionLifecycleKeepsRequestBodyForAnyModule(t *testing.T) {
	lifecycle := analytics.NewAuctionLifecycle(map[string]*analytics.LifecycleOptions{
		"a": {RequestBodyBidders: []string{"appnexus"}},
		"b": {RequestBodyBidders: []string{"rubicon"}},
		"c": {},
	})

	assert.True(t, lifecycle.KeepsRequestBody("appnexus"))
	assert.True(t, lifecycle.KeepsRequestBody("rubicon"))
	assert.False(t, lifecycle.KeepsRequestBody("pubmatic"))
}

func TestAuctionLifecycleNilIsSafe(t *testing.T) {
	var lifecycle *analytics.AuctionLifecycle

	assert.False(t, lifecycle.KeepsRequestBody("appnexus"))
	lifecycle.RecordFiltered("appnexus", "imp1", analytics.RequestBlockedPrivacy, "gdpr")
	lifecycle.Merge(&analytics.AuctionLifecycle{})
	assert.Nil(t, lifecycle.ForModule("module", false))
}

func TestAuctionLifecycleRecordFilteredAndMerge(t *testing.T) {
	lifecycle := &analytics.AuctionLifecycle{}
	bidder := &analytics.AuctionLifecycle{}
	bidder.RecordFiltered("rubicon", "imp2", analytics.RequestBlockedUnsupportedMediaType, "mediaType")
	bidder.BidderRequestDetails = append(bidder.BidderRequestDetails, analytics.BidderRequestDetail{BidderRequestID: "rubicon-0", Bidder: "rubicon"})
	bidder.Bids = append(bidder.Bids, analytics.BidResult{BidderRequestID: "rubicon-0", BidID: "bid-1", ImpID: "imp2"})

	lifecycle.RecordFiltered("appnexus", "imp1", analytics.RequestBlockedPrivacy, "gdpr")
	lifecycle.Merge(bidder)
	lifecycle.Merge(nil)

	assert.Equal(t, []analytics.FilteredBidderImpDetail{
		{Bidder: "appnexus", ImpID: "imp1", Scope: analytics.FilterScopeBidder, ReasonCode: analytics.RequestBlockedPrivacy, ReasonDetail: "gdpr"},
		{Bidder: "rubicon", ImpID: "imp2", Scope: analytics.FilterScopeImpression, ReasonCode: analytics.RequestBlockedUnsupportedMediaType, ReasonDetail: "mediaType"},
	}, lifecycle.FilteredBidderImpDetails)
	assert.Equal(t, []analytics.BidderRequestDetail{{BidderRequestID: "rubicon-0", Bidder: "rubicon"}}, lifecycle.BidderRequestDetails)
	assert.Equal(t, []analytics.BidResult{{BidderRequestID: "rubicon-0", BidID: "bid-1", ImpID: "imp2"}}, lifecycle.Bids)
}

func TestAuctionLifecycleForModule(t *testing.T) {
	lifecycle := analytics.NewAuctionLifecycle(map[string]*analytics.LifecycleOptions{
		"modA": {RequestBodyBidders: []string{"appnexus"}},
		"modB": {RequestBodyBidders: []string{"rubicon"}},
	})
	lifecycle.BidderRequestDetails = []analytics.BidderRequestDetail{
		{BidderRequestID: "appnexus-0", Bidder: "appnexus", RequestBody: []byte(`{"id":"1"}`)},
		{BidderRequestID: "rubicon-0", Bidder: "rubicon", RequestBody: []byte(`{"id":"2"}`)},
	}

	assert.Nil(t, lifecycle.ForModule("other", false))

	viewA := lifecycle.ForModule("modA", false)
	require.NotNil(t, viewA)
	assert.False(t, viewA.RequestBodiesWithheld)
	assert.Equal(t, []byte(`{"id":"1"}`), viewA.BidderRequestDetails[0].RequestBody)
	assert.Nil(t, viewA.BidderRequestDetails[1].RequestBody)

	viewB := lifecycle.ForModule("modB", false)
	require.NotNil(t, viewB)
	assert.Nil(t, viewB.BidderRequestDetails[0].RequestBody)
	assert.Equal(t, []byte(`{"id":"2"}`), viewB.BidderRequestDetails[1].RequestBody)
}

func TestAuctionLifecycleForModuleWithholdsRequestBodiesWithoutChangingOriginal(t *testing.T) {
	lifecycle := analytics.NewAuctionLifecycle(map[string]*analytics.LifecycleOptions{
		"recorder": {RequestBodyBidders: []string{"appnexus"}},
	})
	lifecycle.BidderRequestDetails = []analytics.BidderRequestDetail{{BidderRequestID: "appnexus-0", Bidder: "appnexus", RequestBody: []byte(`{"id":"1"}`)}}

	received := lifecycle.ForModule("recorder", true)
	require.NotNil(t, received)

	assert.True(t, received.RequestBodiesWithheld)
	assert.Equal(t, []analytics.BidderRequestDetail{{BidderRequestID: "appnexus-0", Bidder: "appnexus"}}, received.BidderRequestDetails)
	assert.False(t, lifecycle.RequestBodiesWithheld)
	assert.Equal(t, []byte(`{"id":"1"}`), lifecycle.BidderRequestDetails[0].RequestBody)
}
