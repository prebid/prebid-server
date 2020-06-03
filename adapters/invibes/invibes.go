package invibes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type InvibesBidResponse struct {
	VideoAdContentResult VideoAdContentResult `json:"videoAdContentResult"`
}

type VideoAdContentResult struct {
	Ads                           []Ad        `json:"Ads"`
	AdReason                      interface{} `json:"AdReason"`
	Log                           string      `json:"Log"`
	PageID                        int64       `json:"PageId"`
	PublisherURLID                int64       `json:"PublisherUrlId"`
	BlockingScript                string      `json:"BlockingScript"`
	FallbackScript                interface{} `json:"FallbackScript"`
	SecondsToWaitForVideoAdScroll interface{} `json:"SecondsToWaitForVideoAdScroll"`
	CmpSettings                   CmpSettings `json:"CmpSettings"`
	LocalizedAdvertiserTitle      string      `json:"LocalizedAdvertiserTitle"`
	MinPercentageForAdview        interface{} `json:"MinPercentageForAdview"`
	StickyCFIDelay                interface{} `json:"StickyCFIDelay"`
	AskGeoInfo                    bool        `json:"AskGeoInfo"`
	ArticlePageURL                interface{} `json:"ArticlePageUrl"`
	TeaserFormattingHTML          interface{} `json:"TeaserFormattingHtml"`
	LanguageCode                  string      `json:"LanguageCode"`
	Zone                          string      `json:"Zone"`
	UserDeviceType                int64       `json:"UserDeviceType"`
	BrokerApis                    []BrokerAPI `json:"BrokerApis"`
	SendAdRequest                 bool        `json:"SendAdRequest"`
	BidModel                      BidModel    `json:"BidModel"`
	VideoAdDisplayOption          string      `json:"VideoAdDisplayOption"`
	AdPlacements                  interface{} `json:"AdPlacements"`
	Scenarios                     interface{} `json:"Scenarios"`
}

type Ad struct {
	VideoExposedID            string         `json:"VideoExposedId"`
	HTMLString                string         `json:"HtmlString"`
	IsTrafficCampaign         bool           `json:"IsTrafficCampaign"`
	Token                     string         `json:"Token"`
	TrackingScript            interface{}    `json:"TrackingScript"`
	OverlayType               string         `json:"OverlayType"`
	Ga                        string         `json:"GA"`
	InvoiceOnBoxOpen          bool           `json:"InvoiceOnBoxOpen"`
	BidPrice                  float64        `json:"BidPrice"`
	BidPriceEUR               float64        `json:"BidPrice_EUR"`
	MinPercentageForAdview    interface{}    `json:"MinPercentageForAdview"`
	VisiElementID             interface{}    `json:"VisiElementId"`
	IABVisiAppliesToEntireAd  bool           `json:"IABVisiAppliesToEntireAd"`
	ElementIABDuration        int64          `json:"ElementIABDuration"`
	ElementIABPercent         int64          `json:"ElementIABPercent"`
	InfeedIABDuration         int64          `json:"InfeedIABDuration"`
	InfeedIABPercent          int64          `json:"InfeedIABPercent"`
	PlayVOnIabSettings        bool           `json:"PlayVOnIabSettings"`
	SendQ0AsStartEvt          bool           `json:"SendQ0AsStartEvt"`
	MinVideoVisiPercentToPlay int64          `json:"MinVideoVisiPercentToPlay"`
	PlayForeverAfterView      interface{}    `json:"PlayForeverAfterView"`
	VisiPercent               interface{}    `json:"VisiPercent"`
	VisiDuration              interface{}    `json:"VisiDuration"`
	ViewCapping               interface{}    `json:"ViewCapping"`
	ClickDelay                int64          `json:"ClickDelay"`
	PlayVAfterC               bool           `json:"PlayVAfterC"`
	SendAdViewOnResponse      bool           `json:"SendAdViewOnResponse"`
	VideoCompletionTime       float64        `json:"VideoCompletionTime"`
	HasInspiredBy             bool           `json:"HasInspiredBy"`
	Sticky                    interface{}    `json:"Sticky"`
	EwebToken                 string         `json:"EwebToken"`
	UData                     interface{}    `json:"UData"`
	DmpScript                 interface{}    `json:"DmpScript"`
	CotargetingScript         interface{}    `json:"CotargetingScript"`
	BvOptVotes                interface{}    `json:"BvOptVotes"`
	COptions                  int64          `json:"COptions"`
	CampaignGeoTag            CampaignGeoTag `json:"CampaignGeoTag"`
	CustomInfo                interface{}    `json:"CustomInfo"`
	HasPopupHTML              bool           `json:"HasPopupHtml"`
	ResourceHintsList         interface{}    `json:"ResourceHintsList"`
}

