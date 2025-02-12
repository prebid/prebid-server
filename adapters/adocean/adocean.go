package adocean

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const adapterVersion = "1.3.0"
const maxUriLength = 8000
const measurementCode = `
	<script>
		+function() {
			var wu = "%s";
			var su = "%s".replace(/\[TIMESTAMP\]/, Date.now());

			if (wu && !(navigator.sendBeacon && navigator.sendBeacon(wu))) {
				(new Image(1,1)).src = wu
			}

			if (su && !(navigator.sendBeacon && navigator.sendBeacon(su))) {
				(new Image(1,1)).src = su
			}
		}();
	</script>
`

type ResponseAdUnit struct {
	ID       string `json:"id"`
	CrID     string `json:"crid"`
	Currency string `json:"currency"`
	Price    string `json:"price"`
	Width    string `json:"width"`
	Height   string `json:"height"`
	Code     string `json:"code"`
	WinURL   string `json:"winUrl"`
	StatsURL string `json:"statsUrl"`
	Error    string `json:"error"`
}

type requestData struct {
	Url        *url.URL
	Headers    *http.Header
	SlaveSizes map[string]string
	ImpIDs     []string
}

// Builder builds a new instance of the AdOcean adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpointTemplate, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, errors.New("Unable to parse endpoint template")
	}

	whiteSpace := regexp.MustCompile(`\s+`)

	bidder := &AdOceanAdapter{
		endpointTemplate: endpointTemplate,
		measurementCode:  whiteSpace.ReplaceAllString(measurementCode, " "),
	}
	return bidder, nil
}

type AdOceanAdapter struct {
	endpointTemplate *template.Template
	measurementCode  string
}

func (a *AdOceanAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No impression in the bid request",
		}}
	}

	consentString := ""
	if request.User != nil {
		var extUser openrtb_ext.ExtUser
		if err := jsonutil.Unmarshal(request.User.Ext, &extUser); err == nil {
			consentString = extUser.Consent
		}
	}

	var reqCreationErrors []error
	var err error
	requestsData := make([]*requestData, 0, len(request.Imp))
	for _, auction := range request.Imp {
		requestsData, err = a.addNewBid(requestsData, &auction, request, consentString)
		if err != nil {
			reqCreationErrors = append(reqCreationErrors, err)
		}
	}

	httpRequests := make([]*adapters.RequestData, 0, len(requestsData))
	for _, requestData := range requestsData {
		httpRequests = append(httpRequests, &adapters.RequestData{
			Method:  "GET",
			Uri:     requestData.Url.String(),
			Headers: *requestData.Headers,
			ImpIDs:  requestData.ImpIDs,
		})
	}

	return httpRequests, reqCreationErrors
}

func (a *AdOceanAdapter) addNewBid(
	requestsData []*requestData,
	imp *openrtb2.Imp,
	request *openrtb2.BidRequest,
	consentString string,
) ([]*requestData, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return requestsData, &errortypes.BadInput{
			Message: "Error parsing bidderExt object",
		}
	}

	var adOceanExt openrtb_ext.ExtImpAdOcean
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &adOceanExt); err != nil {
		return requestsData, &errortypes.BadInput{
			Message: "Error parsing adOceanExt parameters",
		}
	}

	if adOceanExt.EmitterPrefix == "" {
		return requestsData, &errortypes.BadInput{
			Message: "No emitterPrefix param",
		}
	}

	addedToExistingRequest := addToExistingRequest(requestsData, &adOceanExt, imp, (request.Test == 1))
	if addedToExistingRequest {
		return requestsData, nil
	}

	slaveSizes := map[string]string{}
	slaveSizes[adOceanExt.SlaveID] = getImpSizes(imp)

	url, err := a.makeURL(&adOceanExt, imp, request, slaveSizes, consentString)
	if err != nil {
		return requestsData, err
	}

	requestsData = append(requestsData, &requestData{
		Url:        url,
		Headers:    a.formHeaders(request),
		SlaveSizes: slaveSizes,
		ImpIDs:     []string{imp.ID},
	})

	return requestsData, nil
}

