package pubnative

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

type PubnativeAdapter struct {
	URI string
}

func (a *PubnativeAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, err)
			continue
		}

		var pubnativeExt openrtb_ext.ExtImpPubnative
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &pubnativeExt); err != nil {
			errs = append(errs, err)
			continue
		}

		err := convertImpression(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requestCopy.Imp = []openrtb2.Imp{imp}
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
			ImpIDs:  openrtb_ext.GetImpIDs(requestCopy.Imp),
		}

		requestData = append(requestData, reqData)
	}
	return requestData, errs
}

func checkRequest(request *openrtb2.BidRequest) error {
	if request.Device == nil || len(request.Device.OS) == 0 {
		return &errortypes.BadInput{
			Message: "Impression is missing device OS information",
		}
	}

	return nil
}

func convertImpression(imp *openrtb2.Imp) error {
	if imp.Banner == nil && imp.Video == nil && imp.Native == nil {
		return &errortypes.BadInput{
			Message: "Pubnative only supports banner, video or native ads.",
		}
	}
	if imp.Banner != nil {
		bannerCopy, err := convertBanner(imp.Banner)
		if err != nil {
			return err
		}
		imp.Banner = bannerCopy
	}

	return nil
}

// make sure that banner has openrtb 2.3-compatible size information
func convertBanner(banner *openrtb2.Banner) (*openrtb2.Banner, error) {
	if banner.W == nil || banner.H == nil || *banner.W == 0 || *banner.H == 0 {
		if len(banner.Format) > 0 {
			f := banner.Format[0]

			bannerCopy := *banner

			bannerCopy.W = ptrutil.ToPtr(f.W)
			bannerCopy.H = ptrutil.ToPtr(f.H)

			return &bannerCopy, nil
		} else {
			return nil, &errortypes.BadInput{
				Message: "Size information missing for banner",
			}
		}
	}
	return banner, nil
}

func (a *PubnativeAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var parsedResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &parsedResponse); err != nil {
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

// Builder builds a new instance of the Pubnative adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &PubnativeAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}

func getMediaTypeForImp(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
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
