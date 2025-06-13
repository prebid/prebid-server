package rediads

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/buger/jsonparser"
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

// Builder builds a new instance of the adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error
	var accountID string
	var endpoint string

	// Iterate through all impressions in the request
	for i, imp := range request.Imp {
		// Extract and validate bidder-specific params from imp.Ext
		var bidderExt adapters.ExtImpBidder
		var rediadsExt openrtb_ext.ExtImpRediads

		// Unmarshal bidder extension
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("Invalid Ext format in impression %s", imp.ID),
			})
			continue
		}
		// Remove "prebid" and "bidder" from imp.Ext
		newExt := jsonparser.Delete(imp.Ext, "prebid")
		newExt = jsonparser.Delete(newExt, "bidder")
		imp.Ext = newExt

		// Unmarshal custom bidder params
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &rediadsExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("Invalid bidder params in impression %s", imp.ID),
			})
			continue
		}

		// Set accountID and slot to use in publisher id and imp tag id respectively
		accountID = rediadsExt.AccountID
		endpoint = rediadsExt.Endpoint

		// Set tagid in the imp object
		if rediadsExt.Slot != "" {
			imp.TagID = rediadsExt.Slot
		}
		request.Imp[i] = imp
	}

	// Update site or app publisher ID with accountID
	if request.Site != nil {
		siteCopy := *request.Site
		if siteCopy.Publisher != nil {
			publisherCopy := *siteCopy.Publisher
			publisherCopy.ID = accountID
			siteCopy.Publisher = &publisherCopy
		} else {
			siteCopy.Publisher = &openrtb2.Publisher{ID: accountID}
		}
		request.Site = &siteCopy
	} else if request.App != nil {
		appCopy := *request.App
		if appCopy.Publisher != nil {
			publisherCopy := *appCopy.Publisher
			publisherCopy.ID = accountID
			appCopy.Publisher = &publisherCopy
		} else {
			appCopy.Publisher = &openrtb2.Publisher{ID: accountID}
		}
		request.App = &appCopy
	}

	// Marshal the final request
	requestJSON, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	finalEndpoint := a.endpoint
	if endpoint != "" {
		parsedURL, _ := url.Parse(finalEndpoint)
		host := parsedURL.Host
		parts := strings.Split(host, ".")
		parts[0] = endpoint
		newHost := strings.Join(parts, ".")
		finalEndpoint = strings.Replace(finalEndpoint, host, newHost, 1)
	}

	// Build adapters.RequestData
	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    finalEndpoint,
		Body:   requestJSON,
		ImpIDs: openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, errors
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("could not define media type for impression: %s", bid.ImpID)
	}
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
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				return nil, []error{err}
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}
	return bidResponse, nil
}
