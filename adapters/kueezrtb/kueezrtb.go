package kueezrtb

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"net/http"
	"net/url"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the kueezrtb for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errors []error

	requestCopy := *request

	for _, imp := range request.Imp {
		requestCopy.Imp = []openrtb2.Imp{imp}

		requestJSON, err := json.Marshal(&requestCopy)
		if err != nil {
			errors = append(errors, fmt.Errorf("marshal bidRequest: %w", err))
			continue
		}

		cId, err := extractCid(&imp)
		if err != nil {
			errors = append(errors, fmt.Errorf("extract cId: %w", err))
			continue
		}

		headers := http.Header{}
		headers.Add("Content-Type", "application/json;charset=utf-8")

		requestData := &adapters.RequestData{
			Method:  "POST",
			Uri:     fmt.Sprintf("%s%s", a.endpoint, url.QueryEscape(cId)),
			Body:    requestJSON,
			Headers: headers,
			ImpIDs:  []string{imp.ID},
		}

		requests = append(requests, requestData)
	}

	return requests, errors
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", responseData.StatusCode),
		}}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("bad server response: %d. ", err),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(response.SeatBid))

	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}

	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return bidResponse, errs
}

func extractCid(imp *openrtb2.Imp) (string, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", fmt.Errorf("unmarshal bidderExt: %w", err)
	}

	var impExt openrtb_ext.ImpExtKueez
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
		return "", fmt.Errorf("unmarshal ImpExtkueez: %w", err)
	}
	return impExt.ConnectionId, nil
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	}
	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Could not define bid type for imp: %s", bid.ImpID),
	}
}
