package decenterads

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	impressions := request.Imp
	result := make([]*adapters.RequestData, 0, len(impressions))
	errs := make([]error, 0, len(impressions))

	for _, impression := range impressions {
		var impExt map[string]json.RawMessage
		if err := jsonutil.Unmarshal(impression.Ext, &impExt); err != nil {
			errs = append(errs, fmt.Errorf("unable to parse bidder parameers: %s", err))
			continue
		}

		bidderExt, bidderExtExists := impExt["bidder"]
		if !bidderExtExists || len(bidderExt) == 0 {
			errs = append(errs, errors.New("bidder parameters required"))
			continue
		}

		impression.Ext = bidderExt
		request.Imp = []openrtb2.Imp{impression}
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
			ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
		})
	}

	request.Imp = impressions
	return result, errs
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var bidResponse openrtb2.BidResponse
	err := jsonutil.Unmarshal(responseData.Body, &bidResponse)
	if err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	response := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))

	for _, seatBid := range bidResponse.SeatBid {
		for i := range seatBid.Bid {
			response.Bids = append(response.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: getMediaTypeForImp(seatBid.Bid[i].ImpID, request.Imp),
			})
		}
	}

	return response, errs
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner
			} else if imp.Video != nil {
				return openrtb_ext.BidTypeVideo
			} else if imp.Native != nil {
				return openrtb_ext.BidTypeNative
			} else if imp.Audio != nil {
				return openrtb_ext.BidTypeAudio
			}
		}
	}
	return openrtb_ext.BidTypeBanner
}

// Builder builds a new instance of the DecenterAds adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