func addToExistingRequest(requestsData []*requestData, newParams *openrtb_ext.ExtImpAdOcean, imp *openrtb2.Imp, testImp bool) bool {
	auctionID := imp.ID

	for _, requestData := range requestsData {
		queryParams := requestData.Url.Query()
		masterID := queryParams["id"][0]

		if masterID == newParams.MasterID {
			if _, has := requestData.SlaveSizes[newParams.SlaveID]; has {
				continue
			}

			queryParams.Add("aid", newParams.SlaveID+":"+auctionID)
			requestData.SlaveSizes[newParams.SlaveID] = getImpSizes(imp)
			setSlaveSizesParam(&queryParams, requestData.SlaveSizes, testImp)

			newUrl := *(requestData.Url)
			newUrl.RawQuery = queryParams.Encode()
			if len(newUrl.String()) < maxUriLength {
				requestData.Url = &newUrl
				requestData.ImpIDs = append(requestData.ImpIDs, auctionID)
				return true
			}

			delete(requestData.SlaveSizes, newParams.SlaveID)
		}
	}

	return false
}

func (a *AdOceanAdapter) makeURL(
	params *openrtb_ext.ExtImpAdOcean,
	imp *openrtb2.Imp,
	request *openrtb2.BidRequest,
	slaveSizes map[string]string,
	consentString string,
) (*url.URL, error) {
	endpointParams := macros.EndpointTemplateParams{Host: params.EmitterPrefix}
	host, err := macros.ResolveMacros(a.endpointTemplate, endpointParams)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: "Unable to parse endpoint url template: " + err.Error(),
		}
	}

	endpointURL, err := url.Parse(host)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: "Malformed URL: " + err.Error(),
		}
	}

	randomizedPart := 10000000 + rand.Intn(99999999-10000000)
	if request.Test == 1 {
		randomizedPart = 10000000
	}
	endpointURL.Path = "/_" + strconv.Itoa(randomizedPart) + "/ad.json"

	auctionID := imp.ID
	queryParams := url.Values{}
	queryParams.Add("pbsrv_v", adapterVersion)
	queryParams.Add("id", params.MasterID)
	queryParams.Add("nc", "1")
	queryParams.Add("nosecure", "1")
	queryParams.Add("aid", params.SlaveID+":"+auctionID)
	if consentString != "" {
		queryParams.Add("gdpr_consent", consentString)
		queryParams.Add("gdpr", "1")
	}
	if request.User != nil && request.User.BuyerUID != "" {
		queryParams.Add("hcuserid", request.User.BuyerUID)
	}
	if request.App != nil {
		queryParams.Add("app", "1")
		queryParams.Add("appname", request.App.Name)
		queryParams.Add("appbundle", request.App.Bundle)
		queryParams.Add("appdomain", request.App.Domain)
	}
	if request.Device != nil {
		if request.Device.IFA != "" {
			queryParams.Add("ifa", request.Device.IFA)
		} else {
			queryParams.Add("dpidmd5", request.Device.DPIDMD5)
		}

		queryParams.Add("devos", request.Device.OS)
		queryParams.Add("devosv", request.Device.OSV)
		queryParams.Add("devmodel", request.Device.Model)
		queryParams.Add("devmake", request.Device.Make)
	}

	setSlaveSizesParam(&queryParams, slaveSizes, (request.Test == 1))
	endpointURL.RawQuery = queryParams.Encode()

	return endpointURL, nil
}

func (a *AdOceanAdapter) formHeaders(req *openrtb2.BidRequest) *http.Header {
	headers := make(http.Header)
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	if req.Device != nil {
		headers.Add("User-Agent", req.Device.UA)

		if req.Device.IP != "" {
			headers.Add("X-Forwarded-For", req.Device.IP)
		} else if req.Device.IPv6 != "" {
			headers.Add("X-Forwarded-For", req.Device.IPv6)
		}
	}

	if req.Site != nil {
		headers.Add("Referer", req.Site.Page)
	}

	return &headers
}

