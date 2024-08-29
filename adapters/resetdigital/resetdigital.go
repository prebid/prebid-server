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
	Site resetDigitalSite  `json:"site"`
	Imps []resetDigitalImp `json:"imps"`
}
type resetDigitalSite struct {
	Domain   string `json:"domain"`
	Referrer string `json:"referrer"`
}
type resetDigitalImp struct {
	ForceBid   bool                   `json:"force_bid"`
	ZoneID     resetDigitalImpZone    `json:"zone_id"`
	BidID      string                 `json:"bid_id"`
	ImpID      string                 `json:"imp_id"`
	Ext        resetDigitalImpExt     `json:"ext"`
	MediaTypes resetDigitalMediaTypes `json:"media_types"`
}
type resetDigitalImpZone struct {
	PlacementID string `json:"placementId"`
}
type resetDigitalImpExt struct {
	Gpid string `json:"gpid"`
}
type resetDigitalMediaTypes struct {
	Banner resetDigitalMediaType `json:"banner"`
	Video  resetDigitalMediaType `json:"video"`
}
type resetDigitalMediaType struct {
	Sizes [][]int64 `json:"sizes"`
}

type resetDigitalBidResponse struct {
	Bids []resetDigitalBid `json:"bids"`
}
type resetDigitalBid struct {
	BidID string  `json:"bid_id"`
	ImpID string  `json:"imp_id"`
	CPM   float64 `json:"cpm"`
	CID   string  `json:"cid,omitempty"`
	CrID  string  `json:"crid,omitempty"`
	AdID  string  `json:"adid"`
	W     string  `json:"w,omitempty"`
	H     string  `json:"h,omitempty"`
	Seat  string  `json:"seat"`
	HTML  string  `json:"html"`
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

	if request != nil && request.Device != nil && request.Site != nil {
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

func (a *adapter) MakeRequests(requestData *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var (
		requests []*adapters.RequestData
		errors   []error
	)

	for _, imp := range requestData.Imp {
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

	return requests, errors
}

func processDataFromRequest(requestData *openrtb2.BidRequest, imp openrtb2.Imp, bidType openrtb_ext.BidType) (resetDigitalRequest, error) {
	var reqData resetDigitalRequest

	if requestData.Site != nil {
		reqData.Site.Domain = requestData.Site.Domain
		reqData.Site.Referrer = requestData.Site.Page
	}

	reqData.Imps = append(reqData.Imps, resetDigitalImp{
		BidID: requestData.ID,
		ImpID: imp.ID,
	})

	if bidType == openrtb_ext.BidTypeBanner && imp.Banner != nil {
		tempH := *imp.Banner.H
		tempW := *imp.Banner.W

		if tempH > 0 && tempW > 0 {
			reqData.Imps[0].MediaTypes.Banner.Sizes = append(
				reqData.Imps[0].MediaTypes.Banner.Sizes,
				[]int64{tempH, tempW},
			)
		}
	}
	if bidType == openrtb_ext.BidTypeVideo && imp.Video != nil {
		tempH := *imp.Video.H
		tempW := *imp.Video.W

		if tempH > 0 && tempW > 0 {
			reqData.Imps[0].MediaTypes.Video.Sizes = append(
				reqData.Imps[0].MediaTypes.Video.Sizes,
				[]int64{tempH, tempW},
			)
		}
	}

	var bidderExt adapters.ExtImpBidder
	var resetDigitalExt openrtb_ext.ImpExtResetDigital

	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return resetDigitalRequest{}, err
	}
	if err := json.Unmarshal(bidderExt.Bidder, &resetDigitalExt); err != nil {
		return resetDigitalRequest{}, err
	}
	reqData.Imps[0].ZoneID.PlacementID = resetDigitalExt.PlacementID

	return reqData, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response resetDigitalBidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))

	requestImps := make(map[string]openrtb2.Imp)
	for _, imp := range request.Imp {
		requestImps[imp.ID] = imp
	}

	for i := range response.Bids {
		resetDigitalBid := &response.Bids[i]

		bid := getBidFromResponse(resetDigitalBid)
		if bid == nil {
			continue
		}

		bidType, err := GetMediaTypeForImp(requestImps, bid.ImpID)
		if err != nil {
			return nil, []error{err}
		}

		b := &adapters.TypedBid{
			Bid:     bid,
			BidType: bidType,
			Seat:    openrtb_ext.BidderName(resetDigitalBid.Seat),
		}
		bidResponse.Bids = append(bidResponse.Bids, b)
	}

	if len(request.Cur) == 0 {
		bidResponse.Currency = "USD"
	}

	return bidResponse, nil
}

func getBidFromResponse(bidResponse *resetDigitalBid) *openrtb2.Bid {
	if bidResponse.CPM == 0 {
		return nil
	}

	bid := &openrtb2.Bid{
		ID:    bidResponse.BidID,
		Price: bidResponse.CPM,
		ImpID: bidResponse.ImpID,
		CID:   bidResponse.CID,
		CrID:  bidResponse.CrID,
		AdM:   bidResponse.HTML,
	}

	if i, err := strconv.ParseInt(bidResponse.W, 10, 64); err == nil && i > 0 {
		bid.W = i
	}
	if i, err := strconv.ParseInt(bidResponse.H, 10, 64); err == nil && i > 0 {
		bid.H = i
	}

	return bid
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

func GetMediaTypeForImp(reqImps map[string]openrtb2.Imp, bidImpID string) (openrtb_ext.BidType, error) {
	mediaType := openrtb_ext.BidTypeBanner

	if reqImp, ok := reqImps[bidImpID]; ok {
		if reqImp.Banner == nil && reqImp.Video != nil {
			mediaType = openrtb_ext.BidTypeVideo
		}
		return mediaType, nil
	}
	return "", fmt.Errorf("unknown media type for bid imp ID %s", bidImpID)
}
