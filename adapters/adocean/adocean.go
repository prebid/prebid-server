package adocean

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"text/template"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

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
}

func NewAdOceanBidder(client *http.Client, endpointTemplateString string) *AdOceanAdapter {
	a := &adapters.HTTPAdapter{Client: client}
	endpointTemplate, err := template.New("endpointTemplate").Parse(endpointTemplateString)
	if err != nil {
		glog.Fatal("Unable to parse endpoint template")
		return nil
	}

	return &AdOceanAdapter{
		http:             a,
		endpointTemplate: *endpointTemplate,
	}
}

type AdOceanAdapter struct {
	http             *adapters.HTTPAdapter
	endpointTemplate template.Template
}

func (a *AdOceanAdapter) Name() string {
	return "adocean"
}

func (a *AdOceanAdapter) SkipNoCookies() bool {
	return false
}

func (a *AdOceanAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No impression in the bid request",
		}}
	}

	var requests []*adapters.RequestData
	for _, auction := range request.Imp {
		request, err := a.makeRequest(&auction, request)
		if err != nil {
			return nil, []error{err}
		}

		requests = append(requests, request)
	}

	return requests, nil
}

func (a *AdOceanAdapter) makeRequest(imp *openrtb.Imp, request *openrtb.BidRequest) (*adapters.RequestData, error) {
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

	url, err := a.makeURL(&adOceanExt, imp.ID)
	if url == "" {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("User-Agent", request.Device.UA)
	headers.Add("Referer", request.Site.Page)

	if request.Device.IP != "" {
		headers.Add("X-Forwarded-For", request.Device.IP)
	} else if request.Device.IPv6 != "" {
		headers.Add("X-Forwarded-For", request.Device.IPv6)
	}

	return &adapters.RequestData{
		Method:  "GET",
		Uri:     url,
		Headers: headers,
	}, nil
}

func (a *AdOceanAdapter) makeURL(params *openrtb_ext.ExtImpAdOcean, auctionID string) (string, error) {
	if error := validateParams(params); error != nil {
		return "", error
	}

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
	endpointURL.Path = "/_" + strconv.Itoa(randomizedPart) + "/ad.json"

	queryParams := url.Values{}
	queryParams.Add("id", params.MasterID)
	queryParams.Add("nc", "1")
	queryParams.Add("nosecure", "1")
	queryParams.Add("aid", auctionID)
	queryParams.Add("sid", params.SlaveID)
	endpointURL.RawQuery = queryParams.Encode()

	fmt.Println("endpointURL: ", endpointURL.String())

	return endpointURL.String(), nil
}

func validateParams(params *openrtb_ext.ExtImpAdOcean) error {
	if params.EmitterDomain == "" {
		return &errortypes.BadInput{
			Message: "Emitter domain undefined",
		}
	}

	if params.MasterID == "" {
		return &errortypes.BadInput{
			Message: "MasterId undefined",
		}
	}

	if params.SlaveID == "" {
		return &errortypes.BadInput{
			Message: "SlaveId undefined",
		}
	}

	return nil
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
	auctionID := queryParams["aid"][0]
	slaveID := queryParams["sid"][0]

	fmt.Println("Bid for auctionID: ", auctionID, " slaveID: ", slaveID)

	bidResponses := make([]ResponseAdUnit, 0)
	if err := json.Unmarshal(response.Body, &bidResponses); err != nil {
		return nil, []error{err}
	}

	parsedResponses := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, bid := range bidResponses {
		if bid.ID == slaveID {
			price, _ := strconv.ParseFloat(bid.Price, 64)
			width, _ := strconv.ParseUint(bid.Width, 10, 64)
			height, _ := strconv.ParseUint(bid.Height, 10, 64)
			adCode, err := prepareAdCodeForBid(bid)
			if err != nil {
				return nil, []error{err}
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

			break
		}
	}

	return parsedResponses, []error{}
}

func prepareAdCodeForBid(bid ResponseAdUnit) (string, error) {
	sspCode, err := url.QueryUnescape(bid.Code)
	if err != nil {
		return "", err
	}

	adCode := fmt.Sprintf(`
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
    `, bid.WinURL, bid.StatsURL) + sspCode

	return adCode, nil
}
