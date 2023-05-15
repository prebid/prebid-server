package xeworks

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
	"github.com/prebid/prebid-server/util/httputil"
)

// { prebid: { type: 'banner' } }

type bidType struct {
	Type string `json:"type"`
}

type bidExt struct {
	Prebid bidType `json:"prebid"`
}

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

func (a *adapter) buildEndpointFromRequest(imp *openrtb2.Imp) (string, error) {
	impExtRaw := imp.Ext
	var impExt adapters.ExtImpBidder

	if err := json.Unmarshal(impExtRaw, &impExt); err != nil {
		return "", &errortypes.BadInput{
			Message: "Bidder impression extension can't be deserialized",
		}
	}

	var xeworksExt openrtb_ext.ExtXeworks
	if err := json.Unmarshal(impExt.Bidder, &xeworksExt); err != nil {
		return "", &errortypes.BadInput{
			Message: "Xeworks extenson can't be deserialized",
		}
	}

	endpointParams := macros.EndpointTemplateParams{
		Host:     xeworksExt.Env,
		SourceId: xeworksExt.Pid,
	}

	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func (a *adapter) MakeRequests(openRTBRequest *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(openRTBRequest.Imp) == 0 {
		return nil, []error{
			&errortypes.BadInput{
				Message: "Imp array can't be empty",
			},
		}
	}

	requests := make([]*adapters.RequestData, 0, len(openRTBRequest.Imp))
	errs := make([]error, 0)

	body, err := json.Marshal(openRTBRequest)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	for _, imp := range openRTBRequest.Imp {
		endpoint, err := a.buildEndpointFromRequest(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		request := &adapters.RequestData{
			Method:  http.MethodPost,
			Body:    body,
			Uri:     endpoint,
			Headers: headers,
		}

		requests = append(requests, request)
	}

	return requests, errs
}

func (a *adapter) MakeBids(openRTBRequest *openrtb2.BidRequest, requestToBidder *adapters.RequestData, bidderRawResponse *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if httputil.IsResponseStatusCodeNoContent(bidderRawResponse) {
		return nil, nil
	}

	if bidderRawResponse.StatusCode == http.StatusServiceUnavailable {
		return nil, []error{&errortypes.BadInput{
			Message: "Bidder unavailable. Please contact the bidder support.",
		}}
	}

	if err := httputil.CheckResponseStatusCodeForErrors(bidderRawResponse); err != nil {
		return nil, []error{err}
	}

	responseBody := bidderRawResponse.Body
	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(responseBody, &bidResp); err != nil {
		return nil, []error{err}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Array SeatBid cannot be empty",
		}}
	}

	return prepareBidResponse(bidResp.SeatBid)
}

func prepareBidResponse(seats []openrtb2.SeatBid) (*adapters.BidderResponse, []error) {
	errs := make([]error, 0)
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(seats))

	for seatId, seatBid := range seats {
		if len(seatBid.Bid) == 0 {
			errs = append(errs, &errortypes.BadServerResponse{
				Message: fmt.Sprintf("Array SeatBid[%d].Bid cannot be empty", seatId),
			})
		}

		for bidId, bid := range seatBid.Bid {
			var bidExt bidExt
			if err := json.Unmarshal(bid.Ext, &bidExt); err != nil {
				errs = append(errs, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("Couldn't parse SeatBid[%d].Bid[%d].Ext, err: %s", seatId, bidId, err.Error()),
				})
				continue
			}

			bidType, err := openrtb_ext.ParseBidType(bidExt.Prebid.Type)

			if err != nil {
				errs = append(errs, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("SeatBid[%d].Bid[%d].Ext.Prebid.Type expects one of the following values: 'banner', 'native', 'video', 'audio', got '%s'", seatId, bidId, bidExt.Prebid.Type),
				})
			}

			// create copy if bid struct since without it bid address get's polluted with previous value
			// because of range
			bidCopy := bid
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bidCopy,
				BidType: bidType,
			})
		}
	}

	return bidResponse, errs
}
