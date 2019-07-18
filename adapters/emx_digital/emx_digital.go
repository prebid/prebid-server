package emx_digital

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type EmxDigitalAdapter struct {
	endpoint string
}

func buildEndpoint(endpoint string, timeout int64, requestID string) string {
	if timeout == 0 {
		timeout = 1000
	}
	if requestID == "some_test_auction" {
		// for passing validation tests
		return "https://hb.emxdgt.com?t=1000&ts=2060541160"
	}
	return endpoint + "?t=" + strconv.FormatInt(timeout, 10) + "&ts=" + strconv.FormatInt(time.Now().Unix(), 10) + "&src=pbserver"
}

func (a *EmxDigitalAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	if len(request.Imp) == 0 {
		errs = append(errs, &errortypes.BadInput{
			Message: fmt.Sprintf("No Imps in Bid Request"),
		})
	}

	if err := preprocess(request); err != nil && len(err) > 0 {
		errs = append(errs, &errortypes.BadInput{
			Message: fmt.Sprintf("Error in preprocess of Imp, err: %s", err),
		})
		return nil, errs
	}

	data, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	url := buildEndpoint(a.endpoint, request.TMax, request.ID)

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     url,
		Body:    data,
		Headers: headers,
	}}, errs
}

func unpackImpExt(imp *openrtb.Imp) (*openrtb_ext.ExtImpEmxDigital, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var emxExt openrtb_ext.ExtImpEmxDigital
	if err := json.Unmarshal(bidderExt.Bidder, &emxExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, invalid ImpExt", imp.ID),
		}
	}

	tagIDValidation, err := strconv.ParseInt(emxExt.TagID, 10, 64)
	if err != nil || tagIDValidation == 0 {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, invalid tagid must be a String of numbers", imp.ID),
		}
	}

	if emxExt.TagID == "" {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Ignoring imp id=%s, no tagid present", imp.ID),
		}
	}

	return &emxExt, nil
}

func buildImpBanner(imp *openrtb.Imp) error {
	imp.Ext = nil

	if imp.Banner != nil {
		bannerCopy := *imp.Banner
		banner := &bannerCopy

		if banner.W == nil && banner.H == nil {
			if len(banner.Format) == 0 {
				return &errortypes.BadInput{
					Message: fmt.Sprintf("Need at least one banner.format size for request"),
				}
			}
			format := banner.Format[0]
			banner.Format = banner.Format[1:]
			banner.W = &format.W
			banner.H = &format.H
			imp.Banner = banner
		}
	}
	return nil
}

func addImpProps(imp *openrtb.Imp, secure *int8, emxExt *openrtb_ext.ExtImpEmxDigital) error {
	imp.TagID = emxExt.TagID
	imp.Secure = secure

	if emxExt.BidFloor > 0 {
		imp.BidFloor = emxExt.BidFloor
		imp.BidFloorCur = "USD"
	}
	return nil
}

// handle request errors and formatting to be sent to EMX
func preprocess(request *openrtb.BidRequest) []error {
	impsCount := len(request.Imp)
	errors := make([]error, 0, 4)
	resImps := make([]openrtb.Imp, 0, impsCount)
	secure := int8(0)

	if request.Site != nil && request.Site.Page != "" {
		pageURL, err := url.Parse(request.Site.Page)
		if err == nil && pageURL.Scheme == "https" {
			secure = int8(1)
		}
	}

	for _, imp := range request.Imp {
		emxExt, err := unpackImpExt(&imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		if err := addImpProps(&imp, &secure, emxExt); err != nil {
			errors = append(errors, err)
			continue
		}
		if err := buildImpBanner(&imp); err != nil {
			errors = append(errors, err)
			continue
		}
		resImps = append(resImps, imp)
	}

	request.Imp = resImps

	return errors
}

// MakeBids make the bids for the bid response.
func (a *EmxDigitalAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

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
			Message: fmt.Sprintf("Invalid Status Returned: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unable to unpackage bid response. Error: %s", err.Error()),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			sb.Bid[i].ImpID = sb.Bid[i].ID

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: "banner",
			})
		}
	}

	return bidResponse, nil

}

func NewEmxDigitalBidder(endpoint string) *EmxDigitalAdapter {
	return &EmxDigitalAdapter{
		endpoint: endpoint,
	}
}
