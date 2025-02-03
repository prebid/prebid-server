package adnuntius

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/timeutil"
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
	MaxDeals   int       `json:"maxDeals,omitempty"`
}

type extDeviceAdnuntius struct {
	NoCookies bool `json:"noCookies,omitempty"`
}
type siteExt struct {
	Data interface{} `json:"data"`
}

type adnAdvertiser struct {
	LegalName string `json:"legalName,omitempty"`
	Name      string `json:"name,omitempty"`
}

type Ad struct {
	Bid struct {
		Amount   float64
		Currency string
	}
	NetBid struct {
		Amount float64
	}
	GrossBid struct {
		Amount float64
	}
	DealID          string `json:"dealId,omitempty"`
	AdId            string
	CreativeWidth   string
	CreativeHeight  string
	CreativeId      string
	LineItemId      string
	Html            string
	DestinationUrls map[string]string
	Advertiser      adnAdvertiser `json:"advertiser,omitempty"`
}

type AdUnit struct {
	AuId       string
	TargetId   string
	Html       string
	ResponseId string
	Ads        []Ad
	Deals      []Ad `json:"deals,omitempty"`
}

type AdnResponse struct {
	AdUnits []AdUnit
}
type adnMetaData struct {
	Usi string `json:"usi,omitempty"`
}
type adnRequest struct {
	AdUnits   []adnAdunit `json:"adUnits"`
	MetaData  adnMetaData `json:"metaData,omitempty"`
	Context   string      `json:"context,omitempty"`
	KeyValues interface{} `json:"kv,omitempty"`
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
			if err := jsonutil.Unmarshal(ortbRequest.Device.Ext, &deviceExt); err != nil {
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
	q.Set("format", "prebidServer")

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
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Error unmarshalling ExtImpBidder: %s", err.Error()),
			}}
		}

		var adnuntiusExt openrtb_ext.ImpExtAdnunitus
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &adnuntiusExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Error unmarshalling ExtImpValues: %s", err.Error()),
			}}
		}

		if adnuntiusExt.NoCookies {
			noCookies = true
		}

		network := defaultNetwork
		if adnuntiusExt.Network != "" {
			network = adnuntiusExt.Network
		}

		adUnit := adnAdunit{
			AuId:       adnuntiusExt.Auid,
			TargetId:   fmt.Sprintf("%s-%s", adnuntiusExt.Auid, imp.ID),
			Dimensions: getImpSizes(imp),
		}
		if adnuntiusExt.MaxDeals > 0 {
			adUnit.MaxDeals = adnuntiusExt.MaxDeals
		}
		networkAdunitMap[network] = append(
			networkAdunitMap[network],
			adUnit)
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

	extSite, erro := getSiteExtAsKv(&ortbRequest)
	if erro != nil {
		return nil, []error{fmt.Errorf("failed to parse site Ext: %v", err)}
	}

	for _, networkAdunits := range networkAdunitMap {

		adnuntiusRequest := adnRequest{
			AdUnits:   networkAdunits,
			Context:   site,
			KeyValues: extSite.Data,
		}

		var extUser openrtb_ext.ExtUser
		if ortbRequest.User != nil && ortbRequest.User.Ext != nil {
			if err := jsonutil.Unmarshal(ortbRequest.User.Ext, &extUser); err != nil {
				return nil, []error{fmt.Errorf("failed to parse Ext User: %v", err)}
			}
		}

		// Will change when our adserver can accept multiple user IDS
		if extUser.Eids != nil && len(extUser.Eids) > 0 {
			if len(extUser.Eids[0].UIDs) > 0 {
				adnuntiusRequest.MetaData.Usi = extUser.Eids[0].UIDs[0].ID
			}
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
			ImpIDs:  openrtb_ext.GetImpIDs(ortbRequest.Imp),
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
	if err := jsonutil.Unmarshal(response.Body, &adnResponse); err != nil {
		return nil, []error{err}
	}

	bidResponse, bidErr := generateBidResponse(&adnResponse, request)
	if bidErr != nil {
		return nil, bidErr
	}

	return bidResponse, nil
}

func getSiteExtAsKv(request *openrtb2.BidRequest) (siteExt, error) {
	var extSite siteExt
	if request.Site != nil && request.Site.Ext != nil {
		if err := jsonutil.Unmarshal(request.Site.Ext, &extSite); err != nil {
			return extSite, fmt.Errorf("failed to parse ExtSite in Adnuntius: %v", err)
		}
	}
	return extSite, nil
}

func getGDPR(request *openrtb2.BidRequest) (string, string, error) {

	gdpr := ""
	var extRegs openrtb_ext.ExtRegs
	if request.Regs != nil && request.Regs.Ext != nil {
		if err := jsonutil.Unmarshal(request.Regs.Ext, &extRegs); err != nil {
			return "", "", fmt.Errorf("failed to parse ExtRegs in Adnuntius GDPR check: %v", err)
		}
		if extRegs.GDPR != nil && (*extRegs.GDPR == 0 || *extRegs.GDPR == 1) {
			gdpr = strconv.Itoa(int(*extRegs.GDPR))
		}
	}

	consent := ""
	if request.User != nil && request.User.Ext != nil {
		var extUser openrtb_ext.ExtUser
		if err := jsonutil.Unmarshal(request.User.Ext, &extUser); err != nil {
			return "", "", fmt.Errorf("failed to parse ExtUser in Adnuntius GDPR check: %v", err)
		}
		consent = extUser.Consent
	}

	return gdpr, consent, nil
}

func generateReturnExt(ad Ad, request *openrtb2.BidRequest) (json.RawMessage, error) {
	// We always force the publisher to render
	var adRender int8 = 0

	var requestRegsExt *openrtb_ext.ExtRegs
	if request.Regs != nil && request.Regs.Ext != nil {
		if err := jsonutil.Unmarshal(request.Regs.Ext, &requestRegsExt); err != nil {

			return nil, fmt.Errorf("Failed to parse Ext information in Adnuntius: %v", err)
		}
	}

	if ad.Advertiser.Name != "" && requestRegsExt != nil && requestRegsExt.DSA != nil {
		legalName := ad.Advertiser.Name
		if ad.Advertiser.LegalName != "" {
			legalName = ad.Advertiser.LegalName
		}
		ext := &openrtb_ext.ExtBid{
			DSA: &openrtb_ext.ExtBidDSA{
				AdRender: &adRender,
				Paid:     legalName,
				Behalf:   legalName,
			},
		}
		returnExt, err := json.Marshal(ext)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse Ext information in Adnuntius: %v", err)
		}

		return returnExt, nil
	}
	return nil, nil
}

