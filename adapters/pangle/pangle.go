package pangle

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

type adapter struct {
	Endpoint string
}

type NetworkIDs struct {
	AppID       string `json:"appid,omitempty"`
	PlacementID string `json:"placementid,omitempty"`
}

type wrappedExtImpBidder struct {
	*adapters.ExtImpBidder
	AdType     int         `json:"adtype,omitempty"`
	IsPrebid   bool        `json:"is_prebid,omitempty"`
	NetworkIDs *NetworkIDs `json:"networkids,omitempty"`
}

type pangleBidExt struct {
	Pangle *bidExt `json:"pangle,omitempty"`
}

type bidExt struct {
	AdType *int `json:"adtype,omitempty"`
}

/* Builder */

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		Endpoint: config.Endpoint,
	}

	return bidder, nil
}

/* MakeRequests */

func getAdType(imp openrtb2.Imp, parsedImpExt *wrappedExtImpBidder) int {
	// video
	if imp.Video != nil {
		if parsedImpExt != nil && parsedImpExt.Prebid != nil && parsedImpExt.Prebid.IsRewardedInventory != nil && *parsedImpExt.Prebid.IsRewardedInventory == 1 {
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
		if err := jsonutil.Unmarshal(imp.Ext, &impExt); err != nil {
			errs = append(errs, fmt.Errorf("failed unmarshalling imp ext (err)%s", err.Error()))
			continue
		}
		// get token & networkIDs
		var bidderImpExt openrtb_ext.ImpExtPangle
		if err := jsonutil.Unmarshal(impExt.Bidder, &bidderImpExt); err != nil {
			errs = append(errs, fmt.Errorf("failed unmarshalling bidder imp ext (err)%s", err.Error()))
			continue
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
		if newImpExt, err := json.Marshal(impExt); err == nil {
			imp.Ext = newImpExt
		} else {
			errs = append(errs, fmt.Errorf("failed re-marshalling imp ext"))
			continue
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
			ImpIDs: openrtb_ext.GetImpIDs(requestCopy.Imp),
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
	if err := jsonutil.Unmarshal(bid.Ext, &bidExt); err != nil {
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
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
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
