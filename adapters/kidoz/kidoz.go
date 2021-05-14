package kidoz

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type KidozAdapter struct {
	endpoint string
}

func (a *KidozAdapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")

	impressions := request.Imp
	result := make([]*adapters.RequestData, 0, len(impressions))
	errs := make([]error, 0, len(impressions))

	for i, impression := range impressions {
		if impression.Banner == nil && impression.Video == nil {
			errs = append(errs, &errortypes.BadInput{
				Message: "Kidoz only supports banner or video ads",
			})
			continue
		}

		if impression.Banner != nil {
			banner := impression.Banner
			if banner.Format == nil {
				errs = append(errs, &errortypes.BadInput{
					Message: "banner format required",
				})
				continue
			}
			if len(banner.Format) == 0 {
				errs = append(errs, &errortypes.BadInput{
					Message: "banner format array is empty",
				})
				continue
			}
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
		var impressionExt openrtb_ext.ExtImpKidoz
		err = json.Unmarshal(bidderExt.Bidder, &impressionExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if impressionExt.AccessToken == "" {
			errs = append(errs, errors.New("Kidoz access_token required"))
			continue
		}
		if impressionExt.PublisherID == "" {
			errs = append(errs, errors.New("Kidoz publisher_id required"))
			continue
		}

		request.Imp = impressions[i : i+1]
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

func (a *KidozAdapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	switch responseData.StatusCode {
	case http.StatusNoContent:
		fallthrough
	case http.StatusServiceUnavailable:
		return nil, nil

	case http.StatusBadRequest:
		fallthrough
	case http.StatusUnauthorized:
		fallthrough
	case http.StatusForbidden:
		return nil, []error{&errortypes.BadInput{
			Message: "unexpected status code: " + strconv.Itoa(responseData.StatusCode) + " " + string(responseData.Body),
		}}

	case http.StatusOK:
		break

	default:
		return nil, []error{&errortypes.BadServerResponse{
			Message: "unexpected status code: " + strconv.Itoa(responseData.StatusCode) + " " + string(responseData.Body),
		}}
	}

	var bidResponse openrtb2.BidResponse
	err := json.Unmarshal(responseData.Body, &bidResponse)
	if err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	response := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))

	for _, seatBid := range bidResponse.SeatBid {
		for _, bid := range seatBid.Bid {
			thisBid := bid
			bidType := GetMediaTypeForImp(bid.ImpID, request.Imp)
			if bidType == UndefinedMediaType {
				errs = append(errs, &errortypes.BadServerResponse{
					Message: "ignoring bid id=" + bid.ID + ", request doesn't contain any valid impression with id=" + bid.ImpID,
				})
				continue
			}
			response.Bids = append(response.Bids, &adapters.TypedBid{
				Bid:     &thisBid,
				BidType: bidType,
			})
		}
	}

	return response, errs
}

// Builder builds a new instance of the Kidoz adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &KidozAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

const UndefinedMediaType = openrtb_ext.BidType("")

func GetMediaTypeForImp(impID string, imps []openrtb2.Imp) openrtb_ext.BidType {
	var bidType openrtb_ext.BidType = UndefinedMediaType
	for _, impression := range imps {
		if impression.ID != impID {
			continue
		}
		switch {
		case impression.Banner != nil:
			bidType = openrtb_ext.BidTypeBanner
		case impression.Video != nil:
			bidType = openrtb_ext.BidTypeVideo
		case impression.Native != nil:
			bidType = openrtb_ext.BidTypeNative
		case impression.Audio != nil:
			bidType = openrtb_ext.BidTypeAudio
		}
		break
	}
	return bidType
}
