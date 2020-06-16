package invibes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
)

// type InvibesBidResponse struct {
// 	VideoAdContentResult VideoAdContentResult `json:"videoAdContentResult"`
// }

// type VideoAdContentResult struct {
// 	Ads          []Ad          `json:"Ads"`
// 	BidModel     BidModel      `json:"BidModel"`
// 	AdPlacements []AdPlacement `json:"AdPlacements"`
// }

// type Ad struct {
// 	VideoExposedID string  `json:"VideoExposedId"`
// 	HTMLString     string  `json:"HtmlString"`
// 	BidPrice       float64 `json:"BidPrice"`
// }

// type AdPlacement struct {
// 	Ads      []Ad     `json:"Ads"`
// 	BidModel BidModel `json:"BidModel"`
// }

// type BidModel struct {
// 	PlacementID  string  `json:"PlacementId"`
// 	CreativeHTML string  `json:"CreativeHtml"`
// 	Width        *uint64 `json:"Width"`
// 	Height       *uint64 `json:"Height"`
// 	Currency     string  `json:"Currency"`
// }

type InvibesAdRequest struct {
	Aver          string
	BidParamsJson string
	Location      string
	Lid           string
	IsTestBid     bool
	Kw            string
	IntegType     int
	Width         string
	Height        string
	GdprConsent   string
	Gdpr          bool
	Bvid          string
}
type InvibesBidParams struct {
	PlacementIds []string
	BidVersion   string
	Properties   map[string]InvibesPlacementProperty
}
type InvibesPlacementProperty struct {
	Formats []openrtb.Format
	ImpId   string
}
type InvibesInternalParams struct {
	BidParams   InvibesBidParams
	IsAmp       bool
	Gdpr        string
	GdprConsent string

	TestIp   string
	TestBvid string
}

const adapterVersion = "prebid_1.0.0"
const maxUriLength = 8000

type BidServerBidderResponse struct {
	Currency string               `json:"currency"`
	Bids     []BidServerItemModel `json:"bids"`
}
type BidServerItemModel struct {
	ID     string  `json:"id"`
	ImpId  string  `json:"impId"`
	Price  float64 `json:"price"`
	CrId   string  `json:"crId"`
	Width  uint64  `json:"width"`
	Height uint64  `json:"height"`
	AdM    string  `json:"adm"`
}

func (a *InvibesInternalParams) IsTestRequest() bool {
	return a.TestIp != "" || a.TestBvid != ""
}

type InvibesAdapter struct {
	EndpointTemplate template.Template
}

func NewInvibesBidder(endpointTemplate string) *InvibesAdapter {
	urlTemplate, err := template.New("endpointTemplate").Parse(endpointTemplate)
	if err != nil {
		glog.Fatal("Unable to parse endpoint url template")
		return nil
	}
	return &InvibesAdapter{EndpointTemplate: *urlTemplate}
}

func (a *InvibesAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No imps in the bid request",
		}}
	}

	consentString := ""
	if request.User != nil {
		var extUser openrtb_ext.ExtUser
		if err := json.Unmarshal(request.User.Ext, &extUser); err == nil {
			consentString = extUser.Consent
		}
	}
	gdprApplies := "1"
	var extRegs openrtb_ext.ExtRegs
	if request.Regs != nil {
		if err := json.Unmarshal(request.Regs.Ext, &extRegs); err == nil {
			if extRegs.GDPR != nil && (*extRegs.GDPR == 0 || *extRegs.GDPR == 1) {
				gdprApplies = strconv.Itoa(int(*extRegs.GDPR))
			}
		}
	}

	var httpRequests []*adapters.RequestData
	var errors []error

	var invibesInternalParams InvibesInternalParams = InvibesInternalParams{
		BidParams: InvibesBidParams{
			Properties: make(map[string]InvibesPlacementProperty),
			BidVersion: "1",
		},
	}

	for _, imp := range request.Imp {

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: "Error parsing bidderExt object",
			})
		}

		if len(errors) == 0 {
			var invibesExt openrtb_ext.ExtImpInvibes
			if err := json.Unmarshal(bidderExt.Bidder, &invibesExt); err != nil {
				errors = append(errors, &errortypes.BadInput{
					Message: "Error parsing invibesExt parameters",
				})
			}

			if imp.Banner == nil {
				errors = append(errors, &errortypes.BadInput{
					Message: "Banner not specified",
				})
			}

			if len(errors) == 0 {

				currentBanner := *imp.Banner
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

				invibesInternalParams.BidParams.PlacementIds = append(invibesInternalParams.BidParams.PlacementIds, strings.TrimSpace(invibesExt.PlacementId))
				invibesInternalParams.BidParams.Properties[invibesExt.PlacementId] = InvibesPlacementProperty{
					ImpId:   imp.ID,
					Formats: adFormats,
				}

				if reqInfo.PbsEntryPoint == pbsmetrics.ReqTypeAMP || invibesExt.Debug.TestAmp == "true" {
					invibesInternalParams.IsAmp = true
				}

				if invibesExt.Debug.TestIp != "" {
					invibesInternalParams.TestIp = invibesExt.Debug.TestIp
				}
				if invibesExt.Debug.TestBvid != "" {
					invibesInternalParams.TestBvid = invibesExt.Debug.TestBvid
				}
			}
		}
	}

	if len(invibesInternalParams.BidParams.PlacementIds) == 0 {
		return nil, errors
	}

	invibesInternalParams.Gdpr = gdprApplies
	invibesInternalParams.GdprConsent = consentString

	newHttpRequest, err := a.makeRequest(invibesInternalParams, reqInfo, httpRequests, request)
	if err != nil {
		errors = append(errors, err)
	} else if newHttpRequest != nil {
		httpRequests = append(httpRequests, newHttpRequest)
	}

	return httpRequests, errors
}