type CampaignGeoTag struct {
	TagReplacements    TagReplacements `json:"TagReplacements"`
	TagsNotSentToSeweb []interface{}   `json:"TagsNotSentToSeweb"`
	APIKeyInfo         string          `json:"ApiKeyInfo"`
	Locator            Locator         `json:"Locator"`
}

type Locator struct {
	APIKey      string       `json:"ApiKey"`
	Latitude    string       `json:"Latitude"`
	Longitude   string       `json:"Longitude"`
	Coordinates []Coordinate `json:"Coordinates"`
	Zoom        string       `json:"Zoom"`
	MapType     string       `json:"MapType"`
	Color       interface{}  `json:"Color"`
}

type Coordinate struct {
	Latitude      string `json:"Latitude"`
	Longitude     string `json:"Longitude"`
	MarkerName    string `json:"markerName"`
	MarkerWebsite string `json:"markerWebsite"`
}

type TagReplacements struct {
	IvGeoShopAddress string `json:"ivGeoShopAddress"`
	MarkerName       string `json:"markerName"`
	MarkerWebsite    string `json:"markerWebsite"`
}

type BidModel struct {
	PlacementID      string      `json:"PlacementId"`
	CreativeHTML     string      `json:"CreativeHtml"`
	AuctionStartTime int64       `json:"AuctionStartTime"`
	PreloadScripts   interface{} `json:"PreloadScripts"`
	BidVersion       int64       `json:"BidVersion"`
	Width            interface{} `json:"Width"`
	Height           interface{} `json:"Height"`
	Currency         string      `json:"Currency"`
	Context          Context     `json:"Context"`
}

type Context struct {
	Placement        Placement `json:"Placement"`
	AuctionStartTime int64     `json:"AuctionStartTime"`
	BidVersion       int64     `json:"BidVersion"`
	IsBiddingTest    bool      `json:"IsBiddingTest"`
	BidCurrencyID    int64     `json:"BidCurrencyId"`
	BidCurrencyCode  string    `json:"BidCurrencyCode"`
	ExchangeRate     int64     `json:"ExchangeRate"`
}

type Placement struct {
	HeaderBiddingPlacementID int64       `json:"HeaderBiddingPlacementId"`
	PlacementID              string      `json:"PlacementId"`
	PublisherURLID           int64       `json:"PublisherUrlId"`
	TemplateModelID          int64       `json:"TemplateModelId"`
	URL                      interface{} `json:"Url"`
	Width                    interface{} `json:"Width"`
	Height                   interface{} `json:"Height"`
}

type BrokerAPI struct {
	BID          int64       `json:"BId"`
	PID          int64       `json:"PId"`
	CID          int64       `json:"CId"`
	URL          string      `json:"Url"`
	URLNoConsent interface{} `json:"UrlNoConsent"`
	Type         int64       `json:"Type"`
	Script       string      `json:"Script"`
}

type CmpSettings struct {
	AutoOI     bool        `json:"AutoOI"`
	Reason     string      `json:"Reason"`
	ConsentPop interface{} `json:"ConsentPop"`
}

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

type AdInvibesAdapter struct {
	http             *adapters.HTTPAdapter
	endpointTemplate template.Template
	measurementCode  string
}

func NewInvibesBidder(client *http.Client, endpointTemplateString string) *AdInvibesAdapter {
	a := &adapters.HTTPAdapter{Client: client}
	endpointTemplate, err := template.New("endpointTemplate").Parse(endpointTemplateString)
	if err != nil {
		glog.Fatal("Unable to parse endpoint template")
		return nil
	}

	return &AdInvibesAdapter{
		http:             a,
		endpointTemplate: *endpointTemplate,
		//measurementCode:  whiteSpace.ReplaceAllString(measurementCode, " "),
	}
}

