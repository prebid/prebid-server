package smrtconnect

import (
	"encoding/json"
	"fmt"
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

// Builder builds a new instance of the Smrtconnect adapter for the given bidder with the given config.
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
	var requests []*adapters.RequestData

	requestCopy := *request
	for _, imp := range request.Imp {
		smrtconnectExt, err := getImpressionExt(&imp)
		if err != nil {
			return nil, []error{err}
		}

		url, err := a.buildEndpointURL(smrtconnectExt)
		if err != nil {
			return nil, []error{err}
		}

		requestCopy.Imp = []openrtb2.Imp{imp}
		requestJSON, err := json.Marshal(requestCopy)
		if err != nil {
			return nil, []error{err}
		}

		requestData := &adapters.RequestData{
			Method: "POST",
			Uri:    url,
			Body:   requestJSON,
			ImpIDs: openrtb_ext.GetImpIDs(requestCopy.Imp),
		}
		requests = append(requests, requestData)
	}
	return requests, nil
}

func getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ExtSmrtconnect, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
	}
	var smrtconnectExt openrtb_ext.ExtSmrtconnect
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &smrtconnectExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
	}
	imp.Ext = nil
	return &smrtconnectExt, nil
}

func (a *adapter) buildEndpointURL(params *openrtb_ext.ExtSmrtconnect) (string, error) {
	endpointParams := macros.EndpointTemplateParams{SupplyId: params.SupplyId}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Bad Server Response",
		}}
	}

	if len(response.SeatBid) == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Empty SeatBid array",
		}}
	}

	var bidErrs []error
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getBidType(seatBid.Bid[i])
			if err != nil {
				// could not determinate media type, append an error and continue with the next bid.
				bidErrs = append(bidErrs, err)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}
	return bidResponse, bidErrs
}

func getBidType(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	// determinate media type by bid response field mtype
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	}

	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Could not define media type for impression: %s", bid.ImpID),
	}
}
