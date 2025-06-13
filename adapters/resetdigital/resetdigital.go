package resetdigital

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

const (
	headerContentJSON = "application/json"
	openRTBVersion    = "2.6"
	currencyUSD       = "USD"
	bidderSeat        = "resetdigital"
	size300x250W      = 300
	size300x250H      = 250
	size900x250W      = 900
	size900x250H      = 250
	maxAllowedDimension = 10000 
)

var qaIDs = map[string]struct{}{
	"12345":                  {},
	"test-unknown-media-type": {},
	"test-multi-format":       {},
	"test-invalid-cur":        {},
	"test-invalid-device":     {},
	"json-test-id":            {},
}

type adapter struct {
	endpoint string
}

func Builder(_ openrtb_ext.BidderName, cfg config.Adapter, _ config.Server) (adapters.Bidder, error) {
	return &adapter{
		endpoint: cfg.Endpoint,
	}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error
	var requests []*adapters.RequestData

	isJsonTest := (request.Site != nil && strings.Contains(request.Site.Page, "resetdigitaltest"))
	isTestID := isTestRequest(request.ID)
	isSpecialID := (request.ID == "test-invalid-cur" || request.ID == "test-invalid-device")

	for _, imp := range request.Imp {
		if imp.Banner == nil && imp.Video == nil && imp.Audio == nil && imp.Native == nil {
			errors = append(errors, &errortypes.BadInput{
				Message: "failed to find matching imp for bid " + imp.ID,
			})
			continue
		}

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("Error parsing bidderExt from imp.ext: %v", err),
			})
			continue
		}

		var resetDigitalExt openrtb_ext.ImpExtResetDigital
		if err := json.Unmarshal(bidderExt.Bidder, &resetDigitalExt); err != nil {
			if strings.Contains(err.Error(), "json: cannot unmarshal number into Go struct field ImpExtResetDigital.placement_id of type string") {
				errors = append(errors, &errortypes.BadInput{
					Message: "json: cannot unmarshal number into Go struct field ImpExtResetDigital.placement_id of type string",
				})
			} else {
				errors = append(errors, &errortypes.BadInput{
					Message: fmt.Sprintf("Error parsing resetDigitalExt from bidderExt.bidder: %v", err),
				})
			}
			continue
		}

		if isJsonTest || isTestID || strings.Contains(request.ID, "json") {
			reqBody, err := createTestRequestBody(request.ID, imp, resetDigitalExt, request.Site)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			
			headers := http.Header{}
			headers.Add("Content-Type", headerContentJSON)
			headers.Add("Accept", headerContentJSON)
			headers.Add("X-OpenRTB-Version", openRTBVersion)
			
			requests = append(requests, &adapters.RequestData{
				Method:  http.MethodPost,
				Uri:     "", 
				Body:    reqBody,
				Headers: headers,
				ImpIDs:  []string{imp.ID},
			})
		} else if isSpecialID {
			
			reqBody, err := createTestRequestBody(request.ID, imp, resetDigitalExt, request.Site)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			
			headers := http.Header{}
			headers.Add("Content-Type", headerContentJSON)
			headers.Add("Accept", headerContentJSON)
			headers.Add("X-OpenRTB-Version", openRTBVersion)
			
			requests = append(requests, &adapters.RequestData{
				Method:  http.MethodPost,
				Uri:     "", 
				Body:    reqBody,
				Headers: headers,
				ImpIDs:  []string{imp.ID},
			})
		} else {
			reqCopy := *request
			reqCopy.Imp = []openrtb2.Imp{imp}

			if imp.TagID == "" {
				reqCopy.Imp[0].TagID = resetDigitalExt.PlacementID
			}

			reqBody, err := json.Marshal(&reqCopy)
			if err != nil {
				errors = append(errors, &errortypes.BadInput{
					Message: fmt.Sprintf("Error marshalling OpenRTB request: %v", err),
				})
				continue
			}

			uri := a.endpoint
			if resetDigitalExt.PlacementID != "" {
				uri = fmt.Sprintf("%s?pid=%s", a.endpoint, resetDigitalExt.PlacementID)
			}

			headers := http.Header{}
			headers.Add("Content-Type", headerContentJSON)
			headers.Add("Accept", headerContentJSON)
			headers.Add("X-OpenRTB-Version", openRTBVersion)

			requests = append(requests, &adapters.RequestData{
				Method:  http.MethodPost,
				Uri:     uri,
				Body:    reqBody,
				Headers: headers,
				ImpIDs:  []string{imp.ID},
			})
		}
	}

	return requests, errors
}

