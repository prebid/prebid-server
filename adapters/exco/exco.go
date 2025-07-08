package exco

import (
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

// Builder initializes the EXCO adapter with the given configuration.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

// MakeRequests creates HTTP requests to the EXCO endpoint based on the OpenRTB bid request.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	adjustedReq, err := adjustRequest(request)
	if err != nil {
		return nil, append(errs, err)
	}

	payload, _ := jsonutil.Marshal(adjustedReq)

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	reqData := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    payload,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{reqData}, errs
}

// MakeBids processes the HTTP response from the EXCO endpoint and extracts bid information.
func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResponse); err != nil {
		return nil, []error{err}
	}

	var errs []error
	bidderResponse := adapters.NewBidderResponse()
	for _, seatBid := range bidResponse.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]

			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			})
		}
	}

	return bidderResponse, errs
}

// getMediaTypeForBid determines the media type of a bid based on its MType.
func getMediaTypeForBid(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", fmt.Errorf("unrecognized bid_ad_type in response from exco: %d", bid.MType)
	}
}

// adjustRequest modifies the OpenRTB bid request to include EXCO-specific parameters.
func adjustRequest(request *openrtb2.BidRequest) (*openrtb2.BidRequest, error) {
	var publisherId string

	// Extracts the publisher ID and tag ID from the impression extension.
	// Updates the impression's TagID with the extracted value.
	for i := 0; i < len(request.Imp); i++ {
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(request.Imp[i].Ext, &bidderExt); err != nil {
			// Handles invalid impression extension by returning a BadInput error.
			return nil, &errortypes.BadInput{
				Message: fmt.Sprintf("Invalid imp.ext for impression index %d. Error Information: %s", i, err.Error()),
			}
		}

		var impExt openrtb_ext.ImpExtExco
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
			// Handles invalid bidder extension by returning a BadInput error.
			return nil, &errortypes.BadInput{
				Message: fmt.Sprintf("Invalid imp.ext.bidder for impression index %d. Error Information: %s", i, err.Error()),
			}
		}

		publisherId = impExt.PublisherId
		request.Imp[i].TagID = impExt.TagId
	}

	// Creates a deep copy of the App object to avoid modifying the original request.
	if request.App != nil {
		appCopy := *request.App
		request.App = &appCopy

		if request.App.Publisher == nil {
			// Initializes the Publisher object if it is nil.
			request.App.Publisher = &openrtb2.Publisher{}
		} else {
			// Creates a deep copy of the Publisher object.
			publisherCopy := *request.App.Publisher
			request.App.Publisher = &publisherCopy
		}

		// Sets the Publisher ID to the extracted publisher ID.
		request.App.Publisher.ID = publisherId
	}

	// Creates a deep copy of the Site object to avoid modifying the original request.
	if request.Site != nil {
		siteCopy := *request.Site
		request.Site = &siteCopy

		if request.Site.Publisher == nil {
			// Initializes the Publisher object if it is nil.
			request.Site.Publisher = &openrtb2.Publisher{}
		} else {
			// Creates a deep copy of the Publisher object.
			publisherCopy := *request.Site.Publisher
			request.Site.Publisher = &publisherCopy
		}

		// Sets the Publisher ID to the extracted publisher ID.
		request.Site.Publisher.ID = publisherId
	}

	return request, nil
}
