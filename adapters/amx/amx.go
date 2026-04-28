package amx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

const nbrHeaderName = "x-nbr"
const adapterVersion = "pbs1.3"
const bidderCurrency = "USD"

// AMXAdapter is the AMX bid adapter
type AMXAdapter struct {
	endpoint string
}

// Builder builds a new instance of the AMX adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpointURL, err := url.Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint: %v", err)
	}

	qs, err := url.ParseQuery(endpointURL.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("invalid query parameters in the endpoint: %v", err)
	}

	qs.Add("v", adapterVersion)
	endpointURL.RawQuery = qs.Encode()

	bidder := &AMXAdapter{
		endpoint: endpointURL.String(),
	}
	return bidder, nil
}

type amxExt struct {
	Bidder openrtb_ext.ExtImpAMX `json:"bidder"`
}

func ensurePublisherWithID(pub *openrtb2.Publisher, publisherID string) openrtb2.Publisher {
	if pub == nil {
		return openrtb2.Publisher{ID: publisherID}
	}

	pubCopy := *pub
	pubCopy.ID = publisherID
	return pubCopy
}

func resolveBidFloor(imp *openrtb2.Imp, reqInfo *adapters.ExtraRequestInfo) error {
	if reqInfo == nil || imp.BidFloor <= 0 || imp.BidFloorCur == "" || strings.EqualFold(imp.BidFloorCur, bidderCurrency) {
		return nil
	}

	convertedValue, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, bidderCurrency)
	if err != nil {
		return err
	}

	imp.BidFloor = convertedValue
	imp.BidFloorCur = bidderCurrency
	return nil
}

// MakeRequests creates AMX adapter requests
func (adapter *AMXAdapter) MakeRequests(request *openrtb2.BidRequest, req *adapters.ExtraRequestInfo) (reqsBidder []*adapters.RequestData, errs []error) {
	reqCopy := *request

	var publisherID string
	hasBidFloor := false
	for idx, imp := range reqCopy.Imp {
		if err := resolveBidFloor(&imp, req); err != nil {
			errs = append(errs, err)
			return nil, errs
		}
		if imp.BidFloor > 0 {
			hasBidFloor = true
		}

		var params amxExt
		if err := jsonutil.Unmarshal(imp.Ext, &params); err == nil {
			if params.Bidder.TagID != "" {
				publisherID = params.Bidder.TagID
			}

			// if it has an adUnitId, set as the tagid
			if params.Bidder.AdUnitID != "" {
				imp.TagID = params.Bidder.AdUnitID
			}
		}

		reqCopy.Imp[idx] = imp
	}

	if hasBidFloor && len(reqCopy.Cur) > 0 {
		reqCopy.Cur = []string{bidderCurrency}
	}

	if publisherID != "" {
		if reqCopy.App != nil {
			publisher := ensurePublisherWithID(reqCopy.App.Publisher, publisherID)
			appCopy := *request.App
			appCopy.Publisher = &publisher
			reqCopy.App = &appCopy
		}
		if reqCopy.Site != nil {
			publisher := ensurePublisherWithID(reqCopy.Site.Publisher, publisherID)
			siteCopy := *request.Site
			siteCopy.Publisher = &publisher
			reqCopy.Site = &siteCopy
		}
	}

	encoded, err := json.Marshal(reqCopy)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	reqBidder := &adapters.RequestData{
		Method:  "POST",
		Uri:     adapter.endpoint,
		Body:    encoded,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(reqCopy.Imp),
	}
	reqsBidder = append(reqsBidder, reqBidder)
	return
}

type amxBidExt struct {
	StartDelay   *int    `json:"startdelay,omitempty"`
	CreativeType *int    `json:"ct,omitempty"`
	DemandSource *string `json:"ds,omitempty"`
	BidderCode   *string `json:"bc,omitempty"`
}

// MakeBids will parse the bids from the AMX server
func (adapter *AMXAdapter) MakeBids(request *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if http.StatusNoContent == response.StatusCode {
		return nil, nil
	}

	if http.StatusBadRequest == response.StatusCode {
		internalNBR := response.Headers.Get(nbrHeaderName)
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Invalid Request: 400. Error Code: %s", internalNBR),
		}}
	}

	if http.StatusOK != response.StatusCode {
		internalNBR := response.Headers.Get(nbrHeaderName)
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected response: %d. Error Code: %s", response.StatusCode, internalNBR),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)
	if bidResp.Cur != "" {
		bidResponse.Currency = bidResp.Cur
	}

	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			bid := bid
			bidExt, bidExtErr := getBidExt(bid.Ext)
			if bidExtErr != nil {
				errs = append(errs, bidExtErr)
				continue
			}

			demandSource := ""
			if bidExt.DemandSource != nil {
				demandSource = *bidExt.DemandSource
			}

			bidType := getMediaTypeForBid(bidExt)
			b := &adapters.TypedBid{
				Bid: &bid,
				BidMeta: &openrtb_ext.ExtBidPrebidMeta{
					AdvertiserDomains: bid.ADomain,
					DemandSource:      demandSource,
				},
				BidType: bidType,
			}

			if bidExt.BidderCode != nil {
				b.Seat = openrtb_ext.BidderName(*bidExt.BidderCode)
			}

			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}

	return bidResponse, errs
}

func getBidExt(ext json.RawMessage) (amxBidExt, error) {
	if len(ext) == 0 {
		return amxBidExt{}, nil
	}

	var bidExt amxBidExt
	err := jsonutil.Unmarshal(ext, &bidExt)
	return bidExt, err
}

func getMediaTypeForBid(bidExt amxBidExt) openrtb_ext.BidType {
	if bidExt.StartDelay != nil {
		return openrtb_ext.BidTypeVideo
	}

	if bidExt.CreativeType != nil && *bidExt.CreativeType == 10 {
		return openrtb_ext.BidTypeNative
	}

	return openrtb_ext.BidTypeBanner
}
