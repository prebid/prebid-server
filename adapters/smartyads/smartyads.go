package smartyads

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Implements Bidder interface.
type SmartyAdsAdapter struct {
	endpoint string
}

func NewSmartyAdsBidder(endpoint string) *SmartyAdsAdapter {
	return &SmartyAdsAdapter{endpoint: endpoint}
}

func (a *SmartyAdsAdapter) CheckHasImps(request *openrtb.BidRequest) error {
	if len(request.Imp) == 0 {
		err := &errortypes.BadInput{
			Message: "SmartyAds: Missing Imp object",
		}
		return err
	}
	return nil
}

func GetHeaders(request *openrtb.BidRequest) *http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.5")

	if request.Device != nil {
		if len(request.Device.UA) > 0 {
			headers.Add("User-Agent", request.Device.UA)
		}

		if len(request.Device.IP) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IP)
		}

		if len(request.Device.Language) > 0 {
			headers.Add("Accept-Language", request.Device.Language)
		}

		if request.Device.DNT != nil {
			headers.Add("Dnt", strconv.Itoa(int(*request.Device.DNT)))
		}
	}

	return &headers
}

func (a *SmartyAdsAdapter) MakeRequests(
	openRTBRequest *openrtb.BidRequest,
	reqInfo *adapters.ExtraRequestInfo,
) (
	requestsToBidder []*adapters.RequestData,
	errs []error,
) {

	request := *openRTBRequest

	if noImps := a.CheckHasImps(&request); noImps != nil {
		return nil, []error{noImps}
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Body:    reqJSON,
		Uri:     a.endpoint,
		Headers: *GetHeaders(&request),
	}}, nil
}

func (a *SmartyAdsAdapter) CheckResponseStatusCodes(response *adapters.ResponseData) error {
	if response.StatusCode == http.StatusNoContent {
		return &errortypes.BadInput{Message: " Invalid Bid Request received by the server"}
	}

	if response.StatusCode == http.StatusBadRequest {
		return &errortypes.BadInput{
			Message: fmt.Sprintf(" Unexpected status code: [ %d ] ", response.StatusCode),
		}
	}

	if response.StatusCode == http.StatusServiceUnavailable {
		return &errortypes.BadInput{
			Message: fmt.Sprintf(" Something went wrong, please contact your Account Manager. Status Code: [ %d ] ", response.StatusCode),
		}
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return &errortypes.BadInput{
			Message: fmt.Sprintf(" Something went wrong, please contact your Account Manager. Status Code: [ %d ] ", response.StatusCode),
		}
	}

	return nil
}

func (a *SmartyAdsAdapter) MakeBids(
	openRTBRequest *openrtb.BidRequest,
	requestToBidder *adapters.RequestData,
	bidderRawResponse *adapters.ResponseData,
) (
	bidderResponse *adapters.BidderResponse,
	errs []error,
) {
	httpStatusError := a.CheckResponseStatusCodes(bidderRawResponse)
	if httpStatusError != nil {
		return nil, []error{httpStatusError}
	}

	responseBody := bidderRawResponse.Body
	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(responseBody, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Bad Server Response",
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))
	sb := bidResp.SeatBid[0]

	for _, bid := range sb.Bid {
		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bid,
			BidType: getMediaTypeForImp(bid.ImpID, openRTBRequest.Imp),
		})
	}
	return bidResponse, nil
}

func getMediaTypeForImp(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			} else if imp.Native != nil {
				mediaType = openrtb_ext.BidTypeNative
			}
			return mediaType
		}
	}
	return mediaType
}
