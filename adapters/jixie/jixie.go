package jixie

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Jixie adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

// Adding header fields to request header
func addHeaderIfNonEmpty(headers http.Header, headerName string, headerValue string) {
	if len(headerValue) > 0 {
		headers.Add(headerName, headerValue)
	}
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs = make([]error, 0)

	data, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	if request.Device != nil {
		addHeaderIfNonEmpty(headers, "User-Agent", request.Device.UA)
		addHeaderIfNonEmpty(headers, "X-Forwarded-For", request.Device.IP)
	}

	if request.Site != nil {
		addHeaderIfNonEmpty(headers, "Referer", request.Site.Page)
	}

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    data,
		Headers: headers,
	}}, errs
}

func containsAny(raw string, keys []string) bool {
	lowerCased := strings.ToLower(raw)
	for i := 0; i < len(keys); i++ {
		if strings.Contains(lowerCased, keys[i]) {
			return true
		}
	}
	return false

}

func getBidType(bidAdm string) openrtb_ext.BidType {
	if bidAdm != "" && containsAny(bidAdm, []string{"<?xml", "<vast"}) {
		return openrtb_ext.BidTypeVideo
	}
	return openrtb_ext.BidTypeBanner
}

// MakeBids make the bids for the bid response.
func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		// no bid response
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

	var bidResp openrtb2.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unable to unpackage bid response. Error: %s", err.Error()),
		}}
	}

	var bids []*adapters.TypedBid

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {

			sb.Bid[i].ImpID = sb.Bid[i].ID

			bids = append(bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: getBidType(sb.Bid[i].AdM),
			})
		}
	}
	adsResp := adapters.NewBidderResponseWithBidsCapacity(len(bids))
	adsResp.Bids = bids
	if bidResp.Cur != "" {
		adsResp.Currency = bidResp.Cur
	} else {
		adsResp.Currency = "USD"
	}

	return adsResp, nil

}
