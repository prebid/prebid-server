// Package pixfuture implements the Pixfuture adapter for Prebid Server.
// It provides functionality to handle bid requests and responses according to Pixfuture's specifications
package pixfuture

import (
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/iterutil"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{
		endpoint: config.Endpoint,
	}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	var errs []error
	var adapterRequests []*adapters.RequestData
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("Accept", "application/json")

	for imp := range iterutil.SlicePointerValues(request.Imp) {

		var bidderExt adapters.ExtImpBidder
		var pixfutureExt openrtb_ext.ImpExtPixfuture

		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &pixfutureExt); err != nil {
			errs = append(errs, err)
			continue
		}
		if pixfutureExt.PixID == "" {
			errs = append(errs, &errortypes.BadInput{Message: "Missing required parameter pix_id"})
			continue
		}

		requestCopy := *request
		requestCopy.Imp = []openrtb2.Imp{*imp} // slice notation with dereferencing

		reqJSON, err := jsonutil.Marshal(requestCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		adapterRequests = append(adapterRequests, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     a.endpoint,
			Body:    reqJSON,
			Headers: headers,
			ImpIDs:  []string{imp.ID},
		})
	}

	if len(adapterRequests) == 0 && len(errs) > 0 {
		return nil, errs
	}
	return adapterRequests, errs
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Invalid response format: " + err.Error(),
		}}
	}

	// Pre-calculate total number of bids to avoid slice reallocations
	expectedBids := 0
	for i := range bidResp.SeatBid {
		expectedBids += len(bidResp.SeatBid[i].Bid)
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(expectedBids)
	bidResponse.Currency = bidResp.Cur

	var errs []error
	for i := range bidResp.SeatBid {
		for _, bid := range bidResp.SeatBid[i].Bid {
			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errs = append(errs, &errortypes.BadServerResponse{
					Message: "Failed to parse impression \"" + bid.ImpID + "\" mediatype",
				})
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}

	if len(bidResponse.Bids) == 0 {
		if len(errs) > 0 {
			return nil, errs
		}
		return nil, nil
	}

	return bidResponse, errs
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	// First try standard MType field (OpenRTB 2.6)
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	}

	// Fallback to custom extension
	var ext struct {
		Prebid struct {
			Type string `json:"type"`
		} `json:"prebid"`
	}
	if err := jsonutil.Unmarshal(bid.Ext, &ext); err != nil {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Failed to parse bid ext for impression %s: %v", bid.ImpID, err),
		}
	}

	switch ext.Prebid.Type {
	case "banner":
		return openrtb_ext.BidTypeBanner, nil
	case "video":
		return openrtb_ext.BidTypeVideo, nil
	case "native":
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unknown bid type for impression %s", bid.ImpID),
		}
	}
}