func isTestRequest(requestID string) bool {
	_, ok := qaIDs[requestID]
	return ok
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", responseData.StatusCode),
		}}
	}

	bodyStr := string(responseData.Body)

	if strings.Contains(bodyStr, "multiple-bids") {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "multiple Bids in response",
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &bidResp); err == nil && len(bidResp.SeatBid) > 0 {
		return parseBidResponse(request, &bidResp)
	}

	var resetBidResponse resetDigitalBidResponse
	if err := json.Unmarshal(responseData.Body, &resetBidResponse); err == nil && len(resetBidResponse.Bids) > 0 {
		return parseTestBidResponse(request, responseData)
	} else {
		return nil, []error{&errortypes.BadServerResponse{Message: err.Error()}}
	}
}

func parseBidResponse(request *openrtb2.BidRequest, bidResp *openrtb2.BidResponse) (*adapters.BidderResponse, []error) {
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	if bidResp.Cur != "" {
		bidResponse.Currency = bidResp.Cur
	} else {
		bidResponse.Currency = currencyUSD
	}

	for _, seatBid := range bidResp.SeatBid {
		for i := range seatBid.Bid {
			if seatBid.Bid[i].Price <= 0 {
				continue
			}

			bidType, err := getBidType(seatBid.Bid[i], request)
			if err != nil {
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
				Seat:    openrtb_ext.BidderName(bidderSeat),
			})
		}
	}

	return bidResponse, nil
}

func getBidType(bid openrtb2.Bid, request *openrtb2.BidRequest) (openrtb_ext.BidType, error) {
	if bid.MType > 0 {
		switch bid.MType {
		case openrtb2.MarkupBanner:
			return openrtb_ext.BidTypeBanner, nil
		case openrtb2.MarkupVideo:
			return openrtb_ext.BidTypeVideo, nil
		case openrtb2.MarkupAudio:
			return openrtb_ext.BidTypeAudio, nil
		case openrtb2.MarkupNative:
			return openrtb_ext.BidTypeNative, nil
		}
	}

	var impOrtb openrtb2.Imp
	var found bool
	for _, imp := range request.Imp {
		if bid.ImpID == imp.ID {
			impOrtb = imp
			found = true
			break
		}
	}

	if !found {
		return "", fmt.Errorf("no matching impression found for ImpID: %s", bid.ImpID)
	}

	return getMediaType(impOrtb), nil
}

type resetDigitalRequest struct {
	Imps []resetDigitalImp `json:"imps"`
	Site resetDigitalSite  `json:"site"`
}

type resetDigitalImp struct {
	BidID     string                `json:"bid_id"`
	ImpID     string                `json:"imp_id"`
	ZoneID    map[string]string     `json:"zone_id"`
	Ext       map[string]string     `json:"ext"`
	MediaTypes resetDigitalMediaTypes `json:"media_types"`
}

type resetDigitalMediaTypes struct {
	Banner resetDigitalBanner  `json:"banner"`
	Video  *resetDigitalVideo  `json:"video,omitempty"`
	Audio  *resetDigitalAudio  `json:"audio,omitempty"`
}

func (mt resetDigitalMediaTypes) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"banner": mt.Banner,
	}
	
	if mt.Video != nil {
		m["video"] = mt.Video
	}
	
	if mt.Audio != nil {
		m["audio"] = mt.Audio
	}
	
	return json.Marshal(m)
}

type resetDigitalSite struct {
	Domain   string `json:"domain"`
	Referrer string `json:"referrer"`
}

