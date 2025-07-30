package evolution

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
	URI string
}

type bidExt struct {
	MediaType openrtb_ext.BidType `json:"mediaType"`
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(
	openRTBRequest *openrtb2.BidRequest,
	reqInfo *adapters.ExtraRequestInfo,
) (
	requestsToBidder []*adapters.RequestData,
	errs []error,
) {

	reqJSON, err := json.Marshal(openRTBRequest)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Body:    reqJSON,
		Uri:     a.URI,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(openRTBRequest.Imp),
	}}, nil
}

func (a *adapter) MakeBids(
	openRTBRequest *openrtb2.BidRequest,
	requestToBidder *adapters.RequestData,
	bidderRawResponse *adapters.ResponseData,
) (
	bidderResponse *adapters.BidderResponse,
	errs []error,
) {
	if bidderRawResponse.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if bidderRawResponse.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Bad Request. %s", string(bidderRawResponse.Body)),
		}}
	}

	if bidderRawResponse.StatusCode == http.StatusServiceUnavailable {
		return nil, nil
	}

	if bidderRawResponse.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Something went wrong, please contact your Account Manager. Status Code: [ %d ] ", bidderRawResponse.StatusCode),
		}}
	}

	responseBody := bidderRawResponse.Body
	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseBody, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad response, %s", err),
		}}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Empty seatbid",
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))
	sb := bidResp.SeatBid[0]
	for i := range sb.Bid {
		var bidType openrtb_ext.BidType
		var bidExt bidExt
		if err := jsonutil.Unmarshal(sb.Bid[i].Ext, &bidExt); err != nil {
			bidType = openrtb_ext.BidTypeBanner
		} else {
			bidType = bidExt.MediaType
		}
		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &sb.Bid[i],
			BidType: bidType,
		})
	}
	return bidResponse, nil
}
