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
const maxBids = 1

type adapter struct {
	endpoint    *template.Template
	endpointUri string
}

type resetDigitalRequest struct {
	Site resetDigitalRequestSite   `json:"site"`
	Imps []resetDigitalRequestImps `json:"imps"`
}

type resetDigitalRequestSite struct {
	Domain   string `json:"domain"`
	Referrer string `json:"referrer"`
}

type resetDigitalRequestImps struct {
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

	for i := range requestData.Imp {
		imp := requestData.Imp[i]

		bidType, err := getBidType(imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		splittedRequestData, err := processDataFromRequest(requestData, imp, bidType)
		if err != nil {
			errors = append(errors, err)
			continue
		}

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
			ImpIDs:  []string{imp.ID},
		})
	}
	//clear requests.body

	return requests, errors
}

func processDataFromRequest(requestData *openrtb2.BidRequest, imp openrtb2.Imp, bidType openrtb_ext.BidType) (resetDigitalRequest, error) {
	var resetDigitalRequestData resetDigitalRequest

	// Check if requestData.Site is not nil before accessing its fields
	if requestData.Site != nil {
		resetDigitalRequestData.Site.Domain = requestData.Site.Domain
		resetDigitalRequestData.Site.Referrer = requestData.Site.Page

	}

	resetDigitalRequestData.Imps = append(resetDigitalRequestData.Imps, resetDigitalRequestImps{
		BidID: requestData.ID,
		ImpID: imp.ID,
	})

	if bidType == openrtb_ext.BidTypeBanner {
		if imp.Banner != nil {
			var tempH int64 = *imp.Banner.H
			var tempW int64 = *imp.Banner.W

			if tempH > 0 && tempW > 0 {
				resetDigitalRequestData.Imps[0].MediaTypes.Banner.Sizes = append(
					resetDigitalRequestData.Imps[0].MediaTypes.Banner.Sizes,
					[]int64{tempH, tempW},
				)
			}

		}
	}

	if bidType == openrtb_ext.BidTypeVideo {
		if imp.Video != nil {
			var tempH int64 = *imp.Video.H
			var tempW int64 = *imp.Video.W

			if tempH > 0 && tempW > 0 {
				resetDigitalRequestData.Imps[0].MediaTypes.Video.Sizes = append(
					resetDigitalRequestData.Imps[0].MediaTypes.Video.Sizes,
					[]int64{tempH, tempW},
				)
			}
		}
	}
	var extData = make(map[string]interface{})
	err := json.Unmarshal(imp.Ext, &extData)
	if err != nil {
		return resetDigitalRequest{}, err
	}

	resetDigitalRequestData.Imps[0].ZoneID.PlacementID = extData["bidder"].(map[string]interface{})["placement_id"].(string)

	return resetDigitalRequestData, nil
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidResp map[string]interface{}
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(maxBids)
	jsonData := bidResp["bids"].([]interface{})

	for _, bidData := range jsonData {
		bidMap := bidData.(map[string]interface{})
		bid := getBidFromResponse(bidMap)
		if bid == nil {
			continue
		}

		bidTypes, err := getBidTypes(internalRequest.Imp[0])
		if err != nil {
			return nil, []error{err}
		}

		for _, bidType := range bidTypes {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			})
		}
	}

	bidResponse.Currency = getCurrency(internalRequest)
	return bidResponse, nil
}
func getBidFromResponse(requestData map[string]interface{}) *openrtb2.Bid {
	processData := requestData["bid_id"].(string)

	bid := &openrtb2.Bid{
		ID:    processData,
		Price: getBidPrice(requestData),
		ImpID: requestData["imp_id"].(string),
		CrID:  requestData["crid"].(string),
	}

	if value, ok := requestData["html"].(string); ok {
		bid.AdM = value
	}

	if value, ok := requestData["w"].(string); ok {
		if i, err := strconv.ParseInt(value, 10, 64); err == nil && i > 0 {
			bid.W = i
		}
	}

	if value, ok := requestData["h"].(string); ok {
		if i, err := strconv.ParseInt(value, 10, 64); err == nil && i > 0 {
			bid.H = i
		}
	}

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

func getBidTypes(imp openrtb2.Imp) ([]openrtb_ext.BidType, error) {
	var bidTypes []openrtb_ext.BidType

	if imp.Banner != nil {
		bidTypes = append(bidTypes, openrtb_ext.BidTypeBanner)
	}
	if imp.Video != nil {
		bidTypes = append(bidTypes, openrtb_ext.BidTypeVideo)
	}
	if imp.Audio != nil {
		bidTypes = append(bidTypes, openrtb_ext.BidTypeAudio)
	}
	if len(bidTypes) == 0 {
		return nil, fmt.Errorf("failed to find matching imp for bid %s", imp.ID)
	}

	return bidTypes, nil
}
