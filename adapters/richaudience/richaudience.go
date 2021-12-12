package richaudience

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the RichAudience adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	raiHeaders := http.Header{}

	setHeaders(&raiHeaders)

	isUrlSecure := getIsUrlSecure(request)

	resImps, err := setImp(request, isUrlSecure)
	if err != nil {
		return nil, []error{err}
	}

	request.Imp = resImps

	if err = validateDevice(request); err != nil {
		return nil, []error{err}
	}

	req, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    req,
		Headers: raiHeaders,
	}

	return []*adapters.RequestData{requestData}, nil
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

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &bidResp); err != nil {

		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: openrtb_ext.BidTypeBanner,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}

	return bidResponse, nil
}

func setHeaders(raiHeaders *http.Header) {
	raiHeaders.Set("Content-Type", "application/json;charset=utf-8")
	raiHeaders.Set("Accept", "application/json")
	raiHeaders.Add("X-Openrtb-Version", "2.5")
}

func setImp(request *openrtb2.BidRequest, isUrlSecure bool) (resImps []openrtb2.Imp, err error) {
	for _, imp := range request.Imp {
		var secure = int8(0)
		raiExt, errImp := parseImpExt(&imp)
		if errImp != nil {
			return nil, errImp
		}

		if raiExt != nil {
			if raiExt.Pid != "" {
				imp.TagID = raiExt.Pid
			}

			if raiExt.Test {
				request.Test = int8(1)
			}

			if raiExt.BidFloorCur != "" {
				imp.BidFloorCur = raiExt.BidFloorCur
			} else if imp.BidFloorCur == "" {
				imp.BidFloorCur = "USD"
			}
		}
		if isUrlSecure {
			secure = int8(1)
		}

		imp.Secure = &secure

		if imp.Banner.W == nil && imp.Banner.H == nil {
			if len(imp.Banner.Format) == 0 {
				err = &errortypes.BadInput{
					Message: "request.Banner.Format is required",
				}
				return nil, err
			}
		}

		resImps = append(resImps, imp)

	}
	return resImps, nil
}

func getIsUrlSecure(request *openrtb2.BidRequest) (isUrlSecure bool) {
	if request.Site != nil {
		if request.Site.Page != "" {
			pageURL, err := url.Parse(request.Site.Page)
			if err == nil {
				if request.Site.Domain == "" {
					request.Site.Domain = pageURL.Host
				}
				isUrlSecure = pageURL.Scheme == "https"
			}
		}
	}
	return
}

func validateDevice(request *openrtb2.BidRequest) (err error) {

	if request.Device != nil && request.Device.IP == "" && request.Device.IPv6 == "" {
		err = &errortypes.BadInput{
			Message: "request.Device.IP is required",
		}
		return err
	}
	return err
}

func parseImpExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpRichaudience, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		err = &errortypes.BadInput{
			Message: fmt.Sprintf("not found parameters ext in ImpID : %s", imp.ID),
		}
		return nil, err
	}

	var richaudienceExt openrtb_ext.ExtImpRichaudience
	if err := json.Unmarshal(bidderExt.Bidder, &richaudienceExt); err != nil {
		err = &errortypes.BadInput{
			Message: fmt.Sprintf("invalid parameters ext in ImpID: %s", imp.ID),
		}
		return nil, err
	}

	return &richaudienceExt, nil
}
