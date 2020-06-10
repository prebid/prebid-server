package invibes

import (
	"encoding/json"
	"fmt"
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
	"github.com/prebid/prebid-server/pbsmetrics"
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
	Width            *uint64     `json:"Width"`
	Height           *uint64     `json:"Height"`
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
	Width                    *uint64     `json:"Width"`
	Height                   *uint64     `json:"Height"`
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

type InvibesAdRequest struct {
	Aver          string
	Impid         string
	BidParamsJson string
	Location      string
	Lid           string
	PsbDebug      bool
	Kw            string
	IntegType     int
	Width         string
	Height        string
	AdFormats     []openrtb.Format
	Gdpr_consent  string
	Gdpr          string
	Bvid          string
}

// type InvibesBidParam struct {
// 	PlacementIds []string
// 	BidVersion   string
// }
type AdFormat struct {
	W int
	H int
}

const adapterVersion = "1.0.0"
const maxUriLength = 8000

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
	EndpointTemplate template.Template
}

func NewInvibesBidder(endpointTemplate string) *AdInvibesAdapter {
	urlTemplate, err := template.New("endpointTemplate").Parse(endpointTemplate)
	if err != nil {
		glog.Fatal("Unable to parse endpoint url template")
		return nil
	}
	return &AdInvibesAdapter{EndpointTemplate: *urlTemplate}
}

// func NewInvibesBidder(client *http.Client, endpointTemplateString string) *AdInvibesAdapter {
// 	a := &adapters.HTTPAdapter{Client: client}
// 	endpointTemplate, err := template.New("endpointTemplate").Parse(endpointTemplateString)
// 	if err != nil {
// 		glog.Fatal("Unable to parse endpoint template")
// 		return nil
// 	}

// 	return &AdInvibesAdapter{
// 		http:             a,
// 		endpointTemplate: *endpointTemplate,
// 	}
// }

func (a *AdInvibesAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No imps in the bid request",
		}}
	}

	var isAmp bool
	if reqInfo.PbsEntryPoint == pbsmetrics.ReqTypeAMP {
		isAmp = true
	}

	consentString := ""
	if request.User != nil {
		var extUser openrtb_ext.ExtUser
		if err := json.Unmarshal(request.User.Ext, &extUser); err == nil {
			consentString = extUser.Consent
		}
	}
	gdpr := "1"

	var httpRequests []*adapters.RequestData
	var errors []error

	for _, auction := range request.Imp {
		newHttpRequest, err := a.makeRequest(httpRequests, &auction, request, consentString, gdpr, isAmp)
		if err != nil {
			errors = append(errors, err)
		} else if newHttpRequest != nil {
			httpRequests = append(httpRequests, newHttpRequest)
		}
	}

	return httpRequests, errors
}

func (a *AdInvibesAdapter) makeRequest(existingRequests []*adapters.RequestData, imp *openrtb.Imp, request *openrtb.BidRequest, consentString string, gdpr string, isAmp bool) (*adapters.RequestData, error) {
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

	//support for multiple imps per request?
	// addedToExistingRequest := addToExistingRequest(existingRequests, &invibesExt, imp.ID)
	// if addedToExistingRequest {
	// 	return nil, nil
	// }

	url, err := a.makeURL(request)
	if err != nil {
		return nil, err
	}
	parameter, errp := a.makeParameter(&invibesExt, imp, request, consentString, gdpr, isAmp)
	if errp != nil {
		return nil, errp
	}
	body, errm := json.Marshal(parameter)
	if errm != nil {
		return nil, errm
	}
	println(url)

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("Impid", imp.ID)

	if request.Device != nil {
		headers.Add("User-Agent", request.Device.UA)
	}

	pbsdebug := "false"
	if invibesExt.Debug.TestIp != "" {
		headers.Add("X-Forwarded-For", invibesExt.Debug.TestIp)
		pbsdebug = "true"
	} else if request.Device != nil {
		headers.Add("User-Agent", request.Device.UA)

		if request.Device.IP != "" {
			headers.Add("X-Forwarded-For", request.Device.IP)
		} else if request.Device.IPv6 != "" {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}
	}
	headers.Add("Pbsdebug", pbsdebug)
	if request.Site != nil {
		headers.Add("Referer", request.Site.Page)
	}

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     url,
		Headers: headers,
		Body:    body,
	}, nil
}

