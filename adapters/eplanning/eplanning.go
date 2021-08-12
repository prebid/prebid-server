package eplanning

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"net/url"
	"strings"

	"regexp"

	"fmt"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"

	"strconv"
)

const nullSize = "1x1"
const defaultPageURL = "FILE"
const sec = "ROS"
const dfpClientID = "1"
const requestTargetInventory = "1"

var priorityOrderForMobileSizesAsc = []string{"1x1", "300x50", "320x50", "300x250"}
var priorityOrderForDesktopSizesAsc = []string{"1x1", "970x90", "970x250", "160x600", "300x600", "728x90", "300x250"}

var cleanNameSteps = []cleanNameStep{
	{regexp.MustCompile(`_|\.|-|\/`), ""},
	{regexp.MustCompile(`\)\(|\(|\)|:`), "_"},
	{regexp.MustCompile(`^_+|_+$`), ""},
}

type cleanNameStep struct {
	expression        *regexp.Regexp
	replacementString string
}

type EPlanningAdapter struct {
	URI     string
	testing bool
}

type hbResponse struct {
	Spaces []hbResponseSpace `json:"sp"`
}

type hbResponseSpace struct {
	Name string         `json:"k"`
	Ads  []hbResponseAd `json:"a"`
}

type hbResponseAd struct {
	ImpressionID string `json:"i"`
	AdID         string `json:"id,omitempty"`
	Price        string `json:"pr"`
	AdM          string `json:"adm"`
	CrID         string `json:"crid"`
	Width        uint64 `json:"w,omitempty"`
	Height       uint64 `json:"h,omitempty"`
}

func (adapter *EPlanningAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errors := make([]error, 0, len(request.Imp))
	totalImps := len(request.Imp)
	spacesStrings := make([]string, 0, totalImps)
	totalRequests := 0
	clientID := ""
	isMobile := isMobileDevice(request)

	for i := 0; i < totalImps; i++ {
		imp := request.Imp[i]
		extImp, err := verifyImp(&imp, isMobile)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		if clientID == "" {
			clientID = extImp.ClientID
		}

		totalRequests++
		// Save valid imp
		name := cleanName(extImp.AdUnitCode)
		spacesStrings = append(spacesStrings, name+":"+extImp.SizeString)
	}

	if totalRequests == 0 {
		return nil, errors
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	headers.Add("Accept", "application/json")
	ip := ""
	if request.Device != nil {
		ip = request.Device.IP
		addHeaderIfNonEmpty(headers, "User-Agent", request.Device.UA)
		addHeaderIfNonEmpty(headers, "X-Forwarded-For", ip)
		addHeaderIfNonEmpty(headers, "Accept-Language", request.Device.Language)
		if request.Device.DNT != nil {
			addHeaderIfNonEmpty(headers, "DNT", strconv.Itoa(int(*request.Device.DNT)))
		}
	}

	pageURL := defaultPageURL
	if request.Site != nil && request.Site.Page != "" {
		pageURL = request.Site.Page
	}

	pageDomain := defaultPageURL
	if request.Site != nil {
		if request.Site.Domain != "" {
			pageDomain = request.Site.Domain
		} else if request.Site.Page != "" {
			u, err := url.Parse(request.Site.Page)
			if err != nil {
				errors = append(errors, err)
				return nil, errors
			}
			pageDomain = u.Hostname()
		}
	}

	requestTarget := pageDomain
	if request.App != nil && request.App.Bundle != "" {
		requestTarget = request.App.Bundle
	}

	uriObj, err := url.Parse(adapter.URI)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	uriObj.Path = uriObj.Path + fmt.Sprintf("/%s/%s/%s/%s", clientID, dfpClientID, requestTarget, sec)
	query := url.Values{}
	query.Set("ncb", "1")
	if request.App == nil {
		query.Set("ur", pageURL)
	}
	query.Set("e", strings.Join(spacesStrings, "+"))

	if request.User != nil && request.User.BuyerUID != "" {
		query.Set("uid", request.User.BuyerUID)
	}

	if ip != "" {
		query.Set("ip", ip)
	}

	var body []byte
	if adapter.testing {
		body = []byte("{}")
	} else {
		t := strconv.Itoa(rand.Int())
		query.Set("rnd", t)
		body = nil
	}

	if request.App != nil {
		if request.App.Name != "" {
			query.Set("appn", request.App.Name)
		}
		if request.App.ID != "" {
			query.Set("appid", request.App.ID)
		}
		if request.Device != nil && request.Device.IFA != "" {
			query.Set("ifa", request.Device.IFA)
		}
		query.Set("app", requestTargetInventory)
	}

	uriObj.RawQuery = query.Encode()
	uri := uriObj.String()

	requestData := adapters.RequestData{
		Method:  "GET",
		Uri:     uri,
		Body:    body,
		Headers: headers,
	}

	requests := []*adapters.RequestData{&requestData}

	return requests, errors
}

func isMobileDevice(request *openrtb2.BidRequest) bool {
	return request.Device != nil && (request.Device.DeviceType == openrtb2.DeviceTypeMobileTablet || request.Device.DeviceType == openrtb2.DeviceTypePhone || request.Device.DeviceType == openrtb2.DeviceTypeTablet)
}

func cleanName(name string) string {
	for _, step := range cleanNameSteps {
		name = step.expression.ReplaceAllString(name, step.replacementString)
	}
	return name
}

func verifyImp(imp *openrtb2.Imp, isMobile bool) (*openrtb_ext.ExtImpEPlanning, error) {
	var bidderExt adapters.ExtImpBidder

	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Ignoring imp id=%s, error while decoding extImpBidder, err: %s", imp.ID, err),
		}
	}

	impExt := openrtb_ext.ExtImpEPlanning{}
	err := json.Unmarshal(bidderExt.Bidder, &impExt)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Ignoring imp id=%s, error while decoding impExt, err: %s", imp.ID, err),
		}
	}

	if impExt.ClientID == "" {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Ignoring imp id=%s, no ClientID present", imp.ID),
		}
	}

	width, height := getSizeFromImp(imp, isMobile)

	if width == 0 && height == 0 {
		impExt.SizeString = nullSize
	} else {
		impExt.SizeString = fmt.Sprintf("%dx%d", width, height)
	}

	if impExt.AdUnitCode == "" {
		impExt.AdUnitCode = impExt.SizeString
	}

	return &impExt, nil
}

