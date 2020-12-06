package jixie

//
import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

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

func buildEndpoint(endpoint string, testing bool, timeout int64) string {
	if timeout == 0 {
		timeout = 1000
	}
	fmt.Println("!... buildEndpoint...!")
	fmt.Println(endpoint)
	

	if testing {
		// for passing validation tests
		return endpoint + "?t=1000&ts=2060541160"
	}
	return endpoint + "?t=" + strconv.FormatInt(timeout, 10) + "&ts=" + strconv.FormatInt(time.Now().Unix(), 10) + "&src=pbserver"
}

func (a *JixieAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	fmt.Println("!... JIXIE MAKE REQUESTS...______ len of imp!")
	fmt.Println(len(request.Imp))

	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("No Imps in Bid Request"),
		}}
	}
	fmt.Println("!... JIXIE MAKE REQUESTS...!1")


	if errs := preprocess(request); errs != nil && len(errs) > 0 {
		return nil, append(errs, &errortypes.BadInput{
			Message: fmt.Sprintf("Error in preprocess of Imp, err: %s", errs),
		})
	}
	fmt.Println("!... JIXIE MAKE REQUESTS...!2")

	data, err := json.Marshal(request)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Error in packaging request to JSON"),
		}}
	}
	fmt.Println("!... JIXIE MAKE REQUESTS...!3")

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	fmt.Println("!... JIXIE MAKE REQUESTS...! 4")

	if request.Device != nil {
		addHeaderIfNonEmpty(headers, "User-Agent", request.Device.UA)
		addHeaderIfNonEmpty(headers, "X-Forwarded-For", request.Device.IP)
		addHeaderIfNonEmpty(headers, "Accept-Language", request.Device.Language)
	}
	if request.Site != nil {
		addHeaderIfNonEmpty(headers, "Referer", request.Site.Page)
	}
	fmt.Println("!... JIXIE MAKE REQUESTS...!5 ")


	url := buildEndpoint(a.endpoint, a.testing, request.TMax)
	fmt.Println(url)
	fmt.Println("!... JIXIE MAKE REQUESTS...!6")


	abc := "http://localhost:8080/v2/hbsvrpost"
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     abc,
		//Uri:     url,
		Body:    data,
		Headers: headers,
	}}, errs
}

func unpackImpExt(imp *openrtb.Imp) (*openrtb_ext.ExtImpJixie, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var jixieExt openrtb_ext.ExtImpJixie
	if err := json.Unmarshal(bidderExt.Bidder, &jixieExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, invalid ImpExt", imp.ID),
		}
	}

	tagIDValidation, err := strconv.ParseInt(jixieExt.TagID, 10, 64)
	if err != nil || tagIDValidation == 0 {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, invalid tagid must be a String of numbers", imp.ID),
		}
	}

	if jixieExt.TagID == "" {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Ignoring imp id=%s, no tagid present", imp.ID),
		}
	}

	if jixieExt.Unit == "" {
		return nil, &errortypes.BadInput{
			Message: "unit is not set",
		}
	}

	fmt.Printf("func unpackImpExt %+v\n", jixieExt)
	fmt.Printf("-------------------------------------\n");

	return &jixieExt, nil
}

func buildImpBanner(imp *openrtb.Imp) error {

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

func buildImpVideo(imp *openrtb.Imp) error {

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

	if imp.Video.Protocols != nil {
		imp.Video.Protocols = cleanProtocol(imp.Video.Protocols)
	}

	return nil
}

// not supporting VAST protocol 7 (VAST 4.0);
func cleanProtocol(protocols []openrtb.Protocol) []openrtb.Protocol {
	newitems := make([]openrtb.Protocol, 0, len(protocols))

	for _, i := range protocols {
		if i != openrtb.ProtocolVAST40 {
			newitems = append(newitems, i)
		}
	}

	return newitems
}

// Add Jixie required properties to Imp object
func addImpProps(imp *openrtb.Imp, secure *int8, jixieExt *openrtb_ext.ExtImpJixie) {
	imp.TagID = jixieExt.Unit
	// imp.Unit = jixieExt.Unit
	imp.Secure = secure

	if jixieExt.BidFloor != "" {
		bidFloor, err := strconv.ParseFloat(jixieExt.BidFloor, 64)
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

// Handle request errors and formatting to be sent to Jixie
func preprocess(request *openrtb.BidRequest) []error {
	impsCount := len(request.Imp)
	errors := make([]error, 0, impsCount)
	resImps := make([]openrtb.Imp, 0, impsCount)
	secure := int8(0)
	domain := ""
	if request.Site != nil && request.Site.Page != "" {
		domain = request.Site.Page
	} //else if request.App != nil {
		//if request.App.Domain != "" {
		//	domain = request.App.Domain
		//} else if request.App.StoreURL != "" {
		//	domain = request.App.StoreURL
		//}
	//}

	pageURL, err := url.Parse(domain)
	if err == nil && pageURL.Scheme == "https" {
		secure = int8(1)
	}

	for _, imp := range request.Imp {
		jixieExt, err := unpackImpExt(&imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		addImpProps(&imp, &secure, jixieExt)

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
func (a *JixieAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	fmt.Println("!... JIXIE MakeBids...!")

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

// NOTE: Builder builds a new instance of the Jixie adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	fmt.Println("!... JIXIE Builder...!")

	bidder := &JixieAdapter{
		endpoint: config.Endpoint,
		testing:  false,
	}
	return bidder, nil
}
