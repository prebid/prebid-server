package adnuntius

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/util/timeutil"
)

type QueryString map[string]string
type adapter struct {
	time      timeutil.Time
	endpoint  string
	extraInfo string
}
type adnAdunit struct {
	AuId       string    `json:"auId"`
	TargetId   string    `json:"targetId"`
	Dimensions [][]int64 `json:"dimensions,omitempty"`
}

type extDeviceAdnuntius struct {
	NoCookies bool `json:"noCookies,omitempty"`
}

type AdUnit struct {
	AuId       string
	TargetId   string
	Html       string
	ResponseId string
	Ads        []struct {
		Bid struct {
			Amount   float64
			Currency string
		}
		DealID          string `json:"dealId,omitempty"`
		AdId            string
		CreativeWidth   string
		CreativeHeight  string
		CreativeId      string
		LineItemId      string
		Html            string
		DestinationUrls map[string]string
	}
}

type AdnResponse struct {
	AdUnits []AdUnit
}
type adnMetaData struct {
	Usi string `json:"usi,omitempty"`
}
type adnRequest struct {
	AdUnits  []adnAdunit `json:"adUnits"`
	MetaData adnMetaData `json:"metaData,omitempty"`
	Context  string      `json:"context,omitempty"`
}

type RequestExt struct {
	Bidder adnAdunit `json:"bidder"`
}

const defaultNetwork = "default"
const defaultSite = "unknown"
const minutesInHour = 60

// Builder builds a new instance of the Adnuntius adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		time:      &timeutil.RealTime{},
		endpoint:  config.Endpoint,
		extraInfo: config.ExtraAdapterInfo,
	}

	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	return a.generateRequests(*request)
}

func setHeaders(ortbRequest openrtb2.BidRequest) http.Header {

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	if ortbRequest.Device != nil {
		if ortbRequest.Device.IP != "" {
			headers.Add("X-Forwarded-For", ortbRequest.Device.IP)
		}
		if ortbRequest.Device.UA != "" {
			headers.Add("user-agent", ortbRequest.Device.UA)
		}
	}
	return headers
}

func makeEndpointUrl(ortbRequest openrtb2.BidRequest, a *adapter, noCookies bool) (string, []error) {
	uri, err := url.Parse(a.endpoint)
	endpointUrl := a.endpoint
	if err != nil {
		return "", []error{fmt.Errorf("failed to parse Adnuntius endpoint: %v", err)}
	}

	gdpr, consent, err := getGDPR(&ortbRequest)
	if err != nil {
		return "", []error{fmt.Errorf("failed to parse Adnuntius endpoint: %v", err)}
	}

	if !noCookies {
		var deviceExt extDeviceAdnuntius
		if ortbRequest.Device != nil && ortbRequest.Device.Ext != nil {
			if err := json.Unmarshal(ortbRequest.Device.Ext, &deviceExt); err != nil {
				return "", []error{fmt.Errorf("failed to parse Adnuntius endpoint: %v", err)}
			}
		}

		if deviceExt.NoCookies {
			noCookies = true
		}

	}

	_, offset := a.time.Now().Zone()
	tzo := -offset / minutesInHour

	q := uri.Query()
	if gdpr != "" {
		endpointUrl = a.extraInfo
		q.Set("gdpr", gdpr)
	}

	if consent != "" {
		q.Set("consentString", consent)
	}

	if noCookies {
		q.Set("noCookies", "true")
	}

	q.Set("tzo", fmt.Sprint(tzo))
	q.Set("format", "json")

	url := endpointUrl + "?" + q.Encode()
	return url, nil
}

func getImpSizes(imp openrtb2.Imp) [][]int64 {

	if len(imp.Banner.Format) > 0 {
		sizes := make([][]int64, len(imp.Banner.Format))
		for i, format := range imp.Banner.Format {
			sizes[i] = []int64{format.W, format.H}
		}

		return sizes
	}

	if imp.Banner.W != nil && imp.Banner.H != nil {
		size := make([][]int64, 1)
		size[0] = []int64{*imp.Banner.W, *imp.Banner.H}
		return size
	}

	return nil
}