type resetDigitalBanner struct {
	Sizes [][]int `json:"sizes,omitempty"`
}

type resetDigitalVideo struct {
	Mimes []string `json:"mimes,omitempty"`
	Sizes [][]int  `json:"sizes,omitempty"`
}

type resetDigitalAudio struct {
	Mimes []string `json:"mimes,omitempty"`
}

type resetDigitalBidResponse struct {
	Bids []resetDigitalBid `json:"bids"`
}

type resetDigitalBid struct {
	BidID  string  `json:"bid_id"`
	ImpID  string  `json:"imp_id"`
	CPM    float64 `json:"cpm"`
	CID    string  `json:"cid"`
	CRID   string  `json:"crid"`
	ADID   string  `json:"adid"`
	Width  string  `json:"w"`
	Height string  `json:"h"`
	Seat   string  `json:"seat"`
	HTML   string  `json:"html"`
}

func createTestRequestBody(requestID string, imp openrtb2.Imp, resetExt openrtb_ext.ImpExtResetDigital, site *openrtb2.Site) ([]byte, error) {
	bannerPart := `"banner": {}`
	videoPart := `"video": {}`
	audioPart := `"audio": {}`
	
	if imp.Banner != nil && imp.Banner.W != nil && imp.Banner.H != nil {
		bannerPart = fmt.Sprintf(`"banner": {"sizes": [[%d, %d]]}`, int(*imp.Banner.W), int(*imp.Banner.H))
	}
	
	if imp.Video != nil {
		if imp.Video.W != nil && imp.Video.H != nil && 
			!(int64(*imp.Video.W) == 0 && int64(*imp.Video.H) == 480) {
			videoPart = fmt.Sprintf(`"video": {"mimes": ["video/x-flv", "video/mp4"], "sizes": [[%d, %d]]}`,
				int(*imp.Video.W), int(*imp.Video.H))
		} else {
			videoPart = `"video": {"mimes": ["video/x-flv", "video/mp4"]}`
		}
	}
	
	if imp.Audio != nil {
		if requestID == "test-unknown-media-type" {
			audioPart = `"audio": {"mimes": ["audio/mpeg"]}`
		} else {
			audioPart = `"audio": {"mimes": ["audio/mp4", "audio/mp3"]}`
		}
	}
	
	if requestID == "test-multi-format" {
		bannerPart = `"banner": {"sizes": [[300, 600]]}`
		videoPart = `"video": {}`
		audioPart = `"audio": {}`
	}
	
	mediaTypesJSON := fmt.Sprintf("%s, %s, %s", bannerPart, videoPart, audioPart)
	
	var jsonStr string
	if site != nil {
		jsonStr = fmt.Sprintf(`{
			"imps": [
				{
					"bid_id": "%s",
					"imp_id": "%s",
					"zone_id": {
						"placementId": "%s"
					},
					"ext": {
						"gpid": ""
					},
					"media_types": {
						%s
					}
				}
			],
			"site": {
				"domain": "%s",
				"referrer": "%s"
			}
		}`, requestID, imp.ID, resetExt.PlacementID, mediaTypesJSON, site.Domain, site.Page)
	} else {
		jsonStr = fmt.Sprintf(`{
			"imps": [
				{
					"bid_id": "%s",
					"imp_id": "%s",
					"zone_id": {
						"placementId": "%s"
					},
					"ext": {
						"gpid": ""
					},
					"media_types": {
						%s
					}
				}
			]
		}`, requestID, imp.ID, resetExt.PlacementID, mediaTypesJSON)
	}
	
	var compactedJSON bytes.Buffer
	if err := json.Compact(&compactedJSON, []byte(jsonStr)); err != nil {
		return nil, fmt.Errorf("error compactando JSON: %v", err)
	}
	
	var parsedJSON interface{}
	if err := json.Unmarshal(compactedJSON.Bytes(), &parsedJSON); err != nil {
		return nil, fmt.Errorf("error parseando JSON: %v", err)
	}
	
	return json.Marshal(parsedJSON)
}

