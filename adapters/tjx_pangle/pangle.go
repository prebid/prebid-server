package pangle

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

// Region ...
type Region string

const (
	USEast Region = "us_east"
	SG     Region = "sg"
)

// SKAN IDs must be lower case
var pangleExtSKADNetIDs = map[string]bool{
	"22mmun2rn5.skadnetwork": true,
}

type adapter struct {
	Endpoint         string
	SupportedRegions map[Region]string
}

type NetworkIDs struct {
	AppID       string `json:"appid,omitempty"`
	PlacementID string `json:"placementid,omitempty"`
}

type wrappedExtImpBidder struct {
	*adapters.ExtImpBidder
	AdType     int                `json:"adtype,omitempty"`
	IsPrebid   bool               `json:"is_prebid,omitempty"`
	NetworkIDs *NetworkIDs        `json:"networkids,omitempty"`
	SKADN      *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

type pangleImpExtBidder struct {
	AppID       string `json:"appid,omitempty"`
	Token       string `json:"token,omitempty"`
	PlacementID string `json:"placementid,omitempty"`
}

type pangleBidExt struct {
	Pangle *bidExt `json:"pangle,omitempty"`
}

type bidExt struct {
	AdType *int `json:"adtype,omitempty"`
}

/* Builder */

func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		Endpoint: config.Endpoint,
		SupportedRegions: map[Region]string{
			USEast: config.XAPI.EndpointUSEast,
			SG:     config.XAPI.EndpointSG,
		},
	}

	return bidder, nil
}

/* MakeRequests */

func getAdType(imp openrtb2.Imp, parsedImpExt *wrappedExtImpBidder) int {
	// attempt to get tj adtype and return if successful
	if adType, err := getTjAdType(imp, parsedImpExt); err == nil {
		return adType
	}

	// video
	if imp.Video != nil {
		if parsedImpExt != nil && parsedImpExt.Prebid != nil && parsedImpExt.Prebid.IsRewardedInventory == 1 {
			return 7
		}
		if imp.Instl == 1 {
			return 8
		}
	}
	// banner
	if imp.Banner != nil {
		if imp.Instl == 1 {
			return 2
		} else {
			return 1
		}
	}
	// native
	if imp.Native != nil && len(imp.Native.Request) > 0 {
		return 5
	}

	return -1
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errs []error

	// copy the bidder request
	pangleRequest := *request
	for _, imp := range pangleRequest.Imp {
		skanSent := false

		var impExt wrappedExtImpBidder
		if err := json.Unmarshal(imp.Ext, &impExt); err != nil {
			errs = append(errs, fmt.Errorf("failed unmarshalling imp ext (err)%s", err.Error()))
			continue
		}

		// get token & networkIDs
		var bidderImpExt openrtb_ext.ImpExtTJXPangle
		if err := json.Unmarshal(impExt.Bidder, &bidderImpExt); err != nil {
			errs = append(errs, fmt.Errorf("failed unmarshalling bidder imp ext (err)%s", err.Error()))
			continue
		}

		if imp.Banner != nil && !bidderImpExt.MRAIDSupported {
			imp.Banner = nil
		}

		// Overwrite BidFloor if present
		if bidderImpExt.BidFloor != nil {
			imp.BidFloor = *bidderImpExt.BidFloor
		}

		if bidderImpExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(impExt.Prebid, pangleExtSKADNetIDs)
			// only add if present
			if len(skadn.SKADNetIDs) > 0 {
				impExt.SKADN = &skadn
				skanSent = true
			}
		}

		// detect and fill adtype
		adType := getAdType(imp, &impExt)
		if adType == -1 {
			errs = append(errs, &errortypes.BadInput{Message: "not a supported adtype"})
			continue
		}

		// remarshal imp.ext
		impExt.AdType = adType
		impExt.IsPrebid = true

		if len(bidderImpExt.AppID) > 0 && len(bidderImpExt.PlacementID) > 0 {
			impExt.NetworkIDs = &NetworkIDs{
				AppID:       bidderImpExt.AppID,
				PlacementID: bidderImpExt.PlacementID,
			}
		} else if len(bidderImpExt.AppID) > 0 || len(bidderImpExt.PlacementID) > 0 {
			errs = append(errs, &errortypes.BadInput{Message: "only one of appid or placementid is provided"})
			continue
		}

		pangleImpExtBidder := pangleImpExtBidder{
			AppID:       bidderImpExt.AppID,
			Token:       bidderImpExt.Token,
			PlacementID: bidderImpExt.PlacementID,
		}

		impExt.Prebid = nil

		if newImpExtBidder, err := json.Marshal(pangleImpExtBidder); err == nil {
			impExt.Bidder = newImpExtBidder
		} else {
			errs = append(errs, fmt.Errorf("failed re-marshalling imp ext bidder"))
			continue
		}

		if newImpExt, err := json.Marshal(impExt); err == nil {
			imp.Ext = newImpExt
		} else {
			errs = append(errs, fmt.Errorf("failed re-marshalling imp ext"))
			continue
		}

		pangleRequest.Imp = []openrtb2.Imp{imp}
		pangleRequest.Ext = nil

		requestJSON, err := json.Marshal(pangleRequest)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// Tapjoy Record placement type
		placementType := adapters.Interstitial
		if bidderImpExt.Reward == 1 {
			placementType = adapters.Rewarded
		}

		uri := a.Endpoint

		if endpoint, ok := a.SupportedRegions[Region(bidderImpExt.Region)]; ok {
			uri = endpoint
		}

		requestData := &adapters.RequestData{
			Method: "POST",
			Uri:    uri,
			Body:   requestJSON,
			Headers: http.Header{
				"TOKEN":        []string{bidderImpExt.Token},
				"Content-Type": []string{"application/json"},
			},

			TapjoyData: adapters.TapjoyData{
				Bidder:        "pangle",
				PlacementType: placementType,
				Region:        bidderImpExt.Region,
				SKAN: adapters.SKAN{
					Supported: bidderImpExt.SKADNSupported,
					Sent:      skanSent,
				},
				MRAID: adapters.MRAID{
					Supported: bidderImpExt.MRAIDSupported,
				},
			},
		}
		requests = append(requests, requestData)
	}

	return requests, errs
}