func (a *AdInvibesAdapter) makeParameter(params *openrtb_ext.ExtImpInvibes, imp *openrtb.Imp, request *openrtb.BidRequest, consentString string, gdpr string, isAmp bool) (*InvibesAdRequest, error) {
	var lid string
	if request.User != nil {
		if request.User.BuyerUID != "" {
			lid = request.User.BuyerUID
		} else if request.User.ID != "" {
			lid = request.User.ID
		}
	}
	if lid == "" {
		return nil, &errortypes.BadInput{
			Message: "No user id",
		}
	}
	if imp.Banner == nil {
		return nil, &errortypes.BadInput{
			Message: "Banner not specified",
		}
	}
	if request.Site == nil {
		return nil, &errortypes.BadInput{
			Message: "Site not specified",
		}
	}

	currentBanner := *imp.Banner

	integType := 0
	if isAmp {
		integType = 2
	}
	width := ""
	height := ""
	if request.Device != nil {
		if request.Device.W > 0 {
			width = strconv.FormatUint(request.Device.W, 10)
		}

		if request.Device.H > 0 {
			height = strconv.FormatUint(request.Device.H, 10)
		}
	}
	pbsdebug := false
	if params.Debug.TestIp != "" {
		pbsdebug = true
		if width == "" {
			width = "500"
		}
		if height == "" {
			height = "500"
		}
	}

	var adFormats []openrtb.Format
	if currentBanner.Format != nil {
		adFormats = currentBanner.Format
	} else if currentBanner.W != nil && currentBanner.H != nil {
		adFormats = []openrtb.Format{
			{
				W: *currentBanner.W,
				H: *currentBanner.H,
			},
		}
	}

	var invRequest InvibesAdRequest = InvibesAdRequest{
		Aver:          adapterVersion,
		Impid:         imp.ID,
		BidParamsJson: "{placementIds:[\"" + params.PlacementId + "\"],bidVersion:\"1\"}",
		Location:      request.Site.Page,
		Lid:           lid,
		PsbDebug:      pbsdebug,
		Kw:            request.Site.Keywords,
		IntegType:     integType,
		Width:         width,
		Height:        height,
		Gdpr_consent:  consentString,
		Gdpr:          gdpr,
		AdFormats:     adFormats,
		Bvid:          params.Debug.TestBvid,
	}

	return &invRequest, nil
}

func (a *AdInvibesAdapter) makeURL(request *openrtb.BidRequest) (string, error) {
	endpointParams := macros.EndpointTemplateParams{}
	host, err := macros.ResolveMacros(a.EndpointTemplate, endpointParams)
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

	// var lid string
	// if request.User != nil {
	// 	if request.User.BuyerUID != "" {
	// 		lid = request.User.BuyerUID
	// 	} else if request.User.ID != "" {
	// 		lid = request.User.ID
	// 	} else {
	// 		return "", &errortypes.BadInput{
	// 			Message: "No user id",
	// 		}
	// 	}
	// }
	// if imp.Banner == nil {
	// 	return "", &errortypes.BadInput{
	// 		Message: "Banner not specified",
	// 	}
	// }

	//queryParams := url.Values{}
	// queryParams.Add("aver", adapterVersion)
	//queryParams.Add("impid", imp.ID)
	// bidParams := "{placementIds:[\"" + params.PlacementId + "\"],bidVersion:\"1\"}"
	// queryParams.Add("BidParamsJson", bidParams)
	// if request.Site != nil {
	// 	queryParams.Add("location", request.Site.Page)
	// }
	// if lid != "" {
	// 	queryParams.Add("lid", lid)
	// }

	// if params.Debug.TestIp != "" {
	// 	queryParams.Add("pbsdebug", "true")
	// }
	// //queryParams.Add("showFallback", "false")
	// if request.Site.Keywords != "" {
	// 	queryParams.Add("kw", request.Site.Keywords)
	// }
	// if isAmp {
	// 	queryParams.Add("integType", "2")
	// } else {
	// 	queryParams.Add("integType", "0")
	// }
	// if request.Device != nil {
	// 	if request.Device.W > 0 {
	// 		queryParams.Add("width", strconv.FormatUint(request.Device.W, 10))
	// 	} else if params.Debug.TestIp != "" {
	// 		queryParams.Add("width", "600")
	// 	}

	// 	if request.Device.H > 0 {
	// 		queryParams.Add("height", strconv.FormatUint(request.Device.H, 10))
	// 	} else if params.Debug.TestIp != "" {
	// 		queryParams.Add("height", "600")
	// 	}
	// }
	// if imp.Banner != nil {
	// 	// imp.Banner.Format
	// 	// imp.Banner.W
	// 	// imp.Banner.H
	// }

	// if consentString != "" {
	// 	queryParams.Add("gdpr_consent", consentString)
	// 	queryParams.Add("gdpr", "1")
	// }
	//endpointURL.RawQuery = endpointURL.RawQuery + queryParams.Encode()

	return endpointURL.String(), nil
}

