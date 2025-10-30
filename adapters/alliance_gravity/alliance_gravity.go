package alliance_gravity

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

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func generateImps(imps []openrtb2.Imp) ([]openrtb2.Imp, error) {
	var generatedImps []openrtb2.Imp
	for _, imp := range imps {
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return nil, &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		var allianceGravityExt openrtb_ext.ExtImpAllianceGravity
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &allianceGravityExt); err != nil {
			return nil, &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		bidderExt.Bidder = json.RawMessage(`{}`)
		bidderExt.Prebid = &openrtb_ext.ExtImpPrebid{
			StoredRequest: &openrtb_ext.ExtStoredRequest{
				ID: allianceGravityExt.SrID,
			},
		}

		bidderExtJSON, err := jsonutil.Marshal(bidderExt)
		if err != nil {
			return nil, &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		impCopy := imp
		impCopy.Ext = bidderExtJSON

		generatedImps = append(generatedImps, impCopy)
	}
	return generatedImps, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var generatedImps, err = generateImps(request.Imp)
	if err != nil {
		return nil, []error{err}
	}

	request.Imp = generatedImps

	requestJSON, err := jsonutil.Marshal(request)

	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	if request.Device != nil {
		if request.Device.UA != "" {
			headers.Add("User-Agent", request.Device.UA)
		}
		if request.Device.IP != "" {
			headers.Add("X-Forwarded-For", request.Device.IP)
		} else if request.Device.IPv6 != "" {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}
	}
	if request.Site != nil {
		if request.Site.Page != "" {
			headers.Add("Referer", request.Site.Page)
		}
	}

	if request.User != nil && request.User.BuyerUID != "" {
		headers.Add("Cookie", "uids="+request.User.BuyerUID)
	}

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    requestJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, nil
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	if bid.Ext != nil {
		var bidExt openrtb_ext.ExtBid
		err := jsonutil.Unmarshal(bid.Ext, &bidExt)
		if err == nil && bidExt.Prebid != nil {
			return openrtb_ext.ParseBidType(string(bidExt.Prebid.Type))
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to parse impression \"%s\" mediatype", bid.ImpID),
	}
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

	var bids []*adapters.TypedBid
	var errors []error
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			bids = append(bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	if len(bids) == 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bids))

	bidResponse.Currency = response.Cur
	bidResponse.Bids = bids

	return bidResponse, errors
}
