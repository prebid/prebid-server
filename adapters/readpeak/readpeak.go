package readpeak

import (
	"encoding/json"
	"fmt"
	"net/http"
  
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
  )
  
type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Readpeak adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
	  endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error

	requestCopy := *request
	requestCopy.id = requestCopy[0].bidderRequestId
	var rpExt openrtb_ext.ImpExtReadpeak
	for i := 0; i < len(requestCopy.Imp); i++ {		
		var impExt adapters.ExtImpBidder
		err := json.Unmarshal(jsonData, &impExt)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		err := json.Unmarshal(impExt.Bidder, &rpExt)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		imp := requestCopy.Imp[i]
		if rpExt.TagId != "" {
			imp.tagid = rpExt.TagId
		}
		if rpExt.Bidfloor != 0 {
			imp.bidfloor = rpExt.Bidfloor
		}
		requestCopy.Imp[i] = imp
	}

	if requestCopy.Site != nil {
		site := *requestCopy.Site
		if rpExt.SiteId != "" {
			site.Id = rpExt.SiteId
		}
		if rpExt.PublisherId != "" {
			site.publisher.id = rpExt.PublisherId
		}
		requestCopy.Site = site
	} else if requestCopy.App != nil {
		app := *requestCopy.App
		if rpExt.PublisherId != "" {
			app.publisher.id = rpExt.PublisherId
		}
		requestCopy.App = app
	}
	
	requestJSON, err := json.Marshal(requestCopy)
	if err != nil {
	  return nil, []error{err}
	}
  
	requestData := &adapters.RequestData{
	  Method:  "POST",
	  Uri:     a.endpoint,
	  Body:    requestJSON,
	}
  
	return []*adapters.RequestData{requestData}, errors
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher.",
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	var errors []error
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getMediaType((seatBid.Bid[i].ImpID, request.Imp))
			if err != nil {
				errors = append(errors, err)
				continue
			}
			resolveMacros(&seatBid.Bid[i])
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
				BidMeta: getBidMeta(bid),
			})
		}
	}
	return bidResponse, errors
}

func resolveMacros(bid *openrtb2.Bid) {
	if bid != nil {
		price := strconv.FormatFloat(bid.Price, 'f', -1, 64)
		bid.NURL = strings.Replace(bid.NURL, "${AUCTION_PRICE}", price, -1)
		bid.AdM = strings.Replace(bid.AdM, "${AUCTION_PRICE}", price, -1)
		bid.BURL = strings.Replace(bid.BURL, "${AUCTION_PRICE}", price, -1)
	}
}

func getMediaType(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			} else if imp.Native != nil {
				return openrtb_ext.BidTypeNative, nil
			}
		}
	}

	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find impression type \"%s\" ", bid.ImpID),
	}
}

func getBidMeta(bid *adapters.TypedBid) *openrtb_ext.ExtBidPrebidMeta {
	// This example includes all fields for demonstration purposes.
	return &openrtb_ext.ExtBidPrebidMeta {
		AdvertiserDomains:    []string{bid.Adomain}
	}
}
