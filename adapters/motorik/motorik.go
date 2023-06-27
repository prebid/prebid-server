package motorik

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint *template.Template
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpoint: template,
	}
	return bidder, nil
}

func getHeaders(request *openrtb2.BidRequest) http.Header {
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

	return headers
}

func (a *adapter) MakeRequests(openRTBRequest *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) (requestsToBidder []*adapters.RequestData, errs []error) {
	motorikExt, err := getImpressionExt(&openRTBRequest.Imp[0])
	if err != nil {
		return nil, []error{err}
	}

	openRTBRequest.Imp[0].Ext = nil

	url, err := a.buildEndpointURL(motorikExt)
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
		Headers: getHeaders(openRTBRequest),
	}}, nil
}

func getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ExtMotorik, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Error parsing motorikExt - " + err.Error(),
		}
	}
	var motorikExt openrtb_ext.ExtMotorik
	if err := json.Unmarshal(bidderExt.Bidder, &motorikExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Error parsing bidderExt - " + err.Error(),
		}
	}

	return &motorikExt, nil
}

func (a *adapter) buildEndpointURL(params *openrtb_ext.ExtMotorik) (string, error) {
	endpointParams := macros.EndpointTemplateParams{AccountID: params.AccountID, SourceId: params.PlacementID}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func checkResponseStatusCodes(response *adapters.ResponseData) error {
	if response.StatusCode == http.StatusServiceUnavailable {
		return &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Something went wrong Status Code: [ %d ] ", response.StatusCode),
		}
	}

	return adapters.CheckResponseStatusCodeForErrors(response)
}

func (a *adapter) MakeBids(openRTBRequest *openrtb2.BidRequest, requestToBidder *adapters.RequestData, bidderRawResponse *adapters.ResponseData) (bidderResponse *adapters.BidderResponse, errs []error) {
	if adapters.IsResponseStatusCodeNoContent(bidderRawResponse) {
		return nil, nil
	}

	httpStatusError := checkResponseStatusCodes(bidderRawResponse)
	if httpStatusError != nil {
		return nil, []error{httpStatusError}
	}

	responseBody := bidderRawResponse.Body
	var bidResp openrtb2.BidResponse
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

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)
	var bidsArray []*adapters.TypedBid

	for _, sb := range bidResp.SeatBid {
		for idx, bid := range sb.Bid {
			bidType, err := getMediaTypeForImp(bid.ImpID, openRTBRequest.Imp)
			if err != nil {
				return nil, []error{err}
			}

			bidsArray = append(bidsArray, &adapters.TypedBid{
				Bid:     &sb.Bid[idx],
				BidType: bidType,
			})
		}
	}

	bidResponse.Bids = bidsArray
	return bidResponse, nil
}

func getMediaTypeForImp(impId string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			} else if imp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			} else if imp.Native != nil {
				return openrtb_ext.BidTypeNative, nil
			}
			break
		}
	}
	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find impression \"%s\"", impId),
	}
}
