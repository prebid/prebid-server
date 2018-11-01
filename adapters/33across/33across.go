package ttx

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type TtxAdapter struct {
	endpoint string
}

// MakeRequests create the object for TTX Reqeust.
func (a *TtxAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	var errs []error
	var adapterRequests []*adapters.RequestData

	adapterReq, errors := a.makeRequest(request)
	if adapterReq != nil {
		adapterRequests = append(adapterRequests, adapterReq)
	}
	errs = append(errs, errors...)

	return adapterRequests, errors
}

// Update the request object to include custom value
// site.id
func (a *TtxAdapter) makeRequest(request *openrtb.BidRequest) (*adapters.RequestData, []error) {
	var errs []error
	var validImps []openrtb.Imp

	for _, imp := range request.Imp {
		validImps = append(validImps, imp)
	}

	// If all the imps were malformed, don't bother making a server call with no impressions.
	if len(validImps) == 0 {
		return nil, errs
	}

	request.Imp = validImps

	// Make a copy as we don't want to change the original request
	reqCopy := *request
	if err := preprocess(&reqCopy); err != nil {
		errs = append(errs, err)
	}

	// Last Step
	reqJSON, err := json.Marshal(reqCopy)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
	}, errs
}

// Mutate the request to get it ready to send to ttx.
func preprocess(request *openrtb.BidRequest) error {
	var imp = &request.Imp[0]
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		fmt.Println("first error:", err)
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var ttxExt openrtb_ext.ExtImpTtx
	if err := json.Unmarshal(bidderExt.Bidder, &ttxExt); err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	impExt := map[string]map[string]string{
		"ttx": {
			"prod": ttxExt.ProductId,
		},
	}
	// Add zoneid if it's defined
	if len(ttxExt.ZoneId) != 0 {
		impExt["ttx"]["zoneid"] = ttxExt.ZoneId
	}

	impExtJSON, err := json.Marshal(impExt)
	if err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	imp.Ext = impExtJSON
	request.Site.ID = ttxExt.SiteId

	return nil
}

// MakeBids make the bids for the bid reponse.
func (a *TtxAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: "banner",
			})
		}
	}
	return bidResponse, nil

}

func NewTtxBidder(endpoint string) *TtxAdapter {
	return &TtxAdapter{
		endpoint: endpoint,
	}
}
