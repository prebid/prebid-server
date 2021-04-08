package zemanta

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb/v14/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Zemanta adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	reqCopy := *request

	var errs []error
	var zemantaExt openrtb_ext.ExtImpZemanta
	for i := 0; i < len(reqCopy.Imp); i++ {
		imp := reqCopy.Imp[i]

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := json.Unmarshal(bidderExt.Bidder, &zemantaExt); err != nil {
			errs = append(errs, err)
			continue
		}
		imp.TagID = zemantaExt.TagId
		reqCopy.Imp[i] = imp
	}

	publisher := &openrtb2.Publisher{
		ID:     zemantaExt.Publisher.Id,
		Name:   zemantaExt.Publisher.Name,
		Domain: zemantaExt.Publisher.Domain,
	}
	if reqCopy.Site != nil {
		siteCopy := *reqCopy.Site
		siteCopy.Publisher = publisher
		reqCopy.Site = &siteCopy
	} else if reqCopy.App != nil {
		appCopy := *reqCopy.App
		appCopy.Publisher = publisher
		reqCopy.App = &appCopy
	}

	if zemantaExt.BCat != nil {
		reqCopy.BCat = zemantaExt.BCat
	}
	if zemantaExt.BAdv != nil {
		reqCopy.BAdv = zemantaExt.BAdv
	}

	requestJSON, err := json.Marshal(reqCopy)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    a.endpoint,
		Body:   requestJSON,
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur

	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bid := seatBid.Bid[i]
			bidType, err := getMediaTypeForImp(bid.ImpID, request.Imp)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			b := &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}

	return bidResponse, errs
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Native != nil {
				return openrtb_ext.BidTypeNative, nil
			} else if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			}
		}
	}

	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find native/banner impression \"%s\" ", impID),
	}
}