/*
Generate the requests to Adnuntius to reduce the amount of requests going out.
*/
func (a *adapter) generateRequests(ortbRequest openrtb2.BidRequest) ([]*adapters.RequestData, []error) {
	var requestData []*adapters.RequestData
	networkAdunitMap := make(map[string][]adnAdunit)
	headers := setHeaders(ortbRequest)
	var noCookies bool = false

	for _, imp := range ortbRequest.Imp {
		if imp.Banner == nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("ignoring imp id=%s, Adnuntius supports only Banner", imp.ID),
			}}
		}
		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Error unmarshalling ExtImpBidder: %s", err.Error()),
			}}
		}

		var adnuntiusExt openrtb_ext.ImpExtAdnunitus
		if err := json.Unmarshal(bidderExt.Bidder, &adnuntiusExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Error unmarshalling ExtImpBmtm: %s", err.Error()),
			}}
		}

		if adnuntiusExt.NoCookies == true {
			noCookies = true
		}

		network := defaultNetwork
		if adnuntiusExt.Network != "" {
			network = adnuntiusExt.Network
		}

		networkAdunitMap[network] = append(
			networkAdunitMap[network],
			adnAdunit{
				AuId:       adnuntiusExt.Auid,
				TargetId:   fmt.Sprintf("%s-%s", adnuntiusExt.Auid, imp.ID),
				Dimensions: getImpSizes(imp),
			})
	}

	endpoint, err := makeEndpointUrl(ortbRequest, a, noCookies)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("failed to parse URL: %s", err),
		}}
	}

	site := defaultSite
	if ortbRequest.Site != nil && ortbRequest.Site.Page != "" {
		site = ortbRequest.Site.Page
	}

	for _, networkAdunits := range networkAdunitMap {

		adnuntiusRequest := adnRequest{
			AdUnits: networkAdunits,
			Context: site,
		}

		ortbUser := ortbRequest.User
		if ortbUser != nil {
			ortbUserId := ortbRequest.User.ID
			if ortbUserId != "" {
				adnuntiusRequest.MetaData.Usi = ortbRequest.User.ID
			}
		}

		adnJson, err := json.Marshal(adnuntiusRequest)
		if err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Error unmarshalling adnuntius request: %s", err.Error()),
			}}
		}

		requestData = append(requestData, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     endpoint,
			Body:    adnJson,
			Headers: headers,
		})

	}

	return requestData, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Status code: %d, Request malformed", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Status code: %d, Something went wrong with your request", response.StatusCode),
		}}
	}

	var adnResponse AdnResponse
	if err := json.Unmarshal(response.Body, &adnResponse); err != nil {
		return nil, []error{err}
	}

	bidResponse, bidErr := generateBidResponse(&adnResponse, request)
	if bidErr != nil {
		return nil, bidErr
	}

	return bidResponse, nil
}

func getGDPR(request *openrtb2.BidRequest) (string, string, error) {

	gdpr := ""
	var extRegs openrtb_ext.ExtRegs
	if request.Regs != nil && request.Regs.Ext != nil {
		if err := json.Unmarshal(request.Regs.Ext, &extRegs); err != nil {
			return "", "", fmt.Errorf("failed to parse ExtRegs in Adnuntius GDPR check: %v", err)
		}
		if extRegs.GDPR != nil && (*extRegs.GDPR == 0 || *extRegs.GDPR == 1) {
			gdpr = strconv.Itoa(int(*extRegs.GDPR))
		}
	}

	consent := ""
	if request.User != nil && request.User.Ext != nil {
		var extUser openrtb_ext.ExtUser
		if err := json.Unmarshal(request.User.Ext, &extUser); err != nil {
			return "", "", fmt.Errorf("failed to parse ExtUser in Adnuntius GDPR check: %v", err)
		}
		consent = extUser.Consent
	}

	return gdpr, consent, nil
}

func generateBidResponse(adnResponse *AdnResponse, request *openrtb2.BidRequest) (*adapters.BidderResponse, []error) {
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(adnResponse.AdUnits))
	var currency string
	adunitMap := map[string]AdUnit{}

	for _, adnRespAdunit := range adnResponse.AdUnits {
		adunitMap[adnRespAdunit.TargetId] = adnRespAdunit
	}

	for i, imp := range request.Imp {

		auId, _, _, err := jsonparser.Get(imp.Ext, "bidder", "auId")
		if err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Error at Bidder auId: %s", err.Error()),
			}}
		}

		targetID := fmt.Sprintf("%s-%s", string(auId), imp.ID)
		adunit := adunitMap[targetID]

		if len(adunit.Ads) > 0 {

			ad := adunit.Ads[0]

			currency = ad.Bid.Currency

			creativeWidth, widthErr := strconv.ParseInt(ad.CreativeWidth, 10, 64)
			if widthErr != nil {
				return nil, []error{&errortypes.BadInput{
					Message: fmt.Sprintf("Value of width: %s is not a string", ad.CreativeWidth),
				}}
			}

			creativeHeight, heightErr := strconv.ParseInt(ad.CreativeHeight, 10, 64)
			if heightErr != nil {
				return nil, []error{&errortypes.BadInput{
					Message: fmt.Sprintf("Value of height: %s is not a string", ad.CreativeHeight),
				}}
			}

			adDomain := []string{}
			for _, url := range ad.DestinationUrls {
				domainArray := strings.Split(url, "/")
				domain := strings.Replace(domainArray[2], "www.", "", -1)
				adDomain = append(adDomain, domain)
			}

			bid := openrtb2.Bid{
				ID:      ad.AdId,
				ImpID:   request.Imp[i].ID,
				W:       creativeWidth,
				H:       creativeHeight,
				AdID:    ad.AdId,
				DealID:  ad.DealID,
				CID:     ad.LineItemId,
				CrID:    ad.CreativeId,
				Price:   ad.Bid.Amount * 1000,
				AdM:     adunit.Html,
				ADomain: adDomain,
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: "banner",
			})
		}

	}
	bidResponse.Currency = currency
	return bidResponse, nil
}
