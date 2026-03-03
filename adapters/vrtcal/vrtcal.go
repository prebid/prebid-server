package vrtcal

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

type VrtcalAdapter struct {
	endpoint string
}

func (a *VrtcalAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var adapterRequests []*adapters.RequestData

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	reqData := adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	adapterRequests = append(adapterRequests, &reqData)

	return adapterRequests, errs
}

// MakeBids make the bids for the bid response.
func (a *VrtcalAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse

	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	var errs []error
	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidType, err := getReturnTypeForImp(sb.Bid[i].MType)
			if err == nil {
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &sb.Bid[i],
					BidType: bidType,
				})
			} else {
				errs = append(errs, err)
			}
		}
	}
	return bidResponse, errs

}

// Builder builds a new instance of the Vrtcal adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &VrtcalAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func getReturnTypeForImp(mType openrtb2.MarkupType) (openrtb_ext.BidType, error) {
	if mType == openrtb2.MarkupBanner {
		return openrtb_ext.BidTypeBanner, nil
	} else if mType == openrtb2.MarkupVideo {
		return openrtb_ext.BidTypeVideo, nil
	} else if mType == openrtb2.MarkupNative {
		return openrtb_ext.BidTypeNative, nil
	} else {
		return "", &errortypes.BadServerResponse{
			Message: "Unsupported return type"}
	}
}
