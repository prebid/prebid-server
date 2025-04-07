package static

import (
	"encoding/json"
	"fmt"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type StaticAdapter struct{}

type staticExtBid struct {
	Bid openrtb2.Bid `json:"bid"`
}

// Builder is the registration entry point
func Builder(_ config.Adapter, _ config.Server) (adapters.Bidder, error) {
	return &StaticAdapter{}, nil
}

func (s *StaticAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	// No outbound HTTP requests for static bidder
	return nil, nil
}

func (s *StaticAdapter) MakeBids(request *openrtb2.BidRequest, req *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	return nil, []error{fmt.Errorf("MakeBids should not be called for static bidder")}
}

// Implement the OpenRTB bidder interface
func (s *StaticAdapter) MakeBidderResponse(request *openrtb2.BidRequest, extra *adapters.ExtraRequestInfo) (*adapters.BidderResponse, []error) {
	var errs []error
	bidResponse := &adapters.BidderResponse{
		Bids:     make([]*adapters.TypedBid, 0),
		Currency: "USD",
	}

	for _, imp := range request.Imp {
		var bidderExt staticExtBid
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, fmt.Errorf("failed to decode imp.ext.bidder: %v", err))
			continue
		}

		bid := bidderExt.Bid
		bid.ImpID = imp.ID

		typedBid := &adapters.TypedBid{
			Bid:     &bid,
			BidType: openrtb_ext.BidTypeBanner,
		}

		bidResponse.Bids = append(bidResponse.Bids, typedBid)
	}

	return bidResponse, errs
}