/* MakeBids */

func getMediaTypeForBid(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	if bid == nil {
		return "", fmt.Errorf("the bid request object is nil")
	}

	var bidExt pangleBidExt
	if err := json.Unmarshal(bid.Ext, &bidExt); err != nil {
		return "", fmt.Errorf("invalid bid ext")
	} else if bidExt.Pangle == nil || bidExt.Pangle.AdType == nil {
		return "", fmt.Errorf("missing pangleExt/adtype in bid ext")
	}

	switch *bidExt.Pangle.AdType {
	case 1:
		return openrtb_ext.BidTypeBanner, nil
	case 2:
		return openrtb_ext.BidTypeBanner, nil
	case 5:
		return openrtb_ext.BidTypeNative, nil
	case 7:
		return openrtb_ext.BidTypeVideo, nil
	case 8:
		return openrtb_ext.BidTypeVideo, nil
	}

	return "", fmt.Errorf("unrecognized adtype in response")
}

func (a *adapter) MakeBids(_ *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var errs []error
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for _, temp := range seatBid.Bid {
			bid := temp // avoid taking address of a for loop variable
			mediaType, err := getMediaTypeForBid(&bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			b := &adapters.TypedBid{
				Bid:     &bid,
				BidType: mediaType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}

	return bidResponse, errs
}

func getTjAdType(imp openrtb2.Imp, parsedImpExt *wrappedExtImpBidder) (int, error) {
	// for setting token
	var bidderImpExt openrtb_ext.ImpExtTJXPangle
	if err := json.Unmarshal(parsedImpExt.Bidder, &bidderImpExt); err != nil {
		return -1, err
	}

	// video
	if imp.Video != nil {
		if bidderImpExt.Reward == 1 {
			return 7, nil
		} else {
			return 8, nil
		}
	}

	// banner
	if imp.Banner != nil {
		if bidderImpExt.Reward == 1 {
			return 1, nil
		}
	}

	return -1, fmt.Errorf("unable to find a tapjoy adtype")
}
