package taboola

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Foo adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	taboolaRequest, errs := createTaboolaRequest(request)
	if len(errs) > 0 {
		return nil, errs
	}

	requestJSON, err := json.Marshal(taboolaRequest)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    a.endpoint + "/" + taboolaRequest.Site.ID,
		Body:   requestJSON,
	}

	return []*adapters.RequestData{requestData}, errs
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

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
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaType(seatBid.Bid[i].ImpID, request.Imp)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, errs
}

func createTaboolaRequest(request *openrtb2.BidRequest) (taboolaRequest *openrtb2.BidRequest, errors []error) {
	modifiedRequest := *request
	var errs []error

	var taboolaExt openrtb_ext.ImpExtTaboola
	for i := 0; i < len(modifiedRequest.Imp); i++ {
		imp := modifiedRequest.Imp[i]

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := json.Unmarshal(bidderExt.Bidder, &taboolaExt); err != nil {
			errs = append(errs, err)
			continue
		}
		if taboolaExt.TagId != "" {
			imp.TagID = taboolaExt.TagId
			modifiedRequest.Imp[i] = imp
		}
		if taboolaExt.BidFloor != 0 {
			imp.BidFloor = taboolaExt.BidFloor
			modifiedRequest.Imp[i] = imp
		}
	}

	publisher := &openrtb2.Publisher{
		ID: taboolaExt.PublisherId,
	}

	if modifiedRequest.Site == nil {
		newSite := &openrtb2.Site{
			ID:        taboolaExt.PublisherId,
			Name:      taboolaExt.PublisherId,
			Domain:    evaluateDomain(taboolaExt.PublisherDomain, request),
			Publisher: publisher,
		}
		modifiedRequest.Site = newSite
	} else {
		modifiedSite := *modifiedRequest.Site
		modifiedSite.Publisher = publisher
		modifiedSite.ID = taboolaExt.PublisherId
		modifiedSite.Name = taboolaExt.PublisherId
		modifiedSite.Domain = evaluateDomain(taboolaExt.PublisherDomain, request)
		modifiedRequest.Site = &modifiedSite
	}

	if taboolaExt.BCat != nil {
		modifiedRequest.BCat = taboolaExt.BCat
	}

	if taboolaExt.BAdv != nil {
		modifiedRequest.BAdv = taboolaExt.BAdv
	}

	return &modifiedRequest, errs
}

func getMediaType(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			}
		}
	}

	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find banner impression \"%s\" ", impID),
	}
}

func evaluateDomain(publisherDomain string, request *openrtb2.BidRequest) (result string) {
	if publisherDomain != "" {
		return publisherDomain
	}
	if request.Site != nil {
		return request.Site.Domain
	}
	return ""
}
