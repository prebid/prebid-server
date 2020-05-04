package adocean

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const adapterVersion = "1.0.0"
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

func NewAdOceanBidder(client *http.Client, endpointTemplateString string) *AdOceanAdapter {
	a := &adapters.HTTPAdapter{Client: client}
	endpointTemplate, err := template.New("endpointTemplate").Parse(endpointTemplateString)
	if err != nil {
		glog.Fatal("Unable to parse endpoint template")
		return nil
	}

	whiteSpace := regexp.MustCompile(`\s+`)

	return &AdOceanAdapter{
		http:             a,
		endpointTemplate: *endpointTemplate,
		measurementCode:  whiteSpace.ReplaceAllString(measurementCode, " "),
	}
}

type AdOceanAdapter struct {
	http             *adapters.HTTPAdapter
	endpointTemplate template.Template
	measurementCode  string
}

func (a *AdOceanAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No impression in the bid request",
		}}
	}

	consentString := ""
	if request.User != nil {
		var extUser openrtb_ext.ExtUser
		if err := json.Unmarshal(request.User.Ext, &extUser); err == nil {
			consentString = extUser.Consent
		}
	}

	var httpRequests []*adapters.RequestData
	var errors []error

	for _, auction := range request.Imp {
		newHttpRequest, err := a.makeRequest(httpRequests, &auction, request, consentString)
		if err != nil {
			errors = append(errors, err)
		} else if newHttpRequest != nil {
			httpRequests = append(httpRequests, newHttpRequest)
		}
	}

	return httpRequests, errors
}

func (a *AdOceanAdapter) makeRequest(existingRequests []*adapters.RequestData, imp *openrtb.Imp, request *openrtb.BidRequest, consentString string) (*adapters.RequestData, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Error parsing bidderExt object",
		}
	}

	var adOceanExt openrtb_ext.ExtImpAdOcean
	if err := json.Unmarshal(bidderExt.Bidder, &adOceanExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Error parsing adOceanExt parameters",
		}
	}

	addedToExistingRequest := addToExistingRequest(existingRequests, &adOceanExt, imp.ID)
	if addedToExistingRequest {
		return nil, nil
	}

	url, err := a.makeURL(&adOceanExt, imp.ID, request, consentString)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	if request.Device != nil {
		headers.Add("User-Agent", request.Device.UA)

		if request.Device.IP != "" {
			headers.Add("X-Forwarded-For", request.Device.IP)
		} else if request.Device.IPv6 != "" {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}
	}

	if request.Site != nil {
		headers.Add("Referer", request.Site.Page)
	}

	return &adapters.RequestData{
		Method:  "GET",
		Uri:     url,
		Headers: headers,
	}, nil
}

func addToExistingRequest(existingRequests []*adapters.RequestData, newParams *openrtb_ext.ExtImpAdOcean, auctionID string) bool {
requestsLoop:
	for _, request := range existingRequests {
		endpointURL, _ := url.Parse(request.Uri)
		queryParams := endpointURL.Query()
		masterID := queryParams["id"][0]

		if masterID == newParams.MasterID {
			aids := queryParams["aid"]
			for _, aid := range aids {
				slaveID := strings.SplitN(aid, ":", 2)[0]
				if slaveID == newParams.SlaveID {
					continue requestsLoop
				}
			}

			queryParams.Add("aid", newParams.SlaveID+":"+auctionID)
			endpointURL.RawQuery = queryParams.Encode()
			newUri := endpointURL.String()
			if len(newUri) < maxUriLength {
				request.Uri = newUri
				return true
			}
		}
	}

	return false
}

func (a *AdOceanAdapter) makeURL(params *openrtb_ext.ExtImpAdOcean, auctionID string, request *openrtb.BidRequest, consentString string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{Host: params.EmitterDomain}
	host, err := macros.ResolveMacros(a.endpointTemplate, endpointParams)
	if err != nil {
		return "", &errortypes.BadInput{
			Message: "Unable to parse endpoint url template: " + err.Error(),
		}
	}

	endpointURL, err := url.Parse(host)
	if err != nil {
		return "", &errortypes.BadInput{
			Message: "Malformed URL: " + err.Error(),
		}
	}

	randomizedPart := 10000000 + rand.Intn(99999999-10000000)
	if request.Test == 1 {
		randomizedPart = 10000000
	}
	endpointURL.Path = "/_" + strconv.Itoa(randomizedPart) + "/ad.json"

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
	endpointURL.RawQuery = queryParams.Encode()

	return endpointURL.String(), nil
}

func (a *AdOceanAdapter) MakeBids(
	internalRequest *openrtb.BidRequest,
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
	if err := json.Unmarshal(response.Body, &bidResponses); err != nil {
		return nil, []error{err}
	}

	var parsedResponses = adapters.NewBidderResponseWithBidsCapacity(len(auctionIDs))
	var errors []error
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
			width, _ := strconv.ParseUint(bid.Width, 10, 64)
			height, _ := strconv.ParseUint(bid.Height, 10, 64)
			adCode, err := a.prepareAdCodeForBid(bid)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			parsedResponses.Bids = append(parsedResponses.Bids, &adapters.TypedBid{
				Bid: &openrtb.Bid{
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

	return parsedResponses, errors
}

func (a *AdOceanAdapter) prepareAdCodeForBid(bid ResponseAdUnit) (string, error) {
	sspCode, err := url.QueryUnescape(bid.Code)
	if err != nil {
		return "", err
	}

	adCode := fmt.Sprintf(a.measurementCode, bid.WinURL, bid.StatsURL) + sspCode

	return adCode, nil
}
