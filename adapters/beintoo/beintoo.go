package beintoo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type BeintooAdapter struct {
	endpoint string
}

func (a *BeintooAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error

	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("No Imps in Bid Request"),
		}}
	}

	if errors := preprocess(request); errors != nil && len(errors) > 0 {
		return nil, append(errors, &errortypes.BadInput{
			Message: fmt.Sprintf("Error in preprocess of Imp, err: %s", errors),
		})
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

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    data,
		Headers: headers,
	}}, errors
}

func unpackImpExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpBeintoo, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var beintooExt openrtb_ext.ExtImpBeintoo
	if err := json.Unmarshal(bidderExt.Bidder, &beintooExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, invalid ImpExt", imp.ID),
		}
	}

	tagIDValidation, err := strconv.ParseInt(beintooExt.TagID, 10, 64)
	if err != nil || tagIDValidation == 0 {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, invalid tagid must be a String of numbers", imp.ID),
		}
	}

	return &beintooExt, nil
}

func buildImpBanner(imp *openrtb2.Imp) error {
	imp.Ext = nil

	if imp.Banner == nil {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Request needs to include a Banner object"),
		}
	}

	bannerCopy := *imp.Banner
	banner := &bannerCopy

	if banner.W == nil && banner.H == nil {
		if len(banner.Format) == 0 {
			return &errortypes.BadInput{
				Message: fmt.Sprintf("Need at least one size to build request"),
			}
		}
		format := banner.Format[0]
		banner.Format = banner.Format[1:]
		banner.W = &format.W
		banner.H = &format.H
		imp.Banner = banner
	}

	return nil
}

// Add Beintoo required properties to Imp object
func addImpProps(imp *openrtb2.Imp, secure *int8, BeintooExt *openrtb_ext.ExtImpBeintoo) {
	imp.TagID = BeintooExt.TagID
	imp.Secure = secure

	if BeintooExt.BidFloor != "" {
		bidFloor, err := strconv.ParseFloat(BeintooExt.BidFloor, 64)
		if err != nil {
			bidFloor = 0
		}

		if bidFloor > 0 {
			imp.BidFloor = bidFloor
		}
	}

	return
}

// Adding header fields to request header
func addHeaderIfNonEmpty(headers http.Header, headerName string, headerValue string) {
	if len(headerValue) > 0 {
		headers.Add(headerName, headerValue)
	}
}

// Handle request errors and formatting to be sent to Beintoo
func preprocess(request *openrtb2.BidRequest) []error {
	errors := make([]error, 0, len(request.Imp))
	resImps := make([]openrtb2.Imp, 0, len(request.Imp))
	secure := int8(0)

	if request.Site != nil && request.Site.Page != "" {
		pageURL, err := url.Parse(request.Site.Page)
		if err == nil && pageURL.Scheme == "https" {
			secure = int8(1)
		}
	}

	for _, imp := range request.Imp {
		beintooExt, err := unpackImpExt(&imp)
		if err != nil {
			errors = append(errors, err)
			return errors
		}

		addImpProps(&imp, &secure, beintooExt)

		if err := buildImpBanner(&imp); err != nil {
			errors = append(errors, err)
			return errors
		}
		resImps = append(resImps, imp)
	}

	request.Imp = resImps

	return errors
}

// MakeBids make the bids for the bid response.
func (a *BeintooAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		// no bid response
		return nil, nil
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

// Builder builds a new instance of the Beintoo adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &BeintooAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
