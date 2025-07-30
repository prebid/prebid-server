package eplanning

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"fmt"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"

	"strconv"
)

const nullSize = "1x1"
const defaultPageURL = "FILE"
const sec = "ROS"
const dfpClientID = "1"
const requestTargetInventory = "1"
const vastInstream = 1
const vastOutstream = 2
const vastVersionDefault = "3"
const vastDefaultSize = "640x480"
const impTypeBanner = 0

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
	impType := getImpTypeRequest(request, totalImps)
	index_vast := 0

	for i := 0; i < totalImps; i++ {
		imp := request.Imp[i]
		extImp, err := verifyImp(&imp, isMobile, impType)
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
		if imp.Video != nil {
			name = getNameVideo(extImp.SizeString, index_vast)
			spacesStrings = append(spacesStrings, name+":"+extImp.SizeString+";1")
			index_vast++
		} else {
			spacesStrings = append(spacesStrings, name+":"+extImp.SizeString)
		}

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

	if impType > 0 {
		query.Set("vctx", strconv.Itoa(impType))
		query.Set("vv", vastVersionDefault)
	}
	if request.Source != nil && request.Source.Ext != nil {
		err := setSchain(request.Source.Ext, &query)
		if err != nil {
			errors = append(errors, err)
			return nil, errors
		}
	}

	uriObj.RawQuery = query.Encode()
	uri := uriObj.String()

	requestData := adapters.RequestData{
		Method:  "GET",
		Uri:     uri,
		Body:    body,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	requests := []*adapters.RequestData{&requestData}

	return requests, errors
}

func setSchain(ext json.RawMessage, query *url.Values) error {
	openRtbSchain, err := unmarshalSupplyChain(ext)
	if err != nil {
		return err
	}
	if openRtbSchain == nil || len(openRtbSchain.Nodes) > 2 {
		return nil
	}

	schainValue, err := makeSupplyChain(*openRtbSchain)
	if err != nil {
		return err
	}

	if schainValue != "" {
		query.Set("sch", schainValue)
	}

	return nil
}

func unmarshalSupplyChain(ext json.RawMessage) (*openrtb2.SupplyChain, error) {
	var extSChain openrtb_ext.ExtRequestPrebidSChain
	err := jsonutil.Unmarshal(ext, &extSChain)
	if err != nil {
		return nil, err
	}
	return &extSChain.SChain, nil
}

func makeSupplyChain(openRtbSchain openrtb2.SupplyChain) (string, error) {
	if len(openRtbSchain.Nodes) == 0 {
		return "", nil
	}

	const schainPrefixFmt = "%s,%d"
	const schainNodeFmt = "!%s,%s,%s,%s,%s,%s,%s"
	schainPrefix := fmt.Sprintf(schainPrefixFmt, openRtbSchain.Ver, openRtbSchain.Complete)
	var sb strings.Builder
	sb.WriteString(schainPrefix)
	for _, node := range openRtbSchain.Nodes {
		nodeValues := []any{
			node.ASI, node.SID, node.HP, node.RID, node.Name, node.Domain, node.Ext,
		}
		formattedValues, err := formatNodeValues(nodeValues)
		if err != nil {
			return "", err
		}

		schainNode := fmt.Sprintf(schainNodeFmt, formattedValues...)
		sb.WriteString(schainNode)
	}

	return sb.String(), nil
}

func formatNodeValues(nodeValues []any) ([]any, error) {
	var formattedValues []any
	for _, value := range nodeValues {
		formattedValue, err := makeNodeValue(value)
		if err != nil {
			return nil, err
		}
		formattedValues = append(formattedValues, formattedValue)
	}
	return formattedValues, nil
}

func makeNodeValue(nodeParam any) (string, error) {
	switch nodeParam := nodeParam.(type) {
	case string:
		// url.QueryEscape() follows the application/x-www-form-urlencoded convention, which encodes spaces as + and RFC 3986 encodes as %20
		return strings.ReplaceAll(url.QueryEscape(nodeParam), "+", "%20"), nil
	case *int8:
		pointer := nodeParam
		if pointer == nil {
			return "", nil
		}
		return makeNodeValue(int(*pointer))
	case int:
		return strconv.Itoa(nodeParam), nil
	case json.RawMessage:
		if nodeParam != nil {
			freeFormJson, err := json.Marshal(nodeParam)
			if err != nil {
				return "", err
			}
			return makeNodeValue(string(freeFormJson))
		}
		return "", nil
	default:
		return "", nil
	}
}

func isMobileDevice(request *openrtb2.BidRequest) bool {
	return request.Device != nil && (request.Device.DeviceType == adcom1.DeviceMobile || request.Device.DeviceType == adcom1.DevicePhone || request.Device.DeviceType == adcom1.DeviceTablet)
}

func getImpTypeRequest(request *openrtb2.BidRequest, totalImps int) int {

	impType := impTypeBanner
	for i := 0; i < totalImps; i++ {
		imp := request.Imp[i]
		if imp.Video != nil {
			if imp.Video.Placement == vastInstream {
				impType = vastInstream
			} else if impType == impTypeBanner {
				impType = vastOutstream
			}
		}
	}

	return impType

}
func cleanName(name string) string {
	for _, step := range cleanNameSteps {
		name = step.expression.ReplaceAllString(name, step.replacementString)
	}
	return name
}

func verifyImp(imp *openrtb2.Imp, isMobile bool, impType int) (*openrtb_ext.ExtImpEPlanning, error) {
	var bidderExt adapters.ExtImpBidder

	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Ignoring imp id=%s, error while decoding extImpBidder, err: %s", imp.ID, err),
		}
	}

	if impType > impTypeBanner {
		if impType == vastInstream {
			// In-stream
			if imp.Video == nil || imp.Video.Placement != vastInstream {
				return nil, &errortypes.BadInput{
					Message: fmt.Sprintf("Ignoring imp id=%s, auction instream and imp no instream", imp.ID),
				}
			}
		} else {
			//Out-stream
			if imp.Video == nil || imp.Video.Placement == vastInstream {
				return nil, &errortypes.BadInput{
					Message: fmt.Sprintf("Ignoring imp id=%s, auction outstream and imp no outstream", imp.ID),
				}
			}
		}
	}

	impExt := openrtb_ext.ExtImpEPlanning{}
	err := jsonutil.Unmarshal(bidderExt.Bidder, &impExt)
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
		if imp.Video != nil {
			impExt.SizeString = vastDefaultSize
		} else {
			impExt.SizeString = nullSize
		}
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

	if imp.Video != nil && imp.Video.W != nil && *imp.Video.W > 0 && imp.Video.H != nil && *imp.Video.H > 0 {
		return *imp.Video.W, *imp.Video.H
	}

	if imp.Banner != nil {
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
	if err := jsonutil.Unmarshal(response.Body, &parsedResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Error unmarshaling HB response: %s", err.Error()),
		}}
	}

	isMobile := isMobileDevice(internalRequest)
	impType := getImpTypeRequest(internalRequest, len(internalRequest.Imp))

	bidResponse := adapters.NewBidderResponse()

	spaceNameToImpID := make(map[string]string)

	index_vast := 0
	for _, imp := range internalRequest.Imp {
		extImp, err := verifyImp(&imp, isMobile, impType)
		if err != nil {
			continue
		}

		name := cleanName(extImp.AdUnitCode)
		if imp.Video != nil {
			name = getNameVideo(extImp.SizeString, index_vast)
			index_vast++
		}
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
					BidType: getBidType(impType),
				})
			}
		}
	}

	return bidResponse, nil
}

func getBidType(impType int) openrtb_ext.BidType {

	bidType := openrtb_ext.BidTypeBanner
	if impType > 0 {
		bidType = openrtb_ext.BidTypeVideo
	}
	return bidType
}

func getNameVideo(size string, index_vast int) string {
	return "video_" + size + "_" + strconv.Itoa(index_vast)
}

// Builder builds a new instance of the EPlanning adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &EPlanningAdapter{
		URI:     config.Endpoint,
		testing: false,
	}
	return bidder, nil
}
