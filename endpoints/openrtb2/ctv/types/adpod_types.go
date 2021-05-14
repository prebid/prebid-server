package types

import (
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/endpoints/openrtb2/ctv/constant"
	"github.com/prebid/prebid-server/openrtb_ext"
)

//Bid openrtb bid object with extra parameters
type Bid struct {
	*openrtb2.Bid
	Duration          int
	FilterReasonCode  constant.FilterReasonCode
	DealTierSatisfied bool
}

//ExtCTVBidResponse object for ctv bid resposne object
type ExtCTVBidResponse struct {
	openrtb_ext.ExtBidResponse
	AdPod *BidResponseAdPodExt `json:"adpod,omitempty"`
}

//BidResponseAdPodExt object for ctv bidresponse adpod object
type BidResponseAdPodExt struct {
	Response openrtb2.BidResponse `json:"bidresponse,omitempty"`
	Config   map[string]*ImpData  `json:"config,omitempty"`
}

//AdPodBid combination contains ImpBid
type AdPodBid struct {
	Bids          []*Bid
	Price         float64
	Cat           []string
	ADomain       []string
	OriginalImpID string
	SeatName      string
}

//AdPodBids combination contains ImpBid
type AdPodBids []*AdPodBid

//BidsBuckets bids bucket
type BidsBuckets map[int][]*Bid

//ImpAdPodConfig configuration for creating ads in adpod
type ImpAdPodConfig struct {
	ImpID          string `json:"id,omitempty"`
	SequenceNumber int8   `json:"seq,omitempty"`
	MinDuration    int64  `json:"minduration,omitempty"`
	MaxDuration    int64  `json:"maxduration,omitempty"`
}

//ImpData example
type ImpData struct {
	//AdPodGenerator
	VideoExt  *openrtb_ext.ExtVideoAdPod `json:"vidext,omitempty"`
	Config    []*ImpAdPodConfig          `json:"imp,omitempty"`
	ErrorCode *int                       `json:"ec,omitempty"`
	Bid       *AdPodBid                  `json:"-"`
}
