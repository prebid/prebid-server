package escalax

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
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
	escalaxExt, err := getImpressionExt(&openRTBRequest.Imp[0])
	if err != nil {
		return nil, []error{err}
	}

	openRTBRequest.Imp[0].Ext = nil

	url, err := a.buildEndpointURL(escalaxExt)
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
		ImpIDs:  openrtb_ext.GetImpIDs(openRTBRequest.Imp),
	}}, nil
}

func getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ExtEscalax, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Error parsing escalaxExt - " + err.Error(),
		}
	}
	var escalaxExt openrtb_ext.ExtEscalax
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &escalaxExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Error parsing bidderExt - " + err.Error(),
		}
	}

	return &escalaxExt, nil
}

func (a *adapter) buildEndpointURL(params *openrtb_ext.ExtEscalax) (string, error) {
	endpointParams := macros.EndpointTemplateParams{AccountID: params.AccountID, SourceId: params.SourceID}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func (a *adapter) MakeBids(openRTBRequest *openrtb2.BidRequest, requestToBidder *adapters.RequestData, bidderRawResponse *adapters.ResponseData) (bidderResponse *adapters.BidderResponse, errs []error) {
	if adapters.IsResponseStatusCodeNoContent(bidderRawResponse) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(bidderRawResponse); err != nil {
		return nil, []error{err}
	}

	responseBody := bidderRawResponse.Body
	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseBody, &bidResp); err != nil {
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
			bidType, err := determineImpressionMediaType(bid)
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

func determineImpressionMediaType(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("unsupported MType %d", bid.MType),
		}
	}
}
