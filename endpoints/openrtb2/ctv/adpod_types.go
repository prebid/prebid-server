package ctv

import (
	"github.com/PubMatic-OpenWrap/openrtb"
	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
)

type Bid struct {
	*openrtb.Bid
	Duration int
}

//AdPodBid combination contains ImpBid
type AdPodBid struct {
	Bids          []*Bid
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
