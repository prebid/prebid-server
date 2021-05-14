package revcontent

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

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Revcontent adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	reqBody, err := json.Marshal(request)

	if err != nil {
		return nil, []error{err}
	}

	if err := checkRequest(request); err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	req := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqBody,
		Headers: headers,
	}
	return []*adapters.RequestData{req}, nil
}

func checkRequest(request *openrtb2.BidRequest) error {
	if (request.App == nil || len(request.App.Name) == 0) && (request.Site == nil || len(request.Site.Domain) == 0) {
		return &errortypes.BadInput{
			Message: "Impression is missing app name or site domain, and must contain one.",
		}
	}

	return nil
}

// MakeBids make the bids for the bid response.
func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("unexpected status code: %d.", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("unexpected status code: %d.", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			var mediaType = getBidType(sb.Bid[i].AdM)
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: mediaType,
			})
		}
	}
	return bidResponse, nil

}

func getBidType(bidAdm string) openrtb_ext.BidType {
	// native: {"ver":"1.1","assets":...
	// banner: <div id='rtb-widget...
	if bidAdm != "" && bidAdm[:1] == "<" {
		return openrtb_ext.BidTypeBanner
	}
	return openrtb_ext.BidTypeNative
}
