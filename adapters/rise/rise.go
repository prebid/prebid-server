package rise

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// adapter is a Rise implementation of the adapters.Bidder interface.
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
		errs = append(errs, fmt.Errorf("extractOrg: %w", err))
		return nil, errs
	}

	openRTBRequestJSON, err := json.Marshal(openRTBRequest)
	if err != nil {
		errs = append(errs, fmt.Errorf("marshal bidRequest: %w", err))
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return append(requestsToBidder, &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.endpointURL + "?publisher_id=" + org,
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
	for _, imp := range openRTBRequest.Imp {
		var bidderExt adapters.ExtImpBidder
		if err = jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return "", fmt.Errorf("unmarshal bidderExt: %w", err)
		}

		var impExt openrtb_ext.ImpExtRise
		if err = jsonutil.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
			return "", fmt.Errorf("unmarshal ImpExtRise: %w", err)
		}

		if impExt.Org != "" {
			return strings.TrimSpace(impExt.Org), nil
		}
		if impExt.PublisherID != "" {
			return strings.TrimSpace(impExt.PublisherID), nil
		}
	}

	return "", errors.New("no org or publisher_id supplied")
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("unsupported MType %d", bid.MType)
	}
}
