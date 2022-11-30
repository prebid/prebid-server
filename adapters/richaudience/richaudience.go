package richaudience

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the RichAudience adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requestDataRequest []*adapters.RequestData

	errs := make([]error, 0, len(request.Imp))

	raiHeaders := http.Header{}
	setHeaders(&raiHeaders)

	isUrlSecure := getIsUrlSecure(request)

	if err := validateDevice(request); err != nil {
		errs = append(errs, &errortypes.BadInput{
			Message: err.Error(),
		})
		return nil, errs
	}

	for _, imp := range request.Imp {
		var secure = int8(0)

		raiExt, err := parseImpExt(&imp)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			return nil, errs
		}

		if raiExt != nil {
			if raiExt.Pid != "" {
				imp.TagID = raiExt.Pid
			}

			if raiExt.Test {
				request.Device.IP = "11.222.33.44"
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

		if imp.Banner != nil {
			if imp.Banner.W == nil && imp.Banner.H == nil {
				if len(imp.Banner.Format) == 0 {
					errs = append(errs, &errortypes.BadInput{
						Message: "request.Banner.Format is required",
					})
					return nil, errs
				}
			}
		}

		request.Imp = []openrtb2.Imp{imp}

		req, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			return nil, errs
		}

		requestDataRequest = append(requestDataRequest, &adapters.RequestData{
			Method:  "POST",
			Uri:     a.endpoint,
			Body:    req,
			Headers: raiHeaders,
		})

	}

	return requestDataRequest, nil
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
				BidType: getMediaType(seatBid.Bid[i].ImpID, request.Imp),
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

func getMediaType(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				return openrtb_ext.BidTypeVideo
			}
			return openrtb_ext.BidTypeBanner
		}
	}
	return openrtb_ext.BidTypeBanner
}