func searchSizePriority(hashedFormats map[string]int, format []openrtb2.Format, priorityOrderForSizesAsc []string) (int64, int64) {
	for i := len(priorityOrderForSizesAsc) - 1; i >= 0; i-- {
		if formatIndex, wasFound := hashedFormats[priorityOrderForSizesAsc[i]]; wasFound {
			return format[formatIndex].W, format[formatIndex].H
		}
	}
	return format[0].W, format[0].H
}

func getSizeFromImp(imp *openrtb2.Imp, isMobile bool) (int64, int64) {
	if imp.Banner.W != nil && imp.Banner.H != nil {
		return *imp.Banner.W, *imp.Banner.H
	}

	if imp.Banner.Format != nil {
		hashedFormats := make(map[string]int, len(imp.Banner.Format))

		for i, format := range imp.Banner.Format {
			if format.W != 0 && format.H != 0 {
				hashedFormats[fmt.Sprintf("%dx%d", format.W, format.H)] = i
			}
		}

		if isMobile {
			return searchSizePriority(hashedFormats, imp.Banner.Format, priorityOrderForMobileSizesAsc)
		} else {
			return searchSizePriority(hashedFormats, imp.Banner.Format, priorityOrderForDesktopSizesAsc)
		}
	}

	return 0, 0
}

func addHeaderIfNonEmpty(headers http.Header, headerName string, headerValue string) {
	if len(headerValue) > 0 {
		headers.Add(headerName, headerValue)
	}
}

func (adapter *EPlanningAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var parsedResponse hbResponse
	if err := json.Unmarshal(response.Body, &parsedResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Error unmarshaling HB response: %s", err.Error()),
		}}
	}

	isMobile := isMobileDevice(internalRequest)

	bidResponse := adapters.NewBidderResponse()

	spaceNameToImpID := make(map[string]string)

	for _, imp := range internalRequest.Imp {
		extImp, err := verifyImp(&imp, isMobile)
		if err != nil {
			continue
		}

		name := cleanName(extImp.AdUnitCode)
		spaceNameToImpID[name] = imp.ID
	}

	for _, space := range parsedResponse.Spaces {
		for _, ad := range space.Ads {
			if price, err := strconv.ParseFloat(ad.Price, 64); err == nil {
				bid := openrtb2.Bid{
					ID:    ad.ImpressionID,
					AdID:  ad.AdID,
					ImpID: spaceNameToImpID[space.Name],
					Price: price,
					AdM:   ad.AdM,
					CrID:  ad.CrID,
					W:     int64(ad.Width),
					H:     int64(ad.Height),
				}

				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: openrtb_ext.BidTypeBanner,
				})
			}
		}
	}

	return bidResponse, nil
}

// Builder builds a new instance of the EPlanning adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &EPlanningAdapter{
		URI:     config.Endpoint,
		testing: false,
	}
	return bidder, nil
}
