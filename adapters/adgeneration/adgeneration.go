package adgeneration

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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

	requestArray := make([]*adapters.RequestData, 0, numRequests)
	for index := 0; index < numRequests; index++ {
		headers := http.Header{}
		headers.Add("Content-Type", "application/json;charset=utf-8")
		headers.Add("Accept", "application/json")

		requestUri, err := adg.getRequestUri(request, index)
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}
		request := &adapters.RequestData{
			Method:  "GET",
			Uri:     requestUri,
			Body:    nil,
			Headers: headers,
		}
		requestArray = append(requestArray, request)
	}

	return requestArray, errs
}

func (adg *AdgenerationAdapter) getRequestUri(request *openrtb.BidRequest, index int) (string, error) {
	imp := request.Imp[index]
	bidderExt, err := adg.unmarshalExtImpBidder(&imp)
	if err != nil {
		return "", &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	adgExt, err := adg.unmarshalExtImpAdgeneration(&bidderExt)
	if err != nil {
		return "", &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	endpoint := adg.endpoint +
		"?posall=SSPLOC" +
		"&id=" + adgExt.Id +
		"&sdktype=0" +
		"&hb=true" +
		"&t=json3" +
		"&currency=" + adg.getCurrency(request) +
		"&sdkname=prebidserver" +
		"&adapterver=" + adg.version
	if adg.getSizes(&imp) != "" {
		endpoint += "&sizes=" + adg.getSizes(&imp)
	}
	if request.Site != nil && request.Site.Page != "" {
		endpoint += "&tp=" + request.Site.Page
	}

	return endpoint, nil
}

func (adg *AdgenerationAdapter) unmarshalExtImpBidder(imp *openrtb.Imp) (adapters.ExtImpBidder, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return bidderExt, err
	}
	return bidderExt, nil
}

func (adg *AdgenerationAdapter) unmarshalExtImpAdgeneration(ext *adapters.ExtImpBidder) (openrtb_ext.ExtImpAdgeneration, error) {
	var adgExt openrtb_ext.ExtImpAdgeneration
	if err := json.Unmarshal(ext.Bidder, &adgExt); err != nil {
		return adgExt, err
	}
	if adgExt.Id == "" {
		return adgExt, errors.New("No Location ID in ExtImpAdgeneration.")
	}
	return adgExt, nil
}

func (adg *AdgenerationAdapter) getSizes(imp *openrtb.Imp) string {
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
		bidderExt, err := adg.unmarshalExtImpBidder(&v)
		if err != nil {
			return nil, []error{&errortypes.BadServerResponse{
				Message: err.Error(),
			},
			}
		}
		adgExt, err := adg.unmarshalExtImpAdgeneration(&bidderExt)
		if err != nil {
			return nil, []error{&errortypes.BadServerResponse{
				Message: err.Error(),
			},
			}
		}
		if adgExt.Id == bidResp.Locationid {
			impId = v.ID
			bitType = openrtb_ext.BidTypeBanner
			adm = adg.createAd(&bidResp, impId)
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
		}
	}
	return bidResponse, nil
}

func (adg *AdgenerationAdapter) createAd(body *adgServerResponse, impId string) string {
	ad := body.Ad
	if body.Vastxml != "" {
		ad = "<body><div id=\"apvad-" + impId + "\"></div><script type=\"text/javascript\" id=\"apv\" src=\"https://cdn.apvdr.com/js/VideoAd.min.js\"></script>" + adg.insertVASTMethod(impId, body.Vastxml) + "</body>"
	}
	ad = adg.appendChildToBody(ad, body.Beacon)
	if adg.removeWrapper(ad) != "" {
		return adg.removeWrapper(ad)
	}
	return ad
}

func (adg *AdgenerationAdapter) insertVASTMethod(bidId string, vastxml string) string {
	rep := regexp.MustCompile(`/\r?\n/g`)
	var replacedVastxml = rep.ReplaceAllString(vastxml, "")
	return "<script type=\"text/javascript\"> (function(){ new APV.VideoAd({s:\"" + bidId + "\"}).load('" + replacedVastxml + "'); })(); </script>"
}

func (adg *AdgenerationAdapter) appendChildToBody(ad string, data string) string {
	rep := regexp.MustCompile(`<\/\s?body>`)
	return rep.ReplaceAllString(ad, data+"</body>")
}

func (adg *AdgenerationAdapter) removeWrapper(ad string) string {
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
		"1.0.0",
		"JPY",
	}
}
