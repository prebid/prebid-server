package eplanning

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"net/url"
	"strings"

	"regexp"

	"fmt"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"

	"strconv"
)

const nullSize = "1x1"
const defaultPageURL = "FILE"
const sec = "ROS"
const dfpClientID = "1"
const requestTargetInventory = "1"

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
	http    *adapters.HTTPAdapter
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

func (adapter *EPlanningAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errors := make([]error, 0, len(request.Imp))
	totalImps := len(request.Imp)
	spacesStrings := make([]string, 0, totalImps)
	totalRequests := 0
	clientID := ""

	for i := 0; i < totalImps; i++ {
		imp := request.Imp[i]
		extImp, err := verifyImp(&imp)
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

	var pageURL string
	if request.Site != nil && request.Site.Page != "" {
		pageURL = request.Site.Page
	} else {
		pageURL = defaultPageURL
	}

	var pageDomain string
	if request.Site != nil && request.Site.Domain != "" {
		pageDomain = request.Site.Domain
	} else {
		pageDomain = defaultPageURL
	}

	var requestTarget string
	if request.App != nil && request.App.Bundle != "" {
		requestTarget = request.App.Bundle
	} else {
		requestTarget = pageDomain
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

func cleanName(name string) string {
	for _, step := range cleanNameSteps {
		name = step.expression.ReplaceAllString(name, step.replacementString)
	}
	return name
}

func verifyImp(imp *openrtb.Imp) (*openrtb_ext.ExtImpEPlanning, error) {
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

	width, height := getSizeFromImp(imp)

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

func getSizeFromImp(imp *openrtb.Imp) (uint64, uint64) {
	if imp.Banner.W != nil && imp.Banner.H != nil {
		return *imp.Banner.W, *imp.Banner.H
	}

	if imp.Banner.Format != nil {
		for _, format := range imp.Banner.Format {
			if format.W != 0 && format.H != 0 {
				return format.W, format.H
			}
		}
	}

	return 0, 0
}

func addHeaderIfNonEmpty(headers http.Header, headerName string, headerValue string) {
	if len(headerValue) > 0 {
		headers.Add(headerName, headerValue)
	}
}

func (adapter *EPlanningAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	bidResponse := adapters.NewBidderResponse()

	spaceNameToImpID := make(map[string]string)

	for _, imp := range internalRequest.Imp {
		extImp, err := verifyImp(&imp)
		if err != nil {
			continue
		}

		name := cleanName(extImp.AdUnitCode)
		spaceNameToImpID[name] = imp.ID
	}

	for _, space := range parsedResponse.Spaces {
		for _, ad := range space.Ads {
			if price, err := strconv.ParseFloat(ad.Price, 64); err == nil {
				bid := openrtb.Bid{
					ID:    ad.ImpressionID,
					AdID:  ad.AdID,
					ImpID: spaceNameToImpID[space.Name],
					Price: price,
					AdM:   ad.AdM,
					CrID:  ad.CrID,
					W:     ad.Width,
					H:     ad.Height,
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

func NewEPlanningBidder(client *http.Client, endpoint string) *EPlanningAdapter {
	adapter := &adapters.HTTPAdapter{Client: client}

	return &EPlanningAdapter{
		http:    adapter,
		URI:     endpoint,
		testing: false,
	}
}
