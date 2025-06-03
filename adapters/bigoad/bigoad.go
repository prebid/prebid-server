package bigoad

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

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	bigoadExt, err := getImpExt(&request.Imp[0])
	if err != nil {
		return nil, []error{err}
	}

	resolvedUrl, err := a.buildEndpointURL(bigoadExt)
	if err != nil {
		return nil, []error{err}
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     resolvedUrl,
		Body:    requestJSON,
		Headers: getHeaders(request),
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, nil
}

func getHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}
	addNonEmptyHeaders(&headers, map[string]string{
		"Content-Type":      "application/json;charset=utf-8",
		"Accept":            "application/json",
		"x-openrtb-version": "2.5",
	})
	return headers
}

func addNonEmptyHeaders(headers *http.Header, headerValues map[string]string) {
	for key, value := range headerValues {
		if len(value) > 0 {
			headers.Add(key, value)
		}
	}
}

func getImpExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpBigoAd, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: unable to unmarshal ext", imp.ID),
		}
	}
	var bigoadExt openrtb_ext.ExtImpBigoAd
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &bigoadExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: unable to unmarshal ext.bidder: %v", imp.ID, err),
		}
	}
	imp.Ext = bidderExt.Bidder
	return &bigoadExt, nil
}

func (a *adapter) buildEndpointURL(params *openrtb_ext.ExtImpBigoAd) (string, error) {
	endpointParams := macros.EndpointTemplateParams{SspId: params.SspId}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var bidResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &bidResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad server response: %d", err),
		}}
	}

	if len(bidResponse.SeatBid) == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Empty SeatBid array",
		}}
	}

	bidResponseWithCapacity := adapters.NewBidderResponseWithBidsCapacity(len(bidResponse.SeatBid[0].Bid))

	var errors []error
	seatBid := bidResponse.SeatBid[0]
	for i := range seatBid.Bid {
		bid := seatBid.Bid[i]
		bidType, err := getBidType(request.Imp[0], bid)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		bidResponseWithCapacity.Bids = append(bidResponseWithCapacity.Bids, &adapters.TypedBid{
			Bid:     &bid,
			BidType: bidType,
		})
	}

	return bidResponseWithCapacity, errors
}

func getBidType(imp openrtb2.Imp, bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	}

	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("unrecognized bid type in response from bigoad %s", imp.ID),
	}
}
