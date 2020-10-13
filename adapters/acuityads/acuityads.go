package acuityads

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type AcuityAdsAdapter struct {
	endpoint template.Template
}

func NewAcuityAdsBidder(endpointTemplate string) *AcuityAdsAdapter {
	template, err := template.New("endpointTemplate").Parse(endpointTemplate)
	if err != nil {
		return nil
	}
	return &AcuityAdsAdapter{endpoint: *template}
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

		if len(request.Device.IPv6) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}

		if len(request.Device.IP) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IP)
		}
	}

	return &headers
}

func (a *AcuityAdsAdapter) MakeRequests(
	openRTBRequest *openrtb.BidRequest,
	reqInfo *adapters.ExtraRequestInfo,
) (
	requestsToBidder []*adapters.RequestData,
	errs []error,
) {

	var errors []error
	var acuityAdsExt *openrtb_ext.ExtAcuityAds
	var err error

	for i, imp := range openRTBRequest.Imp {
		acuityAdsExt, err = a.getImpressionExt(&imp)
		if err != nil {
			errors = append(errors, err)
			break
		}
		openRTBRequest.Imp[i].Ext = nil
	}

	if len(errors) > 0 {
		return nil, errors
	}

	url, err := a.buildEndpointURL(acuityAdsExt)
	if err != nil {
		return nil, []error{err}
	}

	reqJSON, err := json.Marshal(openRTBRequest)
	if err != nil {
		return nil, []error{err}
	}

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Body:    reqJSON,
		Uri:     url,
		Headers: *GetHeaders(openRTBRequest),
	}}, nil
}

func (a *AcuityAdsAdapter) getImpressionExt(imp *openrtb.Imp) (*openrtb_ext.ExtAcuityAds, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
	}
	var acuityAdsExt openrtb_ext.ExtAcuityAds
	if err := json.Unmarshal(bidderExt.Bidder, &acuityAdsExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
	}
	return &acuityAdsExt, nil
}

func (a *AcuityAdsAdapter) buildEndpointURL(params *openrtb_ext.ExtAcuityAds) (string, error) {
	endpointParams := macros.EndpointTemplateParams{Host: params.Host, AccountID: params.AccountID}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func (a *AcuityAdsAdapter) CheckResponseStatusCodes(response *adapters.ResponseData) error {
	if response.StatusCode == http.StatusNoContent {
		return &errortypes.BadInput{Message: "No bid response"}
	}

	if response.StatusCode == http.StatusBadRequest {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: [ %d ]", response.StatusCode),
		}
	}

	if response.StatusCode == http.StatusServiceUnavailable {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Something went wrong, please contact your Account Manager. Status Code: [ %d ] ", response.StatusCode),
		}
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Something went wrong, please contact your Account Manager. Status Code: [ %d ] ", response.StatusCode),
		}
	}

	return nil
}

func (a *AcuityAdsAdapter) MakeBids(
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

	if len(bidResp.SeatBid) == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Empty SeatBid array",
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
