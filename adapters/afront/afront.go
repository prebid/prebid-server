package afront

import (
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

func Builder(
	bidderName openrtb_ext.BidderName,
	config config.Adapter,
	server config.Server,
) (
	adapters.Bidder,
	error,
) {
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
	headers.Add("X-Openrtb-Version", "2.5")
	headers.Add("Accept", "application/json")
	headers.Add("Content-Type", "application/json;charset=utf-8")

	if request.Device != nil {
		if len(request.Device.IP) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IP)
		}
		if len(request.Device.IPv6) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}
		if len(request.Device.UA) > 0 {
			headers.Add("User-Agent", request.Device.UA)
		}
	}

	return headers
}

func (a *adapter) MakeRequests(
	openRTBRequest *openrtb2.BidRequest,
	reqInfo *adapters.ExtraRequestInfo,
) (
	requestsToBidder []*adapters.RequestData,
	errs []error,
) {
	afrontExt, err := a.getImpressionExt(&openRTBRequest.Imp[0])
	if err != nil {
		return nil, []error{err}
	}

	url, err := a.buildEndpointURL(afrontExt)
	if err != nil {
		return nil, []error{err}
	}

	for idx := range openRTBRequest.Imp {
		openRTBRequest.Imp[idx].Ext = nil
	}

	reqJSON, err := jsonutil.Marshal(openRTBRequest)
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

func (a *adapter) getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ExtAfront, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
	}
	var afrontExt openrtb_ext.ExtAfront
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &afrontExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
	}

	return &afrontExt, nil
}

func (a *adapter) buildEndpointURL(params *openrtb_ext.ExtAfront) (string, error) {
	endpointParams := macros.EndpointTemplateParams{
		AccountID: params.AccountID,
		SourceId:  params.SourceId,
	}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func (a *adapter) MakeBids(
	openRTBRequest *openrtb2.BidRequest,
	requestToBidder *adapters.RequestData,
	bidderRawResponse *adapters.ResponseData,
) (
	bidderResponse *adapters.BidderResponse,
	errs []error,
) {
	if adapters.IsResponseStatusCodeNoContent(bidderRawResponse) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(bidderRawResponse); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(bidderRawResponse.Body, &bidResp); err != nil {
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
	var bidsArray []*adapters.TypedBid

	for idx, bid := range bidResp.SeatBid[0].Bid {
		bidsArray = append(bidsArray, &adapters.TypedBid{
			Bid:     &bidResp.SeatBid[0].Bid[idx],
			BidType: getMediaTypeForImp(bid.ImpID, openRTBRequest.Imp),
		})
	}

	bidResponse.Bids = bidsArray
	return bidResponse, nil
}

func getMediaTypeForImp(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
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