func (a *AdInvibesAdapter) MakeBids(
	internalRequest *openrtb.BidRequest,
	externalRequest *adapters.RequestData,
	response *adapters.ResponseData,
) (*adapters.BidderResponse, []error) {
	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d.", response.StatusCode)}
	}

	bidResponses := InvibesBidResponse{}

	currentRequest := *externalRequest
	var impId string = ""
	if len(currentRequest.Headers["Impid"]) > 0 {
		impId = currentRequest.Headers["Impid"][0]
	}
	var pbsdebug bool = false
	if len(currentRequest.Headers["Pbsdebug"]) > 0 {
		pbsdebug = currentRequest.Headers["Pbsdebug"][0] == "true"
	}

	if err := json.Unmarshal(response.Body, &bidResponses); err != nil {
		return nil, []error{err}
	}

	var parsedResponses = adapters.NewBidderResponseWithBidsCapacity(len(bidResponses.VideoAdContentResult.Ads))
	var errors []error

	if bidResponses.VideoAdContentResult.BidModel.Currency != "" {
		parsedResponses.Currency = bidResponses.VideoAdContentResult.BidModel.Currency
	}
	invibesAds := bidResponses.VideoAdContentResult.Ads
	bidResponses.VideoAdContentResult.Ads = nil
	for _, invibesAd := range invibesAds {
		adContentResult := bidResponses.VideoAdContentResult
		adContentResult.Ads = []Ad{invibesAd}

		adjson, _ := json.Marshal(adContentResult)
		adresponse := string(adjson)

		//todoav: use the commented version
		// withScript := "<script>(function () {var i = (top.invibes = top.invibes || {}); i.bidResponse = " + strings.Replace(adresponse, "[attrs]", "", -1) + ";  })();</script>"
		// withScript = withScript + bidResponses.VideoAdContentResult.BidModel.CreativeHTML

		getlinkurl := "getlink.js"
		withScript := "<script id='ivCrHtmlS'>(function () {var i = (top.invibes = top.invibes || {}); i.bidResponse = " + adresponse + ";  })();"
		withScript = withScript + "(function() { var i = top.invibes = top.invibes || {}; if (i.creativeHtmlRan) { return; } i.creativeHtmlRan = true;  var d = top.document; var e = d.getElementById('divVideoStepAdTop') || d.getElementById('divVideoStepAdTop2') || d.getElementById('divVideoStepAdBottom'); if (e) e.parentNode.removeChild(e); var s = document.getElementById('ivCrHtmlS'); var d = document.createElement('div'); d.setAttribute('id', 'divVideoStepAdTop'); d.className += 'divVideoStep'; s.parentNode.insertBefore(d, s); var j = window.invibes = window.invibes || { }; j.getlinkUrl = '" + getlinkurl + "'; var t = document.createElement('script'); t.src = '" + getlinkurl + "'; s.parentNode.insertBefore(t, s); }()) </script>"

		var bidPrice float64 = 0
		if invibesAd.BidPrice > 0 {
			bidPrice = invibesAd.BidPrice
		} else if pbsdebug {
			bidPrice = 0.000001
		}

		var wsize uint64 = 0
		if adContentResult.BidModel.Width != nil {
			wsize = *adContentResult.BidModel.Width
		}
		var hsize uint64 = 0
		if adContentResult.BidModel.Height != nil {
			hsize = *adContentResult.BidModel.Height
		}
		parsedResponses.Bids = append(parsedResponses.Bids, &adapters.TypedBid{
			Bid: &openrtb.Bid{
				ID:    impId + "_" + invibesAd.VideoExposedID,
				ImpID: impId,
				Price: bidPrice,
				AdM:   withScript,
				CrID:  invibesAd.VideoExposedID,
				W:     wsize,
				H:     hsize,
			},
			BidType: openrtb_ext.BidTypeBanner,
		})
	}

	return parsedResponses, errors
}