func generateAdResponse(ad Ad, imp openrtb2.Imp, html string, request *openrtb2.BidRequest) (*openrtb2.Bid, []error) {

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

	price := ad.Bid.Amount

	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Error unmarshalling ExtImpBidder: %s", err.Error()),
		}}
	}

	var adnuntiusExt openrtb_ext.ImpExtAdnunitus
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &adnuntiusExt); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Error unmarshalling ExtImpValues: %s", err.Error()),
		}}
	}

	if adnuntiusExt.BidType != "" {
		if strings.EqualFold(string(adnuntiusExt.BidType), "net") {
			price = ad.NetBid.Amount
		}
		if strings.EqualFold(string(adnuntiusExt.BidType), "gross") {
			price = ad.GrossBid.Amount
		}
	}

	extJson, err := generateReturnExt(ad, request)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Error extracting Ext: %s", err.Error()),
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
		ImpID:   imp.ID,
		W:       creativeWidth,
		H:       creativeHeight,
		AdID:    ad.AdId,
		DealID:  ad.DealID,
		CID:     ad.LineItemId,
		CrID:    ad.CreativeId,
		Price:   price * 1000,
		AdM:     html,
		ADomain: adDomain,
		Ext:     extJson,
	}
	return &bid, nil

}

func generateBidResponse(adnResponse *AdnResponse, request *openrtb2.BidRequest) (*adapters.BidderResponse, []error) {
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(adnResponse.AdUnits))
	var currency string
	adunitMap := map[string]AdUnit{}

	for _, adnRespAdunit := range adnResponse.AdUnits {
		adunitMap[adnRespAdunit.TargetId] = adnRespAdunit
	}

	for _, imp := range request.Imp {

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

			adBid, err := generateAdResponse(ad, imp, adunit.Html, request)
			if err != nil {
				return nil, []error{&errortypes.BadInput{
					Message: "Error at ad generation",
				}}
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     adBid,
				BidType: "banner",
			})

			for _, deal := range adunit.Deals {
				dealBid, err := generateAdResponse(deal, imp, deal.Html, request)
				if err != nil {
					return nil, []error{&errortypes.BadInput{
						Message: "Error at ad generation",
					}}
				}

				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     dealBid,
					BidType: "banner",
				})
			}

		}

	}
	bidResponse.Currency = currency
	return bidResponse, nil
}
