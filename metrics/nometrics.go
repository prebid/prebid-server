package metrics

// This file provides a no-op implementation of PBSMetrics.
// The server code can use this if it doesn't want to export metrics anywhere.

func NewNilMetrics() PBSMetrics {
	return &nilPBSMetrics{}
}

type nilPBSMetrics struct{}

func (m *nilPBSMetrics) StartAuctionRequest(requestInfo *AuctionRequestInfo) AuctionRequestFollowups {
	return &nilAuctionRequestFollowups{}
}

func (m *nilPBSMetrics) StartBidderRequest(auctionRequestInfo *AuctionRequestInfo, bidRequestInfo *BidRequestInfo) BidderRequestFollowups {
	return &nilBidderRequestFollowups{}
}

func (m *nilPBSMetrics) StartUserSyncRequest() UserSyncFollowups {
	return &nilUserSyncFollowups{}
}

func (m *nilPBSMetrics) StartCookieSyncRequest() {}

type nilAuctionRequestFollowups struct{}

func (arf *nilAuctionRequestFollowups) Completed(err error) {}

type nilBidderRequestFollowups struct{}

func (brf *nilBidderRequestFollowups) BidderSkipped() {}

func (brf *nilBidderRequestFollowups) BidderResponded(bidPrices []float64, err error) {}

type nilUserSyncFollowups struct{}

func (brf *nilUserSyncFollowups) UserOptedOut() {}

func (brf *nilUserSyncFollowups) BadRequest() {}

func (f *nilUserSyncFollowups) Completed(bidder string, err error) {}
