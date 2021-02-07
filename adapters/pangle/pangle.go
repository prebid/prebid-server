package pangle

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type pangleAdapter struct {
	Endpoint string
}

type wrappedExtImpBidder struct {
	*adapters.ExtImpBidder
	AdType int `json:"adtype,omitempty"`
}

type pangleBidExt struct {
	Pangle *bidExt `json:"pangle,omitempty"`
}

type bidExt struct {
	AdType *int `json:"adtype,omitempty"`
}

/* Builder */

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &pangleAdapter{
		Endpoint: config.Endpoint,
	}

	return bidder, nil
}

/* MakeRequests */

func getAdType(imp openrtb.Imp, parsedImpExt *wrappedExtImpBidder) (adType int) {
	adType = -1
	// video
	if imp.Video != nil {
		if parsedImpExt != nil && parsedImpExt.Prebid != nil && parsedImpExt.Prebid.IsRewardedInventory == 1 {
			adType = 7
			return
		}
		if imp.Instl == 1 {
			adType = 8
			return
		}
	}
	// banner
	if imp.Banner != nil {
		if imp.Instl == 1 {
			adType = 2
		} else {
			adType = 1
		}
		return
	}
	// native
	if imp.Native != nil && len(imp.Native.Request) > 0 {
		adType = 5
		return
	}

	return
}

func (pa *pangleAdapter) MakeRequests(request *openrtb.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errs []error

	requestCopy := *request
	for _, imp := range request.Imp {
		var impExt wrappedExtImpBidder
		if err := json.Unmarshal(imp.Ext, &impExt); err != nil {
			errs = append(errs, fmt.Errorf("failed unmarshalling imp ext (err)%s", err.Error()))
			continue
		}
		// detect and fill adtype
		if adType := getAdType(imp, &impExt); adType == -1 {
			errs = append(errs, fmt.Errorf("not a supported adtype"))
			continue
		} else {
			impExt.AdType = adType
			if newImpExt, err := json.Marshal(impExt); err == nil {
				imp.Ext = newImpExt
			} else {
				errs = append(errs, fmt.Errorf("failed re-marshalling imp ext with adtype"))
				continue
			}
		}
		// for setting token
		var bidderImpExt openrtb_ext.ImpExtPangle
		if err := json.Unmarshal(impExt.Bidder, &bidderImpExt); err != nil {
			errs = append(errs, fmt.Errorf("failed unmarshalling bidder imp ext (err)%s", err.Error()))
			continue
		}

		requestCopy.Imp = []openrtb.Imp{imp}
		requestJSON, err := json.Marshal(requestCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requestData := &adapters.RequestData{
			Method: "POST",
			Uri:    pa.Endpoint,
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

func getMediaTypeForBid(bid *openrtb.Bid) (openrtb_ext.BidType, error) {
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

func (p *pangleAdapter) MakeBids(request *openrtb.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var response openrtb.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	var errs []error
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for _, bid := range seatBid.Bid {
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
