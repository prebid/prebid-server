package minutemedia

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// adapter is a MinuteMedia implementation of the adapters.Bidder interface.
type adapter struct {
	endpointURL string
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	return &adapter{
		endpointURL: config.Endpoint,
	}, nil
}

// MakeRequests prepares the HTTP requests which should be made to fetch bids.
func (a *adapter) MakeRequests(openRTBRequest *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) (requestsToBidder []*adapters.RequestData, errs []error) {
	org, err := extractOrg(openRTBRequest)
	if err != nil {
		return nil, append(errs, fmt.Errorf("failed to extract org: %w", err))
	}

	openRTBRequestJSON, err := json.Marshal(openRTBRequest)
	if err != nil {
		return nil, append(errs, err)
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return append(requestsToBidder, &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.endpointURL + "?publisher_id=" + url.QueryEscape(org),
		Body:    openRTBRequestJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(openRTBRequest.Imp),
	}), nil
}

// MakeBids unpacks the server's response into Bids.
func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur

	var errs []error

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

func extractOrg(openRTBRequest *openrtb2.BidRequest) (string, error) {
	var err error
	if len(openRTBRequest.Imp) == 0 {
		return "", errors.New("no imps in bid request")
	}

	var bidderExt adapters.ExtImpBidder
	if err = jsonutil.Unmarshal(openRTBRequest.Imp[0].Ext, &bidderExt); err != nil {
		return "", fmt.Errorf("failed to unmarshal bidderExt: %w", err)
	}

	var impExt openrtb_ext.ImpExtMinuteMedia
	if err = jsonutil.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
		return "", fmt.Errorf("failed to unmarshal ImpExtMinuteMedia: %w", err)
	}

	return strings.TrimSpace(impExt.Org), nil
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported MType %d", bid.MType),
		}
	}
}
