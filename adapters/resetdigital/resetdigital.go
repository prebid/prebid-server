package resetdigital

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

// MaximumBids is the maximum number of bids that can be returned by this adapter.
const (
	MaxBids = 1
)

type adapter struct {
	endpoint    *template.Template
	endpointUri string
}

type resetDigitalRequest struct {
	Site resetDigitalRequestSite  `json:"site"`
	Imps []resetDigitalRequesImps `json:"imps"`
}

type resetDigitalRequestSite struct {
	Domain   string `json:"domain"`
	Referrer string `json:"referrer"`
}

type resetDigitalRequesImps struct {
	ForceBid bool `json:"force_bid"`
	ZoneID   struct {
		PlacementID string `json:"placementId"`
	} `json:"zone_id"`
	BidID string `json:"bid_id"`
	ImpID string `json:"imp_id"`
	Ext   struct {
		Gpid string `json:"gpid"`
	} `json:"ext"`
	Sizes      [][]int64 `json:"sizes"`
	MediaTypes struct {
		Banner struct {
			Sizes [][]int64 `json:"sizes"`
		} `json:"banner"`
		Video struct {
			Sizes [][]int64 `json:"sizes"`
		} `json:"video"`
	} `json:"media_types"`
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpoint: template,
	}

	return bidder, nil
}
func getHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}

	if request.Device != nil && request.Site != nil {
		addNonEmptyHeaders(&headers, map[string]string{
			"Referer":         request.Site.Page,
			"Accept-Language": request.Device.Language,
			"User-Agent":      request.Device.UA,
			"X-Forwarded-For": request.Device.IP,
			"X-Real-Ip":       request.Device.IP,
			"Content-Type":    "application/json;charset=utf-8",
			"Accept":          "application/json",
		})
	}

	return headers
}
func addNonEmptyHeaders(headers *http.Header, headerValues map[string]string) {
	for key, value := range headerValues {
		if len(value) > 0 {
			headers.Add(key, value)
		}
	}
}

func getReferer(request *openrtb2.BidRequest) string {
	if request.Site == nil {
		return ""
	}

	return request.Site.Domain
}

func getCurrency(request *openrtb2.BidRequest) string {
	if len(request.Cur) == 0 {
		return "USD"
	}

	return request.Cur[0]
}

func (a *adapter) MakeRequests(requestData *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var (
		requests []*adapters.RequestData
		errors   []error
	)

	referer := getReferer(requestData)
	currency := getCurrency(requestData)

	if referer == currency {
		return nil, nil
	}

	for i := range requestData.Imp {
		imp := requestData.Imp[i]
		bidType, err := getBidType(imp)

		if err != nil {
			errors = append(errors, err)
			continue
		}

		splittedRequestData := processDataFromRequest(requestData, imp, bidType)

		requestBody, err := json.Marshal(splittedRequestData)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		requests = append(requests, &adapters.RequestData{
			Method:  "POST",
			Uri:     a.endpointUri,
			Body:    requestBody,
			Headers: getHeaders(requestData),
		})
	}

	return requests, errors
}

func processDataFromRequest(requestData *openrtb2.BidRequest, imp openrtb2.Imp, bidType openrtb_ext.BidType) resetDigitalRequest {

	var resetDigitalRequestData resetDigitalRequest
	resetDigitalRequestData.Site.Domain = requestData.Site.Domain
	resetDigitalRequestData.Site.Referrer = requestData.Site.Page

	resetDigitalRequestData.Imps = append(resetDigitalRequestData.Imps, resetDigitalRequesImps{})
	resetDigitalRequestData.Imps[0].BidID = requestData.ID
	resetDigitalRequestData.Imps[0].ImpID = imp.ID

	var err error

	if bidType == openrtb_ext.BidTypeBanner {
		resetDigitalRequestData.Imps[0].MediaTypes.Banner.Sizes = append(resetDigitalRequestData.Imps[0].MediaTypes.Banner.Sizes, []int64{imp.Banner.Format[0].W, imp.Banner.Format[0].H})
	}
	if bidType == openrtb_ext.BidTypeVideo {
		resetDigitalRequestData.Imps[0].MediaTypes.Video.Sizes = append(resetDigitalRequestData.Imps[0].MediaTypes.Banner.Sizes, []int64{*imp.Video.W, *imp.Video.H})
	}

	var extData = make(map[string]interface{})
	err = json.Unmarshal(imp.Ext, &extData)
	if err != nil {

	} else {

		resetDigitalRequestData.Imps[0].ZoneID.PlacementID = extData["bidder"].(map[string]interface{})["placement_id"].(string)
		if resetDigitalRequestData.Imps[0].ZoneID.PlacementID == "test" {
			resetDigitalRequestData.Imps[0].ForceBid = true
		}

	}
	return resetDigitalRequestData

}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if response.StatusCode == http.StatusBadRequest {
		return nil, nil
	}
	if response.StatusCode != http.StatusOK {
		return nil, nil
	}

	if err := json.Unmarshal(response.Body, &response); err != nil {
		return nil, []error{err}
	}
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(MaxBids)
	//check no bids
	jsonData := make(map[string]interface{})

	json.Unmarshal([]byte(response.Body), &jsonData)
	//Always one bid
	bid := getBidFromResponse(jsonData)

	bidType, err := getBidType(internalRequest.Imp[0])
	if err != nil {
		// handle error
		return nil, []error{err}
	}
	bidResponse.Currency = getCurrency(internalRequest)
	bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
		Bid:     bid,
		BidType: bidType,
	})

	return bidResponse, nil
}

func getBidFromResponse(requestData map[string]interface{}) *openrtb2.Bid {
	processData := requestData["bids"].([]interface{})[0].(map[string]interface{})

	bid := &openrtb2.Bid{
		ID:    processData["bid_id"].(string),
		Price: getBidPrice(processData),
		ImpID: processData["imp_id"].(string),
		CrID:  processData["crid"].(string),
	}
	//if HTML is filled on jsonData then fill ADM with it
	if value, ok := processData["html"].(string); ok {
		bid.AdM = value
	}
	//if Width and Height are filled on jsonData then fill W and H with it
	if value, ok := processData["w"].(string); ok {

		i, _ := strconv.ParseInt(value, 10, 64)
		if i > 0 {
			bid.W = i
		}
	}
	if value, ok := processData["h"].(string); ok {
		i, _ := strconv.ParseInt(value, 10, 64)
		if i > 0 {
			bid.H = i
		}
	}
	//if Bid Price is 0 then return nil
	if bid.Price == 0 {
		return nil
	}

	return bid
}

func getBidPrice(requestData map[string]interface{}) float64 {
	if value, ok := requestData["cpm"].(float64); ok {
		return value
	}
	return 0.0 // Default value if "cpm" doesn't exist or is not a float64
}

func getBidType(imp openrtb2.Imp) (openrtb_ext.BidType, error) {
	if imp.Banner != nil {
		return openrtb_ext.BidTypeBanner, nil
	} else if imp.Video != nil {
		return openrtb_ext.BidTypeVideo, nil
	} else if imp.Audio != nil {
		return openrtb_ext.BidTypeAudio, nil
	}

	return "", fmt.Errorf("failed to find matching imp for bid %s", imp.ID)
}
