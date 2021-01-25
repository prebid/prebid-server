package decenterads

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type DecenterAdsAdapter struct {
	endpoint string
}

func (a *DecenterAdsAdapter) MakeRequests(request *openrtb.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	impressions := request.Imp
	result := make([]*adapters.RequestData, 0, len(impressions))
	errs := make([]error, 0, len(impressions))

	for _, impression := range impressions {
		if impression.Banner == nil && impression.Video == nil && impression.Native == nil {
			errs = append(errs, &errortypes.BadInput{
				Message: "DecenterAds only supports banner, video or native ads",
			})
			continue
		}
		if len(impression.Ext) == 0 {
			errs = append(errs, errors.New("impression extensions required"))
			continue
		}
		var bidderExt adapters.ExtImpBidder
		err := json.Unmarshal(impression.Ext, &bidderExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if len(bidderExt.Bidder) == 0 {
			errs = append(errs, errors.New("bidder required"))
			continue
		}
		var decenteradsExt openrtb_ext.ExtImpDecenterAds
		err = json.Unmarshal(bidderExt.Bidder, &decenteradsExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if decenteradsExt.PlacementID == "" {
			errs = append(errs, errors.New("DecenterAds placementId required"))
			continue
		}
		impExtJSON, err := json.Marshal(decenteradsExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		impression.Ext = impExtJSON
		request.Imp = []openrtb.Imp{impression}
		body, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		result = append(result, &adapters.RequestData{
			Method:  "POST",
			Uri:     a.endpoint,
			Body:    body,
			Headers: headers,
		})
	}

	request.Imp = impressions

	if len(result) == 0 {
		return nil, errs
	}
	return result, errs
}

func (a *DecenterAdsAdapter) MakeBids(request *openrtb.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	switch responseData.StatusCode {
	case http.StatusNoContent:
		return nil, nil
	case http.StatusBadRequest:
		return nil, []error{&errortypes.BadInput{
			Message: "unexpected status code: " + strconv.Itoa(responseData.StatusCode),
		}}
	case http.StatusOK:
		break
	default:
		return nil, []error{&errortypes.BadServerResponse{
			Message: "unexpected status code: " + strconv.Itoa(responseData.StatusCode),
		}}
	}

	var bidResponse openrtb.BidResponse
	err := json.Unmarshal(responseData.Body, &bidResponse)
	if err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	response := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))

	for _, seatBid := range bidResponse.SeatBid {
		for _, bid := range seatBid.Bid {
			bid := bid // pin https://github.com/kyoh86/scopelint#whats-this
			var bidType openrtb_ext.BidType
			for _, impression := range request.Imp {
				if impression.ID != bid.ImpID {
					continue
				}
				switch {
				case impression.Banner != nil:
					bidType = openrtb_ext.BidTypeBanner
				case impression.Video != nil:
					bidType = openrtb_ext.BidTypeVideo
				case impression.Native != nil:
					bidType = openrtb_ext.BidTypeNative
				}
				break
			}
			if bidType == "" {
				errs = append(errs, &errortypes.BadServerResponse{
					Message: "ignoring bid id=" + bid.ID + ", request doesn't contain any valid impression with id=" + bid.ImpID,
				})
				continue
			}
			response.Bids = append(response.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}

	return response, errs
}

// Builder builds a new instance of the DecenterAds adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &DecenterAdsAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
