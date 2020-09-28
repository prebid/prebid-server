package adgeneration

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type AdgenerationAdapter struct {
	endpoint        string
	version         string
	defaultCurrency string
}

// Server Responses
type adgServerResponse struct {
	Locationid string        `json:"locationid"`
	Dealid     string        `json:"dealid"`
	Ad         string        `json:"ad"`
	Beacon     string        `json:"beacon"`
	Beaconurl  string        `json:"beaconurl"`
	Cpm        float64       `jsons:"cpm"`
	Creativeid string        `json:"creativeid"`
	H          uint64        `json:"h"`
	W          uint64        `json:"w"`
	Ttl        uint64        `json:"ttl"`
	Vastxml    string        `json:"vastxml,omitempty"`
	LandingUrl string        `json:"landing_url"`
	Scheduleid string        `json:"scheduleid"`
	Results    []interface{} `json:"results"`
}

func (adg *AdgenerationAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	numRequests := len(request.Imp)
	var errs []error

	if numRequests == 0 {
		errs = append(errs, &errortypes.BadInput{
			Message: "No impression in the bid request",
		})
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	bidRequestArray := make([]*adapters.RequestData, 0, numRequests)

	for index := 0; index < numRequests; index++ {
		bidRequestUri, err := adg.getRequestUri(request, index)
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}
		bidRequest := &adapters.RequestData{
			Method:  "GET",
			Uri:     bidRequestUri,
			Body:    nil,
			Headers: headers,
		}
		bidRequestArray = append(bidRequestArray, bidRequest)
	}

	return bidRequestArray, errs
}

func (adg *AdgenerationAdapter) getRequestUri(request *openrtb.BidRequest, index int) (string, error) {
	imp := request.Imp[index]
	adgExt, err := unmarshalExtImpAdgeneration(&imp)
	if err != nil {
		return "", &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	uriObj, err := url.Parse(adg.endpoint)
	if err != nil {
		return "", &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	v := adg.getRawQuery(adgExt.Id, request, &imp)
	uriObj.RawQuery = v.Encode()
	return uriObj.String(), err
}

func (adg *AdgenerationAdapter) getRawQuery(id string, request *openrtb.BidRequest, imp *openrtb.Imp) *url.Values {
	v := url.Values{}
	v.Set("posall", "SSPLOC")
	v.Set("id", id)
	v.Set("sdktype", "0")
	v.Set("hb", "true")
	v.Set("t", "json3")
	v.Set("currency", adg.getCurrency(request))
	v.Set("sdkname", "prebidserver")
	v.Set("adapterver", adg.version)
	adSize := getSizes(imp)
	if adSize != "" {
		v.Set("size", adSize)
	}
	if request.Site != nil && request.Site.Page != "" {
		v.Set("tp", request.Site.Page)
	}
	return &v
}

func unmarshalExtImpAdgeneration(imp *openrtb.Imp) (*openrtb_ext.ExtImpAdgeneration, error) {
	var bidderExt adapters.ExtImpBidder
	var adgExt openrtb_ext.ExtImpAdgeneration
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(bidderExt.Bidder, &adgExt); err != nil {
		return nil, err
	}
	if adgExt.Id == "" {
		return nil, errors.New("No Location ID in ExtImpAdgeneration.")
	}
	return &adgExt, nil
}

func getSizes(imp *openrtb.Imp) string {
	if imp.Banner == nil || len(imp.Banner.Format) == 0 {
		return ""
	}
	var sizeStr string
	for _, v := range imp.Banner.Format {
		sizeStr += strconv.FormatUint(v.W, 10) + "×" + strconv.FormatUint(v.H, 10) + ","
	}
	if len(sizeStr) > 0 && strings.LastIndex(sizeStr, ",") == len(sizeStr)-1 {
		sizeStr = sizeStr[:len(sizeStr)-1]
	}
	return sizeStr
}

func (adg *AdgenerationAdapter) getCurrency(request *openrtb.BidRequest) string {
	if len(request.Cur) <= 0 {
		return adg.defaultCurrency
	} else {
		for _, c := range request.Cur {
			if adg.defaultCurrency == c {
				return c
			}
		}
		return request.Cur[0]
	}
}

func (adg *AdgenerationAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
	var bidResp adgServerResponse
	err := json.Unmarshal(response.Body, &bidResp)
	if err != nil {
		return nil, []error{err}
	}
	if len(bidResp.Results) <= 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	var impId string
	var bitType openrtb_ext.BidType
	var adm string
	for _, v := range internalRequest.Imp {
		adgExt, err := unmarshalExtImpAdgeneration(&v)
		if err != nil {
			return nil, []error{&errortypes.BadServerResponse{
				Message: err.Error(),
			},
			}
		}
		if adgExt.Id == bidResp.Locationid {
			impId = v.ID
			bitType = openrtb_ext.BidTypeBanner
			adm = createAd(&bidResp, impId)
			bid := openrtb.Bid{
				ID:     bidResp.Locationid,
				ImpID:  impId,
				AdM:    adm,
				Price:  bidResp.Cpm,
				W:      bidResp.W,
				H:      bidResp.H,
				CrID:   bidResp.Creativeid,
				DealID: bidResp.Dealid,
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bitType,
			})
			bidResponse.Currency = adg.getCurrency(internalRequest)
			return bidResponse, nil
		}
	}
	return nil, nil
}

func createAd(body *adgServerResponse, impId string) string {
	ad := body.Ad
	if body.Vastxml != "" {
		ad = "<body><div id=\"apvad-" + impId + "\"></div><script type=\"text/javascript\" id=\"apv\" src=\"https://cdn.apvdr.com/js/VideoAd.min.js\"></script>" + insertVASTMethod(impId, body.Vastxml) + "</body>"
	}
	ad = appendChildToBody(ad, body.Beacon)
	unwrappedAd := removeWrapper(ad)
	if unwrappedAd != "" {
		return unwrappedAd
	}
	return ad
}

func insertVASTMethod(bidId string, vastxml string) string {
	rep := regexp.MustCompile(`/\r?\n/g`)
	var replacedVastxml = rep.ReplaceAllString(vastxml, "")
	return "<script type=\"text/javascript\"> (function(){ new APV.VideoAd({s:\"" + bidId + "\"}).load('" + replacedVastxml + "'); })(); </script>"
}

func appendChildToBody(ad string, data string) string {
	rep := regexp.MustCompile(`<\/\s?body>`)
	return rep.ReplaceAllString(ad, data+"</body>")
}

func removeWrapper(ad string) string {
	bodyIndex := strings.Index(ad, "<body>")
	lastBodyIndex := strings.LastIndex(ad, "</body>")
	if bodyIndex == -1 || lastBodyIndex == -1 {
		return ""
	}

	str := strings.TrimSpace(strings.Replace(strings.Replace(ad[bodyIndex:lastBodyIndex], "<body>", "", 1), "</body>", "", 1))
	return str
}

func NewAdgenerationAdapter(endpoint string) *AdgenerationAdapter {
	return &AdgenerationAdapter{
		endpoint,
		"1.0.1",
		"JPY",
	}
}