func (a *InvibesAdapter) makeRequest(invibesParams InvibesInternalParams, reqInfo *adapters.ExtraRequestInfo, existingRequests []*adapters.RequestData, request *openrtb.BidRequest) (*adapters.RequestData, error) {

	url, err := a.makeURL(request)
	if err != nil {
		return nil, err
	}
	parameter, errp := a.makeParameter(invibesParams, request)
	if errp != nil {
		return nil, errp
	}
	body, errm := json.Marshal(parameter)
	if errm != nil {
		return nil, errm
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	if request.Device != nil {
		headers.Add("User-Agent", request.Device.UA)
	}

	pbsdebug := "false"
	if invibesParams.IsTestRequest() {
		pbsdebug = "true"
	}
	if invibesParams.TestIp != "" {
		headers.Add("X-Forwarded-For", invibesParams.TestIp)
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

func (a *InvibesAdapter) makeParameter(invibesParams InvibesInternalParams, request *openrtb.BidRequest) (*InvibesAdRequest, error) {
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
	if request.Site == nil {
		return nil, &errortypes.BadInput{
			Message: "Site not specified",
		}
	}

	integType := 0
	if invibesParams.IsAmp {
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
	if invibesParams.IsTestRequest() {
		if width == "" {
			width = "500"
		}
		if height == "" {
			height = "500"
		}
	}

	var invRequest InvibesAdRequest
	bidParamsJson, err := json.Marshal(invibesParams.BidParams)
	if err == nil {
		invRequest = InvibesAdRequest{
			Aver:          adapterVersion,
			IsTestBid:     invibesParams.IsTestRequest(),
			BidParamsJson: string(bidParamsJson), //"{placementIds:[\"" + params.PlacementId + "\"],bidVersion:\"1\"}",
			Location:      request.Site.Page,
			Lid:           lid,
			Kw:            request.Site.Keywords,
			IntegType:     integType,
			Width:         width,
			Height:        height,
			GdprConsent:   invibesParams.GdprConsent,
			Gdpr:          invibesParams.Gdpr != "0",
			Bvid:          invibesParams.TestBvid,
		}
	}

	return &invRequest, err
}

func (a *InvibesAdapter) makeURL(request *openrtb.BidRequest) (string, error) {
	var endpointURL *url.URL
	endpointParams := macros.EndpointTemplateParams{}
	host, err := macros.ResolveMacros(a.EndpointTemplate, endpointParams)

	if err == nil {
		endpointURL, err = url.Parse(host)
	}
	if err != nil {
		return "", &errortypes.BadInput{
			Message: "Unable to parse url template: " + err.Error(),
		}
	}

	return endpointURL.String(), nil
}

func (a *InvibesAdapter) MakeBids(
	internalRequest *openrtb.BidRequest,
	externalRequest *adapters.RequestData,
	response *adapters.ResponseData,
) (*adapters.BidderResponse, []error) {
	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d.", response.StatusCode)}
	}

	bidResponse := BidServerBidderResponse{}
	if err := json.Unmarshal(response.Body, &bidResponse); err != nil {
		return nil, []error{err}
	}

	var parsedResponses = adapters.NewBidderResponseWithBidsCapacity(len(bidResponse.Bids))
	var errors []error
	parsedResponses.Currency = bidResponse.Currency

	for _, bid := range bidResponse.Bids {
		parsedResponses.Bids = append(parsedResponses.Bids, &adapters.TypedBid{
			Bid: &openrtb.Bid{
				ID:    bid.ID,
				ImpID: bid.ImpId,
				Price: bid.Price,
				AdM:   bid.AdM,
				CrID:  bid.CrId,
				W:     bid.Width,
				H:     bid.Height,
			},
			BidType: openrtb_ext.BidTypeBanner,
		})
	}

	return parsedResponses, errors
}

// func (a *InvibesAdapter) MakeBids(
// 	internalRequest *openrtb.BidRequest,
// 	externalRequest *adapters.RequestData,
// 	response *adapters.ResponseData,
// ) (*adapters.BidderResponse, []error) {
// 	if response.StatusCode != http.StatusOK {
// 		return nil, []error{fmt.Errorf("Unexpected status code: %d.", response.StatusCode)}
// 	}

// 	bidResponses := InvibesBidResponse{}

// 	currentRequest := *externalRequest
// 	var impId string = ""
// 	if len(currentRequest.Headers["Impid"]) > 0 {
// 		impId = currentRequest.Headers["Impid"][0]
// 	}
// 	var pbsdebug bool = false
// 	if len(currentRequest.Headers["Pbsdebug"]) > 0 {
// 		pbsdebug = currentRequest.Headers["Pbsdebug"][0] == "true"
// 	}

// 	if err := json.Unmarshal(response.Body, &bidResponses); err != nil {
// 		return nil, []error{err}
// 	}

// 	var parsedResponses = adapters.NewBidderResponseWithBidsCapacity(len(bidResponses.VideoAdContentResult.Ads))
// 	var errors []error

// 	bidModel := bidResponses.VideoAdContentResult.BidModel
// 	invibesAds := bidResponses.VideoAdContentResult.Ads
// 	if len(invibesAds) == 0 {
// 		invibesAds = bidResponses.VideoAdContentResult.AdPlacements[0].Ads
// 		if bidModel.Currency == "" {
// 			bidModel = bidResponses.VideoAdContentResult.AdPlacements[0].BidModel
// 		}
// 	}

// 	if bidModel.Currency != "" {
// 		parsedResponses.Currency = bidModel.Currency
// 	}

// 	bidResponses.VideoAdContentResult.Ads = nil
// 	for _, invibesAd := range invibesAds {
// 		adContentResult := bidResponses.VideoAdContentResult
// 		// adContentResult.Ads = []Ad{invibesAd}

// 		// adjson, _ := json.Marshal(adContentResult)
// 		// adresponse := string(adjson)

// 		//todoav: use the commented version
// 		// withScript := "<script id='ivCrHtmlS'>(function () {var i = (top.invibes = top.invibes || {}); var fullResponse = " + adresponse + "; debugger; i.bidResponse = fullResponse.videoAdContentResult;  })();"
// 		// withScript = withScript + bidResponses.VideoAdContentResult.BidModel.CreativeHTML

// 		getlinkurl := "getlink.js"
// 		withScript := "<script id='ivCrHtmlS'>(function () {var i = (top.invibes = top.invibes || {}); var fullResponse = " + string(response.Body) + "; debugger; i.bidResponse = fullResponse.videoAdContentResult;  })();"
// 		withScript = withScript + "(function() { var i = top.invibes = top.invibes || {}; if (i.creativeHtmlRan) { return; } i.creativeHtmlRan = true;  var d = top.document; var e = d.getElementById('divVideoStepAdTop') || d.getElementById('divVideoStepAdTop2') || d.getElementById('divVideoStepAdBottom'); if (e) e.parentNode.removeChild(e); var s = document.getElementById('ivCrHtmlS'); var d = document.createElement('div'); d.setAttribute('id', 'divVideoStepAdTop'); d.className += 'divVideoStep'; s.parentNode.insertBefore(d, s); var j = window.invibes = window.invibes || { }; j.getlinkUrl = '" + getlinkurl + "'; var t = document.createElement('script'); t.src = '" + getlinkurl + "'; s.parentNode.insertBefore(t, s); }()) </script>"

// 		var bidPrice float64 = 0
// 		if invibesAd.BidPrice > 0 {
// 			bidPrice = invibesAd.BidPrice
// 		} else if pbsdebug {
// 			bidPrice = 0.000001
// 		}

// 		var wsize uint64 = 0
// 		if adContentResult.BidModel.Width != nil {
// 			wsize = *adContentResult.BidModel.Width
// 		}
// 		var hsize uint64 = 0
// 		if adContentResult.BidModel.Height != nil {
// 			hsize = *adContentResult.BidModel.Height
// 		}
// 		parsedResponses.Bids = append(parsedResponses.Bids, &adapters.TypedBid{
// 			Bid: &openrtb.Bid{
// 				ID:    impId + "_" + invibesAd.VideoExposedID,
// 				ImpID: "1",
// 				Price: bidPrice,
// 				AdM:   withScript,
// 				CrID:  invibesAd.VideoExposedID,
// 				W:     wsize,
// 				H:     hsize,
// 			},
// 			BidType: openrtb_ext.BidTypeBanner,
// 		})
// 	}

// 	return parsedResponses, errors
// }
