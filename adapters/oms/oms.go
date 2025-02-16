package oms

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the OMS adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	var uri string = a.endpoint
	if len(request.Imp[0].Ext) > 0 {
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(request.Imp[0].Ext, &bidderExt); err != nil {
			return nil, []error{err}
		}

		var omsImpExt openrtb_ext.ExtImpOms
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &omsImpExt); err != nil {
			return nil, []error{err}
		}

		uri = fmt.Sprintf("%s?publisherId=%s", a.endpoint, omsImpExt.Pid)
		if omsImpExt.Pid == "" && omsImpExt.PublisherID > 0 {
			uri = fmt.Sprintf("%s?publisherId=%d", a.endpoint, omsImpExt.PublisherID)
		}
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    uri,
		Body:   requestJSON,
		ImpIDs: openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if len(response.Cur) == 0 {
		bidResponse.Currency = response.Cur
	}

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType := getBidType(seatBid.Bid[i].MType)

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:      &seatBid.Bid[i],
				BidType:  bidType,
				BidVideo: getBidVideo(bidType, &seatBid.Bid[i]),
			})
		}
	}

	return bidResponse, nil
}

func getBidType(markupType openrtb2.MarkupType) openrtb_ext.BidType {
	switch markupType {
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo
	default:
		return openrtb_ext.BidTypeBanner
	}
}

func getBidVideo(bidType openrtb_ext.BidType, bid *openrtb2.Bid) *openrtb_ext.ExtBidPrebidVideo {
	if bidType != openrtb_ext.BidTypeVideo {
		return nil
	}

	var primaryCategory string
	if len(bid.Cat) > 0 {
		primaryCategory = bid.Cat[0]
	}

	return &openrtb_ext.ExtBidPrebidVideo{
		Duration:        int(bid.Dur),
		PrimaryCategory: primaryCategory,
	}
}
