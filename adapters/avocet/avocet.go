package avocet

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// AvocetAdapter implements a adapters.Bidder compatible with the Avocet advertising platform.
type AvocetAdapter struct {
	// Endpoint is a http endpoint to use when making requests to the Avocet advertising platform.
	Endpoint string
}

func (a *AvocetAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, nil
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	body, err := json.Marshal(request)
	if err != nil {
		return nil, []error{&errortypes.FailedToRequestBids{
			Message: err.Error(),
		}}
	}
	reqData := &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.Endpoint,
		Body:    body,
		Headers: headers,
	}
	return []*adapters.RequestData{reqData}, nil
}

type avocetBidExt struct {
	Avocet avocetBidExtension `json:"avocet"`
}

type avocetBidExtension struct {
	Duration     int `json:"duration"`
	DealPriority int `json:"deal_priority"`
}

func (a *AvocetAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		var errStr string
		if len(response.Body) > 0 {
			errStr = string(response.Body)
		} else {
			errStr = "no response body"
		}
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("received status code: %v error: %s", response.StatusCode, errStr),
		}}
	}

	var br openrtb2.BidResponse
	err := json.Unmarshal(response.Body, &br)
	if err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}
	var errs []error

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)
	for i := range br.SeatBid {
		for j := range br.SeatBid[i].Bid {
			var ext avocetBidExt
			if len(br.SeatBid[i].Bid[j].Ext) > 0 {
				err := json.Unmarshal(br.SeatBid[i].Bid[j].Ext, &ext)
				if err != nil {
					errs = append(errs, err)
					continue
				}
			}
			tbid := &adapters.TypedBid{
				Bid:          &br.SeatBid[i].Bid[j],
				DealPriority: ext.Avocet.DealPriority,
			}
			tbid.BidType = getBidType(br.SeatBid[i].Bid[j], ext)
			if tbid.BidType == openrtb_ext.BidTypeVideo {
				tbid.BidVideo = &openrtb_ext.ExtBidPrebidVideo{
					Duration: ext.Avocet.Duration,
				}
			}
			bidResponse.Bids = append(bidResponse.Bids, tbid)
		}
	}
	return bidResponse, nil
}

// getBidType returns the openrtb_ext.BidType for the provided bid.
func getBidType(bid openrtb2.Bid, ext avocetBidExt) openrtb_ext.BidType {
	if ext.Avocet.Duration != 0 {
		return openrtb_ext.BidTypeVideo
	}
	switch bid.API {
	case openrtb2.APIFrameworkVPAID10, openrtb2.APIFrameworkVPAID20:
		return openrtb_ext.BidTypeVideo
	default:
		return openrtb_ext.BidTypeBanner
	}
}

// Builder builds a new instance of the Avocet adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &AvocetAdapter{
		Endpoint: config.Endpoint,
	}
	return bidder, nil
}
