package pangle

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	openrtb2 "github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
)

// SKAN IDs must be lower case
var pangleExtSKADNetIDs = map[string]bool{
	"22mmun2rn5.skadnetwork": true,
}

type adapter struct {
	Endpoint string
}

type wrappedExtImpBidder struct {
	*adapters.ExtImpBidder
	AdType int `json:"adtype,omitempty"`
}

type pangleImpExt struct {
	AdType int                `json:"adtype,omitempty"`
	SKADN  *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

type pangleBidExt struct {
	Pangle *bidExt `json:"pangle,omitempty"`
}

type bidExt struct {
	AdType *int `json:"adtype,omitempty"`
}

// Name is used for cookies and such
func (a *adapter) Name() string {
	return "pangle"
}

// SkipNoCookies ...
func (a *adapter) SkipNoCookies() bool {
	return false
}

// Call is legacy, and added only to support Pangle interface
func (a *adapter) Call(_ context.Context, _ *pbs.PBSRequest, _ *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	return pbs.PBSBidSlice{}, nil
}

// NewPangleAdapter ...
func NewPangleAdapter(config *adapters.HTTPAdapterConfig, uri string) *adapter {
	return NewPangleBidder(adapters.NewHTTPAdapter(config).Client, uri)
}

// NewPangleBidder ...
func NewPangleBidder(_ *http.Client, uri string) *adapter {
	return &adapter{
		Endpoint: uri,
	}
}

/* Builder */

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		Endpoint: config.Endpoint,
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

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errs []error

	requestCopy := *request
	for _, imp := range request.Imp {
		var impExt wrappedExtImpBidder
		if err := json.Unmarshal(imp.Ext, &impExt); err != nil {
			errs = append(errs, fmt.Errorf("failed unmarshalling imp ext (err)%s", err.Error()))
			continue
		}

		var bidderImpExt openrtb_ext.ImpExtPangle
		if err := json.Unmarshal(impExt.Bidder, &bidderImpExt); err != nil {
			errs = append(errs, fmt.Errorf("failed unmarshalling bidder imp ext (err)%s", err.Error()))
			continue
		}

		var pangleImpExt pangleImpExt

		if imp.Banner != nil && bidderImpExt.MRAIDSupported == false {
			imp.Banner = nil
		}

		if bidderImpExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(impExt.Prebid, pangleExtSKADNetIDs)
			// only add if present
			if len(skadn.SKADNetIDs) > 0 {
				pangleImpExt.SKADN = &skadn
			}
		}

		// detect and fill adtype
		if adType := getAdType(imp, &impExt); adType == -1 {
			errs = append(errs, &errortypes.BadInput{Message: "not a supported adtype"})
			continue
		} else {
			pangleImpExt.AdType = adType
			if newImpExt, err := json.Marshal(pangleImpExt); err == nil {
				imp.Ext = newImpExt
			} else {
				errs = append(errs, fmt.Errorf("failed re-marshalling imp ext with adtype"))
				continue
			}
		}

		requestCopy.Imp = []openrtb2.Imp{imp}
		requestJSON, err := json.Marshal(requestCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requestData := &adapters.RequestData{
			Method: "POST",
			Uri:    a.Endpoint,
			Body:   requestJSON,
			Headers: http.Header{
				"TOKEN":        []string{bidderImpExt.Token},
				"Content-Type": []string{"application/json"},
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
	var bidderImpExt openrtb_ext.ImpExtPangle
	if err := json.Unmarshal(parsedImpExt.Bidder, &bidderImpExt); err != nil {
		return 0, err
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

	return 0, fmt.Errorf("unable to find a tapjoy adtype")
}
