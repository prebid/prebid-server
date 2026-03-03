package adquery

import "github.com/prebid/prebid-server/v3/openrtb_ext"

type BidderRequest struct {
	V                   string `json:"v"`
	PlacementCode       string `json:"placementCode"`
	AuctionId           string `json:"auctionId,omitempty"`
	BidType             string `json:"type"`
	AdUnitCode          string `json:"adUnitCode"`
	BidQid              string `json:"bidQid"`
	BidId               string `json:"bidId"`
	BidIp               string `json:"bidIp"`
	BidIpv6             string `json:"bidIpv6"`
	BidUa               string `json:"bidUa"`
	Bidder              string `json:"bidder"`
	BidPageUrl          string `json:"bidPageUrl"`
	BidderRequestId     string `json:"bidderRequestId"`
	BidRequestsCount    int    `json:"bidRequestsCount"`
	BidderRequestsCount int    `json:"bidderRequestsCount"`
	Sizes               string `json:"sizes"`
}

type ResponseAdQuery struct {
	Data *ResponseData `json:"data"`
}

type ResponseData struct {
	ReqID     string           `json:"requestId"`
	CrID      int64            `json:"creationId"`
	Currency  string           `json:"currency"`
	CPM       string           `json:"cpm"`
	Code      string           `json:"code"`
	AdQLib    string           `json:"adqLib"`
	Tag       string           `json:"tag"`
	ADomains  []string         `json:"adDomains"`
	DealID    string           `json:"dealid"`
	MediaType AdQueryMediaType `json:"mediaType"`
}

type AdQueryMediaType struct {
	Name   openrtb_ext.BidType `json:"name"`
	Width  string              `json:"width"`
	Height string              `json:"height"`
}
