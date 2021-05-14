package emx_digital

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type EmxDigitalAdapter struct {
	endpoint string
	testing  bool
}

func buildEndpoint(endpoint string, testing bool, timeout int64) string {
	if timeout == 0 {
		timeout = 1000
	}
	if testing {
		// for passing validation tests
		return endpoint + "?t=1000&ts=2060541160"
	}
	return endpoint + "?t=" + strconv.FormatInt(timeout, 10) + "&ts=" + strconv.FormatInt(time.Now().Unix(), 10) + "&src=pbserver"
}

func (a *EmxDigitalAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("No Imps in Bid Request"),
		}}
	}

	if errs := preprocess(request); errs != nil && len(errs) > 0 {
		return nil, append(errs, &errortypes.BadInput{
			Message: fmt.Sprintf("Error in preprocess of Imp, err: %s", errs),
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

	url := buildEndpoint(a.endpoint, a.testing, request.TMax)

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     url,
		Body:    data,
		Headers: headers,
	}}, errs
}

func unpackImpExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpEmxDigital, error) {
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

func buildImpBanner(imp *openrtb2.Imp) error {

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

func buildImpVideo(imp *openrtb2.Imp) error {

	if len(imp.Video.MIMEs) == 0 {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Video: missing required field mimes"),
		}
	}

	if imp.Video.H == 0 && imp.Video.W == 0 {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Video: Need at least one size to build request"),
		}
	}

	if len(imp.Video.Protocols) > 0 {
		videoCopy := *imp.Video
		videoCopy.Protocols = cleanProtocol(imp.Video.Protocols)
		imp.Video = &videoCopy
	}

	return nil
}

// not supporting VAST protocol 7 (VAST 4.0);
func cleanProtocol(protocols []openrtb2.Protocol) []openrtb2.Protocol {
	newitems := make([]openrtb2.Protocol, 0, len(protocols))

	for _, i := range protocols {
		if i != openrtb2.ProtocolVAST40 {
			newitems = append(newitems, i)
		}
	}

	return newitems
}

// Add EMX required properties to Imp object
func addImpProps(imp *openrtb2.Imp, secure *int8, emxExt *openrtb_ext.ExtImpEmxDigital) {
	imp.TagID = emxExt.TagID
	imp.Secure = secure

	if emxExt.BidFloor != "" {
		bidFloor, err := strconv.ParseFloat(emxExt.BidFloor, 64)
		if err != nil {
			bidFloor = 0
		}

		if bidFloor > 0 {
			imp.BidFloor = bidFloor
			imp.BidFloorCur = "USD"
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

// Handle request errors and formatting to be sent to EMX
func preprocess(request *openrtb2.BidRequest) []error {
	impsCount := len(request.Imp)
	errors := make([]error, 0, impsCount)
	resImps := make([]openrtb2.Imp, 0, impsCount)
	secure := int8(0)
	domain := ""
	if request.Site != nil && request.Site.Page != "" {
		domain = request.Site.Page
	} else if request.App != nil {
		if request.App.Domain != "" {
			domain = request.App.Domain
		} else if request.App.StoreURL != "" {
			domain = request.App.StoreURL
		}
	}

	pageURL, err := url.Parse(domain)
	if err == nil && pageURL.Scheme == "https" {
		secure = int8(1)
	}

	for _, imp := range request.Imp {
		emxExt, err := unpackImpExt(&imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		addImpProps(&imp, &secure, emxExt)

		if imp.Video != nil {
			if err := buildImpVideo(&imp); err != nil {
				errors = append(errors, err)
				continue
			}
		} else if err := buildImpBanner(&imp); err != nil {
			errors = append(errors, err)
			continue

		}

		resImps = append(resImps, imp)
	}

	request.Imp = resImps

	return errors
}

// MakeBids make the bids for the bid response.
func (a *EmxDigitalAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

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
				BidType: getBidType(sb.Bid[i].AdM),
			})
		}
	}

	return bidResponse, nil

}

func getBidType(bidAdm string) openrtb_ext.BidType {
	if bidAdm != "" && ContainsAny(bidAdm, []string{"<?xml", "<vast"}) {
		return openrtb_ext.BidTypeVideo
	}
	return openrtb_ext.BidTypeBanner
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

// Builder builds a new instance of the EmxDigital adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &EmxDigitalAdapter{
		endpoint: config.Endpoint,
		testing:  false,
	}
	return bidder, nil
}
