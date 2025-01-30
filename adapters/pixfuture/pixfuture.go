package pixfuture

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Pixfuture adapter.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{
		endpoint: config.Endpoint,
	}, nil
}

// MakeRequests prepares and serializes HTTP requests to be sent to the Pixfuture endpoint.
func (a *adapter) MakeRequests(bidRequest *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	// Validate the bid request
	if bidRequest == nil || len(bidRequest.Imp) == 0 {
		errs = append(errs, fmt.Errorf("no valid impressions in bid request"))
		return nil, errs
	}

	// Process impressions
	var validImpressions []openrtb2.Imp
	for i := range bidRequest.Imp {
		imp := &bidRequest.Imp[i]
		if imp.Banner == nil && imp.Video == nil {
			errs = append(errs, fmt.Errorf("unsupported impression type for impID: %s", imp.ID))
			continue
		}
		validImpressions = append(validImpressions, imp)
	}

	if len(validImpressions) == 0 {
		errs = append(errs, errors.New("no valid impressions after filtering"))
		return nil, errs
	}

	// Create the outgoing request
	bidRequest.Imp = validImpressions
	body, err := jsonutil.Marshal(bidRequest)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to marshal bid request: %w", err))
		return nil, errs
	}

	request := &adapters.RequestData{
		Method: "POST",
		Uri:    a.endpoint,
		Body:   body,
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
		},
	}

	return []*adapters.RequestData{request}, errs
}

// getMediaTypeForBid extracts the bid type based on the bid extension data.
func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	if bid.Ext != nil {
		var bidExt openrtb_ext.ExtBid
		err := jsonutil.Unmarshal(bid.Ext, &bidExt)
		if err == nil && bidExt.Prebid != nil {
			return openrtb_ext.ParseBidType(string(bidExt.Prebid.Type))
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to parse impression \"%s\" mediatype", bid.ImpID),
	}
}

// MakeBids parses the HTTP response from the Pixfuture endpoint and generates a BidderResponse.
func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	// Handle No Content response
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	// Check for errors in response status code
	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	// Parse the response body
	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	var errors []error

	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return bidResponse, errors
}
