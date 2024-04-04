package readpeak

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
  
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
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
	var rpExt openrtb_ext.ImpExtReadpeak
	var imps []openrtb2.Imp
	for i := 0; i < len(requestCopy.Imp); i++ {
		if requestCopy.Imp[i].Native == nil && requestCopy.Imp[i].Banner == nil {
			continue
		}
		var impExt adapters.ExtImpBidder
		err := json.Unmarshal(requestCopy.Imp[i].Ext, &impExt)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		err2 := json.Unmarshal(impExt.Bidder, &rpExt)
		if err2 != nil {
			errors = append(errors, err2)
			continue
		}
		imp := requestCopy.Imp[i]
		if rpExt.TagId != "" {
			imp.TagID = rpExt.TagId
		}
		if rpExt.Bidfloor != 0 {
			imp.BidFloor = rpExt.Bidfloor
		}
		imps = append(imps, imp)
	}

	if len(imps) == 0 {
		err := &errortypes.BadInput{
			Message: fmt.Sprintf("Failed to find compatible impressions for request %s", requestCopy.ID),
		}
		return nil, []error{err}
	}
	requestCopy.Imp = imps
	publisher := &openrtb2.Publisher{
		ID: rpExt.PublisherId,
	}

	if requestCopy.Site != nil {
		siteCopy := *request.Site
		if rpExt.SiteId != "" {
			siteCopy.ID = rpExt.SiteId
		}
		siteCopy.Publisher = publisher
		requestCopy.Site = &siteCopy
	} else if requestCopy.App != nil {
		appCopy := *request.App
		if rpExt.SiteId != "" {
			appCopy.ID = rpExt.SiteId
		}
		appCopy.Publisher = publisher
		requestCopy.App = &appCopy
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
		for i := range seatBid.Bid {
			bidType, err := getMediaType(seatBid.Bid[i].ImpID, request.Imp)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			resolveMacros(&seatBid.Bid[i])
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
				BidMeta: getBidMeta(&seatBid.Bid[i]),
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
		Message: fmt.Sprintf("Failed to find impression type \"%s\"", impID),
	}
}

func getBidMeta(bid *openrtb2.Bid) *openrtb_ext.ExtBidPrebidMeta {
	return &openrtb_ext.ExtBidPrebidMeta {
		AdvertiserDomains: bid.ADomain,
	}
}