func getImpSizes(imp *openrtb2.Imp) string {
	if imp.Banner == nil {
		return ""
	}

	if len(imp.Banner.Format) > 0 {
		sizes := make([]string, len(imp.Banner.Format))
		for i, format := range imp.Banner.Format {
			sizes[i] = strconv.FormatInt(format.W, 10) + "x" + strconv.FormatInt(format.H, 10)
		}

		return strings.Join(sizes, "_")
	}

	if imp.Banner.W != nil && imp.Banner.H != nil {
		return strconv.FormatInt(*imp.Banner.W, 10) + "x" + strconv.FormatInt(*imp.Banner.H, 10)
	}

	return ""
}

func setSlaveSizesParam(queryParams *url.Values, slaveSizes map[string]string, orderByKey bool) {
	sizeValues := make([]string, 0, len(slaveSizes))
	slaveIDs := make([]string, 0, len(slaveSizes))
	for k := range slaveSizes {
		slaveIDs = append(slaveIDs, k)
	}

	if orderByKey {
		sort.Strings(slaveIDs)
	}

	for _, slaveID := range slaveIDs {
		sizes := slaveSizes[slaveID]
		if sizes == "" {
			continue
		}

		rawSlaveID := strings.Replace(slaveID, "adocean", "", 1)
		sizeValues = append(sizeValues, rawSlaveID+"~"+sizes)
	}

	if len(sizeValues) > 0 {
		queryParams.Set("aosspsizes", strings.Join(sizeValues, "-"))
	}
}

func (a *AdOceanAdapter) MakeBids(
	internalRequest *openrtb2.BidRequest,
	externalRequest *adapters.RequestData,
	response *adapters.ResponseData,
) (*adapters.BidderResponse, []error) {
	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d. Network error?", response.StatusCode)}
	}

	requestURL, _ := url.Parse(externalRequest.Uri)
	queryParams := requestURL.Query()
	auctionIDs := queryParams["aid"]

	bidResponses := make([]ResponseAdUnit, 0)
	if err := jsonutil.Unmarshal(response.Body, &bidResponses); err != nil {
		return nil, []error{err}
	}

	var parsedResponses = adapters.NewBidderResponseWithBidsCapacity(len(auctionIDs))
	var parsingErrors []error
	var slaveToAuctionIDMap = make(map[string]string, len(auctionIDs))

	for _, auctionFullID := range auctionIDs {
		auctionIDsSlice := strings.SplitN(auctionFullID, ":", 2)
		slaveToAuctionIDMap[auctionIDsSlice[0]] = auctionIDsSlice[1]
	}

	for _, bid := range bidResponses {
		if auctionID, found := slaveToAuctionIDMap[bid.ID]; found {
			if bid.Error == "true" {
				continue
			}

			price, _ := strconv.ParseFloat(bid.Price, 64)
			width, _ := strconv.ParseInt(bid.Width, 10, 64)
			height, _ := strconv.ParseInt(bid.Height, 10, 64)
			adCode, err := a.prepareAdCodeForBid(bid)
			if err != nil {
				parsingErrors = append(parsingErrors, err)
				continue
			}

			parsedResponses.Bids = append(parsedResponses.Bids, &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					ID:    bid.ID,
					ImpID: auctionID,
					Price: price,
					AdM:   adCode,
					CrID:  bid.CrID,
					W:     width,
					H:     height,
				},
				BidType: openrtb_ext.BidTypeBanner,
			})
			parsedResponses.Currency = bid.Currency
		}
	}

	return parsedResponses, parsingErrors
}

func (a *AdOceanAdapter) prepareAdCodeForBid(bid ResponseAdUnit) (string, error) {
	sspCode, err := url.QueryUnescape(bid.Code)
	if err != nil {
		return "", err
	}

	adCode := fmt.Sprintf(a.measurementCode, bid.WinURL, bid.StatsURL) + sspCode

	return adCode, nil
}