func (a *AdInvibesAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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

func (a *AdInvibesAdapter) makeRequest(existingRequests []*adapters.RequestData, imp *openrtb.Imp, request *openrtb.BidRequest, consentString string) (*adapters.RequestData, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Error parsing bidderExt object",
		}
	}

	var invibesExt openrtb_ext.ExtImpInvibes
	if err := json.Unmarshal(bidderExt.Bidder, &invibesExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Error parsing invibesExt parameters",
		}
	}

	//support for multiple imps?
	addedToExistingRequest := addToExistingRequest(existingRequests, &invibesExt, imp.ID)
	if addedToExistingRequest {
		return nil, nil
	}

	url, err := a.makeURL(&invibesExt, imp.ID, request, consentString)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	if request.Device != nil {
		headers.Add("User-Agent", request.Device.UA)

		//todoav: fix this
		// if request.Device.IP != "" {
		// 	headers.Add("X-Forwarded-For", request.Device.IP)
		// } else if request.Device.IPv6 != "" {
		// 	headers.Add("X-Forwarded-For", request.Device.IPv6)
		// }
		headers.Add("X-Forwarded-For", "86.104.183.197")
	}

	if request.Site != nil {
		//todoav: fix this
		// headers.Add("Referer", request.Site.Page)
		headers.Add("Referer", "https://demo.invibesstage.com/qa/infeed.html?videoaddebug=1&invibbvlog=true")
	}

	return &adapters.RequestData{
		Method:  "GET",
		Uri:     url,
		Headers: headers,
	}, nil
}

func addToExistingRequest(existingRequests []*adapters.RequestData, newParams *openrtb_ext.ExtImpInvibes, auctionID string) bool {
	// requestsLoop:
	// 	for _, request := range existingRequests {
	// 		endpointURL, _ := url.Parse(request.Uri)
	// 		queryParams := endpointURL.Query()
	// 		masterID := queryParams["id"][0]

	// 		if masterID == newParams.MasterID {
	// 			aids := queryParams["aid"]
	// 			for _, aid := range aids {
	// 				slaveID := strings.SplitN(aid, ":", 2)[0]
	// 				if slaveID == newParams.SlaveID {
	// 					continue requestsLoop
	// 				}
	// 			}

	// 			queryParams.Add("aid", newParams.SlaveID+":"+auctionID)
	// 			endpointURL.RawQuery = queryParams.Encode()
	// 			newUri := endpointURL.String()
	// 			if len(newUri) < maxUriLength {
	// 				request.Uri = newUri
	// 				return true
	// 			}
	// 		}
	// 	}

	return false
}

func (a *AdInvibesAdapter) makeURL(params *openrtb_ext.ExtImpInvibes, auctionID string, request *openrtb.BidRequest, consentString string) (string, error) {
	//endpointParams := macros.EndpointTemplateParams{Host: params.EmitterDomain}
	endpointParams := macros.EndpointTemplateParams{Host: request.Site.Domain}
	host, err := macros.ResolveMacros(a.endpointTemplate, endpointParams)
	if err != nil {
		return "", &errortypes.BadInput{
			Message: "Unable to parse endpoint url template: " + err.Error(),
		}
	}

	return host, nil

	// endpointURL, err := url.Parse(host)
	// if err != nil {
	// 	return "", &errortypes.BadInput{
	// 		Message: "Malformed URL: " + err.Error(),
	// 	}
	// }

	// randomizedPart := 10000000 + rand.Intn(99999999-10000000)
	// if request.Test == 1 {
	// 	randomizedPart = 10000000
	// }
	// endpointURL.Path = "/_" + strconv.Itoa(randomizedPart) + "/ad.json"

	// queryParams := url.Values{}
	// queryParams.Add("pbsrv_v", adapterVersion)
	// //queryParams.Add("id", params.MasterID)
	// queryParams.Add("nc", "1")
	// queryParams.Add("nosecure", "1")
	// //queryParams.Add("aid", params.SlaveID+":"+auctionID)
	// if consentString != "" {
	// 	queryParams.Add("gdpr_consent", consentString)
	// 	queryParams.Add("gdpr", "1")
	// }
	// if request.User != nil && request.User.BuyerUID != "" {
	// 	queryParams.Add("hcuserid", request.User.BuyerUID)
	// }
	// endpointURL.RawQuery = queryParams.Encode()

	// return endpointURL.String(), nil
}

