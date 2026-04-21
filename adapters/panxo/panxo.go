package panxo

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
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

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "no impressions in request",
		}}
	}

	// Extract propertyKey from first impression's bidder params
	var bidderExt openrtb_ext.ExtImpPanxo
	var extBidder adapters.ExtImpBidder

	if err := jsonutil.Unmarshal(request.Imp[0].Ext, &extBidder); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("invalid imp.ext for impression index 0: %s", err.Error()),
		}}
	}

	if err := jsonutil.Unmarshal(extBidder.Bidder, &bidderExt); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("invalid imp.ext.bidder for impression index 0: %s", err.Error()),
		}}
	}

	if bidderExt.PropertyKey == "" {
		return nil, []error{&errortypes.BadInput{
			Message: "propertyKey is required",
		}}
	}

	// Validate all impressions share the same propertyKey
	for i := 1; i < len(request.Imp); i++ {
		var impExtBidder adapters.ExtImpBidder
		var impBidderExt openrtb_ext.ExtImpPanxo
		if err := jsonutil.Unmarshal(request.Imp[i].Ext, &impExtBidder); err != nil {
			continue
		}
		if err := jsonutil.Unmarshal(impExtBidder.Bidder, &impBidderExt); err != nil {
			continue
		}
		if impBidderExt.PropertyKey != bidderExt.PropertyKey {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("all impressions must share the same propertyKey, imp[%d] has different value", i),
			}}
		}
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, append(errs, err)
	}

	url := fmt.Sprintf("%s?key=%s&source=prebid-server", a.endpoint, bidderExt.PropertyKey)

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Uri:     url,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, errs
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("unexpected status code: %d", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("unexpected status code: %d", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid))

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: openrtb_ext.BidTypeBanner,
			})
		}
	}

	return bidResponse, nil
}
