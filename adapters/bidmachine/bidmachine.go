package bidmachine

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type BidmachineAdapter struct {
	endpoint template.Template
}

func (a *BidmachineAdapter) MakeRequests(request *openrtb.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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
				Message: "Bidmachine supports only banner or video ads",
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
		var impressionExt openrtb_ext.ExtImpBidmachine
		err = json.Unmarshal(bidderExt.Bidder, &impressionExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		url, err := a.buildEndpointURL(impressionExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if impressionExt.SellerID == "" {
			errs = append(errs, errors.New("Bidmachine seller_id is required"))
			continue
		}
		if impressionExt.Path == "" {
			errs = append(errs, errors.New("Bidmachine path is required"))
			continue
		}
		if bidderExt.Prebid != nil && bidderExt.Prebid.IsRewardedInventory == 1 {
			if impression.Banner != nil && !hasRewardedBattr(impression.Banner.BAttr) {
				impression.Banner.BAttr = append(impression.Banner.BAttr, openrtb.CreativeAttribute(16))

			}
			if impression.Video != nil && !hasRewardedBattr(impression.Video.BAttr) {
				impression.Video.BAttr = append(impression.Video.BAttr, openrtb.CreativeAttribute(16))
			}
		}
		request.Imp = impressions[i : i+1]
		body, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		result = append(result, &adapters.RequestData{
			Method:  "POST",
			Uri:     url + "/" + impressionExt.Path + "/" + impressionExt.SellerID,
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

func hasRewardedBattr(attr []openrtb.CreativeAttribute) bool {
	for i := 0; i < len(attr); i++ {
		if attr[i] == openrtb.CreativeAttribute(16) {
			return true
		}
	}
	return false
}

func (a *BidmachineAdapter) MakeBids(request *openrtb.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

// Builder builds a new instance of the Bidmachine adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &BidmachineAdapter{
		endpoint: *template,
	}

	return bidder, nil
}

const UndefinedMediaType = openrtb_ext.BidType("")

func (a *BidmachineAdapter) buildEndpointURL(params openrtb_ext.ExtImpBidmachine) (string, error) {
	endpointParams := macros.EndpointTemplateParams{Host: params.Host}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func GetMediaTypeForImp(impID string, imps []openrtb.Imp) openrtb_ext.BidType {
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
		}
		break
	}
	return bidType
}
