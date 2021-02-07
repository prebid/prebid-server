package jixie

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type JixieAdapter struct {
	endpoint string
	testing  bool
}

// Builder builds a new instance of the Jixie adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &JixieAdapter{
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

func buildEndpoint(endpoint string, testing bool, timeout int64) string {

	if timeout == 0 {
		timeout = 1000
	}

	if testing {
		return endpoint + "?hbofftest=1"
	}
	return endpoint
}

func (a *JixieAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("No Imps in Bid Request"),
		}}
	}

	for _, imp := range request.Imp {

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: err.Error()},
			}
		}
		var jxExt openrtb_ext.ExtImpJixie
		if err := json.Unmarshal(bidderExt.Bidder, &jxExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("ignoring imp id=%s, invalid ImpExt", imp.ID)},
			}
		}
	}

	data, err := json.Marshal(request)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Error in packaging request to JSON"),
		}}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	if request.Device != nil {
		addHeaderIfNonEmpty(headers, "User-Agent", request.Device.UA)
		addHeaderIfNonEmpty(headers, "X-Forwarded-For", request.Device.IP)
		addHeaderIfNonEmpty(headers, "Accept-Language", request.Device.Language)
		if request.Device.DNT != nil {
			addHeaderIfNonEmpty(headers, "DNT", strconv.Itoa(int(*request.Device.DNT)))
		}
	}

	if request.Site != nil {
		addHeaderIfNonEmpty(headers, "Referer", request.Site.Page)
	}

	theurl := buildEndpoint(a.endpoint, a.testing, request.TMax)
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     theurl,
		Body:    data,
		Headers: headers,
	}}, errs
}

func ContainsAny(raw string, keys []string) bool {
	lowerCased := strings.ToLower(raw)
	for i := 0; i < len(keys); i++ {
		if strings.Contains(lowerCased, keys[i]) {
			return true
		}
	}
	return false

}

func getBidType(bidAdm string) openrtb_ext.BidType {
	if bidAdm != "" && ContainsAny(bidAdm, []string{"<?xml", "<vast"}) {
		return openrtb_ext.BidTypeVideo
	}
	return openrtb_ext.BidTypeBanner
}

// MakeBids make the bids for the bid response.
func (a *JixieAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		// no bid response
		return nil, nil
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

	return adsResp, nil

}
