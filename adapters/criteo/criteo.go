package criteo

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	uri             string
	slotIDGenerator slotIDGenerator
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, extraRequestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	criteoRequest, errs := newCriteoRequest(a.slotIDGenerator, request)
	if len(errs) > 0 {
		return nil, errs
	}

	jsonRequest, err := json.Marshal(criteoRequest)
	if err != nil {
		return nil, []error{err}
	}

	rqData := adapters.RequestData{
		Method:  "POST",
		Uri:     a.uri,
		Body:    jsonRequest,
		Headers: getCriteoRequestHeaders(&criteoRequest),
	}

	return []*adapters.RequestData{&rqData}, nil
}

func getCriteoRequestHeaders(criteoRequest *criteoRequest) http.Header {
	headers := http.Header{}

	// criteoRequest is known not to be nil
	// If there was an error generating it from newCriteoRequest, the errors will be returned immediately
	// and this method won't be called

	if criteoRequest.User.CookieID != "" {
		headers.Add("Cookie", "uid="+criteoRequest.User.CookieID)
	}

	if criteoRequest.User.IP != "" {
		headers.Add("X-Forwarded-For", criteoRequest.User.IP)
	}

	if criteoRequest.User.IPv6 != "" {
		headers.Add("X-Forwarded-For", criteoRequest.User.IPv6)
	}

	if criteoRequest.User.UA != "" {
		headers.Add("User-Agent", criteoRequest.User.UA)
	}

	return headers
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	bidResponse, err := newCriteoResponseFromBytes(response.Body)
	if err != nil {
		return nil, []error{err}
	}

	bidderResponse := adapters.NewBidderResponse()
	bidderResponse.Bids = make([]*adapters.TypedBid, len(bidResponse.Slots))

	for i := 0; i < len(bidResponse.Slots); i++ {
		bidderResponse.Bids[i] = &adapters.TypedBid{
			Bid: &openrtb2.Bid{
				ID:    bidResponse.Slots[i].ArbitrageID,
				ImpID: bidResponse.Slots[i].ImpID,
				Price: bidResponse.Slots[i].CPM,
				AdM:   bidResponse.Slots[i].Creative,
				W:     bidResponse.Slots[i].Width,
				H:     bidResponse.Slots[i].Height,
				CrID:  bidResponse.Slots[i].CreativeCode,
			},
			BidType: openrtb_ext.BidTypeBanner,
		}
	}

	return bidderResponse, nil
}

// Builder builds a new instance of the Criteo adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	return builderWithGuidGenerator(bidderName, config, newRandomSlotIDGenerator())
}

func builderWithGuidGenerator(bidderName openrtb_ext.BidderName, config config.Adapter, slotIDGenerator slotIDGenerator) (adapters.Bidder, error) {
	return &adapter{
		uri:             config.Endpoint,
		slotIDGenerator: slotIDGenerator,
	}, nil
}
