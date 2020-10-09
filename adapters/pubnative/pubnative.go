package pubnative

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type PubnativeAdapter struct {
	URI string
}

func (a *PubnativeAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	impCount := len(request.Imp)
	requestData := make([]*adapters.RequestData, 0, impCount)
	errs := []error{}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	err := checkRequest(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	for _, imp := range request.Imp {
		requestCopy := *request
		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, err)
			continue
		}

		var pubnativeExt openrtb_ext.ExtImpPubnative
		if err := json.Unmarshal(bidderExt.Bidder, &pubnativeExt); err != nil {
			errs = append(errs, err)
			continue
		}

		err := convertImpression(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requestCopy.Imp = []openrtb.Imp{imp}
		reqJSON, err := json.Marshal(&requestCopy)
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}

		queryParams := url.Values{}
		queryParams.Add("apptoken", pubnativeExt.AppAuthToken)
		queryParams.Add("zoneid", strconv.Itoa(pubnativeExt.ZoneID))
		queryString := queryParams.Encode()

		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     fmt.Sprintf("%s?%s", a.URI, queryString),
			Body:    reqJSON,
			Headers: headers,
		}

		requestData = append(requestData, reqData)
	}
	return requestData, errs
}

func checkRequest(request *openrtb.BidRequest) error {
	if request.Device == nil || len(request.Device.OS) == 0 {
		return &errortypes.BadInput{
			Message: "Impression is missing device OS information",
		}
	}

	return nil
}

func convertImpression(imp *openrtb.Imp) error {
	if imp.Banner == nil && imp.Video == nil && imp.Native == nil {
		return &errortypes.BadInput{
			Message: "Pubnative only supports banner, video or native ads.",
		}
	}
	if imp.Banner != nil {
		err := convertBanner(imp.Banner)
		if err != nil {
			return err
		}
	}

	return nil
}

// make sure that banner has openrtb 2.3-compatible size information
func convertBanner(banner *openrtb.Banner) error {
	if banner.W == nil || banner.H == nil || *banner.W == 0 || *banner.H == 0 {
		if len(banner.Format) > 0 {
			f := banner.Format[0]
			banner.W = &f.W
			banner.H = &f.H
		} else {
			return &errortypes.BadInput{
				Message: "Size information missing for banner",
			}
		}
	}
	return nil
}

func (a *PubnativeAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var parsedResponse openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &parsedResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range parsedResponse.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			if bid.Price != 0 {
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: getMediaTypeForImp(bid.ImpID, internalRequest.Imp),
				})
			}
		}
	}
	return bidResponse, nil
}

func NewPubnativeBidder(uri string) *PubnativeAdapter {
	return &PubnativeAdapter{URI: uri}
}

func getMediaTypeForImp(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			} else if imp.Native != nil {
				mediaType = openrtb_ext.BidTypeNative
			}
			return mediaType
		}
	}
	return mediaType
}
