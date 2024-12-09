package between

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type BetweenAdapter struct {
	EndpointTemplate *template.Template
}

func (a *BetweenAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No valid Imps in Bid Request",
		}}
	}
	ext, errors := preprocess(request)
	if len(errors) > 0 {
		return nil, errors
	}
	endpoint, err := a.buildEndpointURL(ext)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Failed to build endpoint URL: %s", err),
		}}
	}
	data, err := json.Marshal(request)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: "Error in packaging request to JSON",
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
		Uri:     endpoint,
		Body:    data,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, errors
}

func unpackImpExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpBetween, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, invalid BidderExt", imp.ID),
		}
	}

	var betweenExt openrtb_ext.ExtImpBetween
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &betweenExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, invalid ImpExt", imp.ID),
		}
	}

	return &betweenExt, nil
}

func (a *BetweenAdapter) buildEndpointURL(e *openrtb_ext.ExtImpBetween) (string, error) {
	missingRequiredParameterMessage := "required BetweenSSP parameter \"%s\" is missing"
	if e.Host == "" {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf(missingRequiredParameterMessage, "host"),
		}
	}
	if e.PublisherID == "" {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf(missingRequiredParameterMessage, "publisher_id"),
		}
	}
	return macros.ResolveMacros(a.EndpointTemplate, macros.EndpointTemplateParams{Host: e.Host, PublisherID: e.PublisherID})
}

func buildImpBanner(imp *openrtb2.Imp) error {
	if imp.Banner == nil {
		return &errortypes.BadInput{
			Message: "Request needs to include a Banner object",
		}
	}
	banner := *imp.Banner
	if banner.W == nil && banner.H == nil {
		if len(banner.Format) == 0 {
			return &errortypes.BadInput{
				Message: "Need at least one size to build request",
			}
		}
		format := banner.Format[0]
		banner.Format = banner.Format[1:]
		banner.W = &format.W
		banner.H = &format.H
		imp.Banner = &banner
	}

	return nil
}

// Add Between required properties to Imp object
func addImpProps(imp *openrtb2.Imp, secure *int8, betweenExt *openrtb_ext.ExtImpBetween) {
	imp.Secure = secure
}

// Adding header fields to request header
func addHeaderIfNonEmpty(headers http.Header, headerName string, headerValue string) {
	if len(headerValue) > 0 {
		headers.Add(headerName, headerValue)
	}
}

// Handle request errors and formatting to be sent to Between
func preprocess(request *openrtb2.BidRequest) (*openrtb_ext.ExtImpBetween, []error) {
	errors := make([]error, 0, len(request.Imp))
	resImps := make([]openrtb2.Imp, 0, len(request.Imp))
	secure := int8(0)

	if request.Site != nil && request.Site.Page != "" {
		pageURL, err := url.Parse(request.Site.Page)
		if err == nil && pageURL.Scheme == "https" {
			secure = int8(1)
		}
	}

	var betweenExt *openrtb_ext.ExtImpBetween
	for _, imp := range request.Imp {
		var err error
		betweenExt, err = unpackImpExt(&imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		addImpProps(&imp, &secure, betweenExt)

		if err := buildImpBanner(&imp); err != nil {
			errors = append(errors, err)
			continue
		}
		resImps = append(resImps, imp)
	}
	request.Imp = resImps

	return betweenExt, errors
}

func (a *BetweenAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

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
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unable to unpackage bid response. Error %s", err.Error()),
		}}
	}
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: openrtb_ext.BidTypeBanner,
			})
		}
	}

	return bidResponse, nil
}

// Builder builds a new instance of the Between adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := BetweenAdapter{
		EndpointTemplate: template,
	}
	return &bidder, nil
}
