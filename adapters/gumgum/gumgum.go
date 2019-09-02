package gumgum

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

type GumGumAdapter struct {
	URI string
}

func (g *GumGumAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var validImps []openrtb.Imp
	var trackingId string

	numRequests := len(request.Imp)
	errs := make([]error, 0, numRequests)

	for i := 0; i < numRequests; i++ {
		imp := request.Imp[i]
		zone, err := preprocess(&imp)
		if err != nil {
			errs = append(errs, err)
		} else {
			if request.Imp[i].Banner != nil {
				bannerCopy := *request.Imp[i].Banner
				if bannerCopy.W == nil && bannerCopy.H == nil && len(bannerCopy.Format) > 0 {
					format := bannerCopy.Format[0]
					bannerCopy.W = &(format.W)
					bannerCopy.H = &(format.H)
				}
				request.Imp[i].Banner = &bannerCopy
				validImps = append(validImps, request.Imp[i])
				trackingId = zone
			}
		}
	}

	if len(validImps) == 0 {
		return nil, errs
	}

	request.Imp = validImps

	if request.Site != nil {
		siteCopy := *request.Site
		siteCopy.ID = trackingId
		request.Site = &siteCopy
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     g.URI,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
}

func (g *GumGumAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Bad user input: HTTP status %d", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad server response: HTTP status %d", response.StatusCode),
		}}
	}
	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad server response: %d. ", err),
		}}
	}

	var errs []error
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: openrtb_ext.BidTypeBanner,
			})
		}
	}
	return bidResponse, errs
}

func preprocess(imp *openrtb.Imp) (string, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		err = &errortypes.BadInput{
			Message: err.Error(),
		}
		return "", err
	}

	var gumgumExt openrtb_ext.ExtImpGumGum
	if err := json.Unmarshal(bidderExt.Bidder, &gumgumExt); err != nil {
		err = &errortypes.BadInput{
			Message: err.Error(),
		}
		return "", err
	}

	zone := gumgumExt.Zone
	return zone, nil
}

func NewGumGumBidder(endpoint string) *GumGumAdapter {
	return &GumGumAdapter{
		URI: endpoint,
	}
}
