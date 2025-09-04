package showheroes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type ShowheroesAdapter struct {
	endpoint string
}

type shExtImpBidder struct {
	Prebid *openrtb_ext.ExtImpPrebid    `json:"prebid,omitempty"`
	Bidder openrtb_ext.ExtImpShowheroes `json:"bidder,omitempty"`
	Gpid   string                       `json:"gpid,omitempty"`
	Tid    string                       `json:"tid,omitempty"`
	Data   json.RawMessage              `json:"data,omitempty"`
	Params openrtb_ext.ExtImpShowheroes `json:"params"`
}

// Builder builds a new instance of the Showheroes adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &ShowheroesAdapter{
		endpoint: config.Endpoint,
	}, nil
}

func validate(request *openrtb2.BidRequest) error {
	if request.Site != nil && request.Site.Page == "" {
		return &errortypes.BadInput{
			Message: "site request doesn't have a page URL",
		}
	}

	if request.App != nil && request.App.Bundle == "" {
		return &errortypes.BadInput{
			Message: "app request doesn't have a bundle ID",
		}
	}

	return nil
}

func processImp(imp *openrtb2.Imp, reqInfo *adapters.ExtraRequestInfo) error {
	if imp.Banner == nil && imp.Video == nil {
		return &errortypes.BadInput{
			Message: "banner or video must be specified",
		}
	}

	var bidderExt shExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil || bidderExt.Bidder.UnitID == "" {
		return &errortypes.BadInput{
			Message: "Error parsing bidder params",
		}
	}

	// move params from .bidder to .params
	// this is required since openrtb_ext.ExtImpShowheroes is used in 2 places:
	// 1. shExtImpBidder.Bidder - for parsing the incoming request
	// 2. shExtImpBidder.Params - for marshaling the outgoing request to showheroes
	bidderExt.Params.UnitID = bidderExt.Bidder.UnitID

	impExt, err := jsonutil.Marshal(bidderExt)
	if err != nil {
		return err
	}
	imp.Ext = impExt

	if imp.BidFloor == 0 || imp.BidFloorCur == "EUR" {
		return nil
	}

	// convert the bid floor to EUR
	currency := imp.BidFloorCur
	// default currency according to the openRTB is USD
	if currency == "" {
		currency = "USD"
	}

	eurFloor, err := reqInfo.ConvertCurrency(imp.BidFloor, currency, "EUR")
	if err != nil {
		return err
	}

	imp.BidFloor = eurFloor
	imp.BidFloorCur = "EUR"

	return nil
}

func getPrebidChannel(request *openrtb2.BidRequest) (string, string) {
	var channelName string
	var channelVersion string
	reqExt := &openrtb_ext.ExtRequest{}

	if err := jsonutil.Unmarshal(request.Ext, &reqExt); err == nil && reqExt.Prebid.Channel != nil {
		channelName = reqExt.Prebid.Channel.Name
		channelVersion = reqExt.Prebid.Channel.Version
	}
	return channelName, channelVersion
}

func (a *ShowheroesAdapter) MakeRequests(request *openrtb2.BidRequest, extra *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if err := validate(request); err != nil {
		return nil, []error{err}
	}
	var errors []error
	validImps := make([]openrtb2.Imp, 0, len(request.Imp))

	prebidChannelName, channelVersion := getPrebidChannel(request)

	// pre-process the imps
	for _, imp := range request.Imp {
		if err := processImp(&imp, extra); err != nil {
			errors = append(errors, err)
			continue
		}

		// if display manager is not set and request came from prebid.js
		// store it and its version
		if imp.DisplayManager == "" {
			imp.DisplayManager = prebidChannelName
			imp.DisplayManagerVer = channelVersion
		}
		validImps = append(validImps, imp)
	}

	if len(validImps) == 0 {
		return nil, errors
	}

	request.Imp = validImps
	reqJSON, err := json.Marshal(request)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return []*adapters.RequestData{
		{
			Method:  "POST",
			Uri:     a.endpoint,
			Body:    reqJSON,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
		},
	}, errors
}

func (a *ShowheroesAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("unexpected status code: %d", response.StatusCode)}
	}
	var bidResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResponse); err != nil {
		return nil, []error{err}
	}

	bidderResponse := adapters.NewBidderResponse()
	bidderResponse.Currency = bidResponse.Cur

	for _, seatBid := range bidResponse.SeatBid {
		for _, bid := range seatBid.Bid {
			bidType, err := getBidType(bid.MType)
			if err != nil {
				return nil, []error{err}
			}

			b := &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			}
			bidderResponse.Bids = append(bidderResponse.Bids, b)
		}
	}

	return bidderResponse, nil
}

func getBidType(markupType openrtb2.MarkupType) (openrtb_ext.BidType, error) {
	switch markupType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", fmt.Errorf("unsupported mtype: %d", markupType)
	}
}
