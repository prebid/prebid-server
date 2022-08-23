package seedingAlliance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint string
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, extraRequestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	for i := range request.Imp {
		if err := addTagID(&request.Imp[i]); err != nil {
			return nil, []error{err}
		}
	}

	if !curExists(request.Cur, "EUR") {
		request.Cur = append(request.Cur, "EUR")
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method: http.MethodPost,
		Uri:    a.endpoint,
		Body:   requestJSON,
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
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			resolvePriceMacro(&seatBid.Bid[i])

			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: getMediaTypeForImp(seatBid.Bid[i].ImpID, request.Imp),
			}

			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}

	return bidResponse, nil
}

func resolvePriceMacro(bid *openrtb2.Bid) {
	price := strconv.FormatFloat(bid.Price, 'f', -1, 64)
	bid.AdM = strings.Replace(bid.AdM, "${AUCTION_PRICE}", price, -1)
}

func getMediaTypeForImp(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
	var UnknownBidType openrtb_ext.BidType = "unknown"

	for _, imp := range imps {
		if imp.ID == impId {
			switch {
			case imp.Native != nil:
				return openrtb_ext.BidTypeNative
			case imp.Banner != nil:
				return openrtb_ext.BidTypeBanner
			}
		}
	}

	return UnknownBidType
}

func curExists(cc []string, c string) bool {
	for i := range cc {
		if cc[i] == c {
			return true
		}
	}
	return false
}

func addTagID(imp *openrtb2.Imp) error {
	var ext adapters.ExtImpBidder
	var extSA openrtb_ext.ImpExtSeedingAlliance

	if err := json.Unmarshal(imp.Ext, &ext); err != nil {
		return fmt.Errorf("could not unmarshal adapters.ExtImpBidder: %w", err)
	}

	if err := json.Unmarshal(ext.Bidder, &extSA); err != nil {
		return fmt.Errorf("could not unmarshal openrtb_ext.ImpExtSeedingAlliance: %w", err)
	}

	imp.TagID = extSA.AdUnitID

	return nil
}