func (a *AdInvibesAdapter) MakeBids(
	internalRequest *openrtb.BidRequest,
	externalRequest *adapters.RequestData,
	response *adapters.ResponseData,
) (*adapters.BidderResponse, []error) {
	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d. Network error?", response.StatusCode)}
	}

	//requestURL, _ := url.Parse(externalRequest.Uri)
	//queryParams := requestURL.Query()
	//auctionIDs := queryParams["aid"]

	bidResponses := InvibesBidResponse{}

	// sliceBytes := response.Body[20 : len(response.Body)-2]
	// jsonString := string(sliceBytes)
	// println(jsonString)

	if err := json.Unmarshal(response.Body, &bidResponses); err != nil {
		return nil, []error{err}
	}

	var parsedResponses = adapters.NewBidderResponseWithBidsCapacity(len(bidResponses.VideoAdContentResult.Ads))
	var errors []error

	adjson, _ := json.Marshal(bidResponses.VideoAdContentResult)
	adresponse := string(adjson)
	println(string(adresponse))
	// getlinkurl := "https://static.videostepstage.com/desktop/getlink.desktop.js"
	// withScript := "<script id='ivCrHtmlS'>(function () {var i = (top.invibes = top.invibes || {}); i.bidResponse = " + adresponse + ";  })();"
	// withScript = withScript + "(function() { var i = top.invibes = top.invibes || {}; if (i.creativeHtmlRan) { return; } i.creativeHtmlRan = true;  var d = top.document; var e = d.getElementById('divVideoStepAdTop') || d.getElementById('divVideoStepAdTop2') || d.getElementById('divVideoStepAdBottom'); if (e) e.parentNode.removeChild(e); var s = document.getElementById('ivCrHtmlS'); var d = document.createElement('div'); d.setAttribute('id', 'divVideoStepAdTop'); d.className += 'divVideoStep'; s.parentNode.insertBefore(d, s); var j = window.invibes = window.invibes || { }; j.getlinkUrl = '" + getlinkurl + "'; var t = document.createElement('script'); t.src = '" + getlinkurl + "'; s.parentNode.insertBefore(t, s); }()) </script>"
	withScript := "<script id='ivCrHtmlS0'>(function () {var i = (top.invibes = top.invibes || {}); i.bidResponse = " + adresponse + ";  })();</script>"
	withScript = withScript + bidResponses.VideoAdContentResult.BidModel.CreativeHTML

	parsedResponses.Bids = append(parsedResponses.Bids, &adapters.TypedBid{
		Bid: &openrtb.Bid{
			ID:    bidResponses.VideoAdContentResult.Ads[0].VideoExposedID,
			ImpID: "111",
			Price: 1.1, //bidResponses.VideoAdContentResult.Ads[0].BidPrice,
			AdM:   strings.Replace(withScript, "[attrs]", "", -1),
			CrID:  bidResponses.VideoAdContentResult.Ads[0].VideoExposedID,
			W:     320,
			H:     150,
		},
		BidType: openrtb_ext.BidTypeBanner,
	})
	parsedResponses.Currency = bidResponses.VideoAdContentResult.BidModel.Currency

	// var slaveToAuctionIDMap = make(map[string]string, len(auctionIDs))

	// for _, auctionFullID := range auctionIDs {
	// 	auctionIDsSlice := strings.SplitN(auctionFullID, ":", 2)
	// 	slaveToAuctionIDMap[auctionIDsSlice[0]] = auctionIDsSlice[1]
	// }

	// for _, bid := range bidResponses {
	// 	if auctionID, found := slaveToAuctionIDMap[bid.ID]; found {
	// 		if bid.Error == "true" {
	// 			continue
	// 		}

	// 		price, _ := strconv.ParseFloat(bid.Price, 64)
	// 		width, _ := strconv.ParseUint(bid.Width, 10, 64)
	// 		height, _ := strconv.ParseUint(bid.Height, 10, 64)
	// 		adCode, err := a.prepareAdCodeForBid(bid)
	// 		if err != nil {
	// 			errors = append(errors, err)
	// 			continue
	// 		}

	// 		parsedResponses.Bids = append(parsedResponses.Bids, &adapters.TypedBid{
	// 			Bid: &openrtb.Bid{
	// 				ID:    bid.ID,
	// 				ImpID: auctionID,
	// 				Price: price,
	// 				AdM:   adCode,
	// 				CrID:  bid.CrID,
	// 				W:     width,
	// 				H:     height,
	// 			},
	// 			BidType: openrtb_ext.BidTypeBanner,
	// 		})
	// 		parsedResponses.Currency = bid.Currency
	// 	}
	// }

	return parsedResponses, errors
}

func (a *AdInvibesAdapter) prepareAdCodeForBid(bid ResponseAdUnit) (string, error) {
	sspCode, err := url.QueryUnescape(bid.Code)
	if err != nil {
		return "", err
	}

	adCode := fmt.Sprintf(a.measurementCode, bid.WinURL, bid.StatsURL) + sspCode

	return adCode, nil
}