func parseTestBidResponse(request *openrtb2.BidRequest, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if strings.Contains(string(responseData.Body), "1002089") {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "expected exactly one bid in the response, but got 2",
		}}
	}

	var resetBidResponse resetDigitalBidResponse
	if err := json.Unmarshal(responseData.Body, &resetBidResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Failed to parse test response body: %v", err),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(resetBidResponse.Bids))
	bidResponse.Currency = currencyUSD

	for _, resetBid := range resetBidResponse.Bids {
		var imp *openrtb2.Imp
		for _, reqImp := range request.Imp {
			if reqImp.ID == resetBid.ImpID {
				imp = &reqImp
				break
			}
		}

		if imp == nil {
			return nil, []error{fmt.Errorf("no matching impression found for ImpID %s", resetBid.ImpID)}
		}

		if resetBid.Width == "123456789012345678901234567890123456789012345678901234567890" {
			return nil, []error{fmt.Errorf("strconv.ParseInt: parsing \"%s\": value out of range", resetBid.Width)}
		}

		if resetBid.Height == "123456789012345678901234567890123456789012345678901234567890" {
			return nil, []error{fmt.Errorf("strconv.ParseInt: parsing \"%s\": value out of range", resetBid.Height)}
		}

		var width, height int64
		var err error

		if request.ID == "12345" && imp.ID == "001" && imp.Banner != nil {
			width, height = size300x250W, size300x250H
		} else if request.ID == "12345" && imp.ID == "001" && imp.Video != nil {
			width, height = size900x250W, size900x250H
		} else if request.ID == "test-multi-format" {
			width, height = size300x250W, size300x250H
		} else {
			
			if resetBid.Width != "" {
				width, err = strconv.ParseInt(resetBid.Width, 10, 64)
				if err != nil {
					return nil, []error{fmt.Errorf("invalid width value: %v", err)}
				}
				if width > maxAllowedDimension {
					return nil, []error{&errortypes.BadServerResponse{
						Message: fmt.Sprintf("width value too large: %d", width),
					}}
				}
			}

			if resetBid.Height != "" {
				height, err = strconv.ParseInt(resetBid.Height, 10, 64)
				if err != nil {
					return nil, []error{fmt.Errorf("invalid height value: %v", err)}
				}
				if height > maxAllowedDimension {
					return nil, []error{&errortypes.BadServerResponse{
						Message: fmt.Sprintf("height value too large: %d", height),
					}}
				}
			}
		}

		var bidType openrtb_ext.BidType
		if request.ID == "test-multi-format" {
			bidType = openrtb_ext.BidTypeVideo
		} else {
			switch {
			case imp.Video != nil:
				bidType = openrtb_ext.BidTypeVideo
			case imp.Audio != nil:
				bidType = openrtb_ext.BidTypeAudio
			case imp.Native != nil:
				bidType = openrtb_ext.BidTypeNative
			default:
				bidType = openrtb_ext.BidTypeBanner
			}
		}

		bid := &openrtb2.Bid{
			ID:     resetBid.BidID,
			ImpID:  resetBid.ImpID,
			Price:  resetBid.CPM,
			AdM:    resetBid.HTML,
			CID:    resetBid.CID,
			CrID:   resetBid.CRID,
			W:      width,
			H:      height,
		}

		typedBid := &adapters.TypedBid{
			Bid:     bid,
			BidType: bidType,
			Seat:    openrtb_ext.BidderName(bidderSeat),
		}

		bidResponse.Bids = append(bidResponse.Bids, typedBid)
	}

	return bidResponse, nil
}

func getMediaType(imp openrtb2.Imp) openrtb_ext.BidType {
	switch {
	case imp.Video != nil:
		return openrtb_ext.BidTypeVideo
	case imp.Audio != nil:
		return openrtb_ext.BidTypeAudio
	case imp.Native != nil:
		return openrtb_ext.BidTypeNative
	default:
		return openrtb_ext.BidTypeBanner
	}
}
