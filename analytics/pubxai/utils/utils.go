package utils

import (
	"errors"
	"net/url"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	useragentutil "github.com/prebid/prebid-server/v3/util/useragentutil"
)

type LogObject struct {
	Status         int
	Errors         []error
	Response       *openrtb2.BidResponse
	StartTime      time.Time
	SeatNonBid     []openrtb_ext.SeatNonBid
	RequestWrapper *openrtb_ext.RequestWrapper
}

type Bid struct {
	AdUnitCode        string                 `json:"adUnitCode"`
	GptSlotCode       string                 `json:"gptSlotCode"`
	AuctionId         string                 `json:"auctionId"`
	BidderCode        string                 `json:"bidderCode"`
	Cpm               float64                `json:"cpm"`
	CreativeId        string                 `json:"creativeId"`
	Currency          string                 `json:"currency"`
	FloorData         map[string]interface{} `json:"floorData"`
	NetRevenue        bool                   `json:"netRevenue"`
	RequestTimestamp  int64                  `json:"requestTimestamp"`
	ResponseTimestamp int64                  `json:"responseTimestamp"`
	Status            string                 `json:"status"`
	StatusMessage     string                 `json:"statusMessage"`
	TimeToRespond     int64                  `json:"timeToRespond"`
	TransactionId     string                 `json:"transactionId"`
	BidId             string                 `json:"bidId"`
	BidType           int64                  `json:"renderStatus"`
	Sizes             [][]int64              `json:"sizes"`
	FloorProvider     string                 `json:"floorProvider"`
	FloorFetchStatus  string                 `json:"floorFetchStatus"`
	FloorLocation     string                 `json:"floorLocation"`
	FloorModelVersion string                 `json:"floorModelVersion"`
	FloorSkipRate     int64                  `json:"floorSkipRate"`
	IsFloorSkipped    bool                   `json:"isFloorSkipped"`
	IsWinningBid      bool                   `json:"IsWinningBid"`
	PlacementId       float64                `json:"placementId"`
	RenderedSize      string                 `json:"renderedSize"`
}

type AuctionBids struct {
	AuctionDetail AuctionDetail          `json:"auctionDetail"`
	FloorDetail   FloorDetail            `json:"floorDetail"`
	PageDetail    PageDetail             `json:"pageDetail"`
	DeviceDetail  DeviceDetail           `json:"deviceDetail"`
	UserDetail    UserDetail             `json:"userDetail"`
	ConsentDetail ConsentDetail          `json:"consentDetail"`
	PmacDetail    map[string]interface{} `json:"pmacDetail"`
	InitOptions   InitOptions            `json:"initOptions"`
	Bids          []Bid                  `json:"bids"`
	Source        string                 `json:"source"`
}

type WinningBid struct {
	AuctionDetail AuctionDetail          `json:"auctionDetail"`
	FloorDetail   FloorDetail            `json:"floorDetail"`
	PageDetail    PageDetail             `json:"pageDetail"`
	DeviceDetail  DeviceDetail           `json:"deviceDetail"`
	UserDetail    UserDetail             `json:"userDetail"`
	ConsentDetail ConsentDetail          `json:"consentDetail"`
	PmacDetail    map[string]interface{} `json:"pmacDetail"`
	InitOptions   InitOptions            `json:"initOptions"`
	WinningBid    Bid                    `json:"winningBid"`
	Source        string                 `json:"source"`
}

type PageDetail struct {
	Host   string `json:"host"`
	Path   string `json:"path"`
	Search string `json:"search"`
}

type DeviceDetail struct {
	Platform   string `json:"platform"`
	DeviceType int    `json:"deviceType"`
	DeviceOS   int    `json:"deviceOS"`
	Browser    int    `json:"browser"`
}

type UserDetail struct {
	UserIdTypes []string `json:"userIdTypes"`
}

type ConsentDetail struct {
	ConsentTypes []string `json:"consentTypes"`
}

type InitOptions struct {
	AuctionId    string `json:"auctionId"`
	SamplingRate int    `json:"samplingRate"`
	Pubxid       string `json:"pubxid"`
}

type FloorDetail struct {
	FetchStatus     string `json:"fetchStatus"`
	FloorProvider   string `json:"floorProvider"`
	Location        string `json:"location"`
	ModelVersion    string `json:"modelVersion"`
	NoFloorSignaled bool   `json:"noFloorSignaled"`
	SkipRate        int64  `json:"skipRate"`
	Skipped         bool   `json:"skipped"`
	SkippedReason   string `json:"skippedReason"`
}

type AuctionDetail struct {
	AdUnitCodes []string `json:"adUnitCodes"`
	RefreshRank int64    `json:"refreshRank"`
	AuctionId   string   `json:"auctionId"`
	Timestamp   int64    `json:"timestamp"`
}

func ExtractUserIds(requestExt map[string]interface{}) UserDetail {

	eidsInterface, ok := nestedMapLookup(requestExt, "user", "ext", "eids")
	if !ok {
		return UserDetail{}
	}

	eids, ok := eidsInterface.([]interface{})
	if !ok {
		return UserDetail{}
	}

	var userIds []string
	for _, eid := range eids {
		if eidMap, ok := eid.(map[string]interface{}); ok {
			if source, ok := eidMap["source"].(string); ok {
				userIds = append(userIds, source)
			}
		}
	}

	return UserDetail{
		UserIdTypes: userIds,
	}
}

func ExtractConsentTypes(requestExt map[string]interface{}) ConsentDetail {

	consentInterface, ok := nestedMapLookup(requestExt, "regs", "ext")
	if !ok {
		return ConsentDetail{}
	}

	consent, ok := consentInterface.(map[string]interface{})
	if !ok {
		return ConsentDetail{}
	}

	var consentTypes []string
	for key := range consent {
		consentTypes = append(consentTypes, key)
	}
	return ConsentDetail{
		ConsentTypes: consentTypes,
	}
}

func ExtractDeviceData(requestExt map[string]interface{}) DeviceDetail {
	var deviceDetail DeviceDetail

	userAgentInterface, ok := nestedMapLookup(requestExt, "device", "ua")
	if !ok {
		return deviceDetail
	}

	userAgent, ok := userAgentInterface.(string)
	if !ok {
		return deviceDetail
	}

	deviceDetail.Browser = useragentutil.GetBrowser(userAgent)
	deviceDetail.DeviceOS = useragentutil.GetOS(userAgent)
	deviceDetail.DeviceType = useragentutil.GetDeviceType(userAgent)
	return deviceDetail
}

func ExtractPageData(requestExt map[string]interface{}) PageDetail {
	var pageDetail PageDetail

	siteExt, ok := requestExt["site"].(map[string]interface{})
	if !ok {
		return pageDetail
	}

	if domain, ok := siteExt["domain"].(string); ok {
		pageDetail.Host = domain
	} else {
		return pageDetail
	}

	if fullUrl, ok := siteExt["page"].(string); ok {
		parsedURL, err := url.Parse(fullUrl)
		if err != nil {
			pageDetail.Path = ""
			return pageDetail
		}
		pageDetail.Path = parsedURL.RequestURI()
	}

	return pageDetail
}

func ExtractFloorDetail(requestExt map[string]interface{}) FloorDetail {
	floorDetail := FloorDetail{}

	ext := getMap(requestExt, "ext")
	prebidExt := getMap(ext, "prebid")
	floors := getMap(prebidExt, "floors")
	floorData := getMap(floors, "data")
	modelGroups := getSlice(floorData, "modelgroups")

	imps, ok := requestExt["imp"].([]interface{})

	if !ok {
		return floorDetail
	}
	imp, ok := imps[0].(map[string]interface{})
	if !ok {
		return floorDetail
	}
	bidFloorsInterface, ok := nestedMapLookup(imp, "ext", "prebid", "floors")
	if !ok {
		return floorDetail
	}
	bidFloors, ok := bidFloorsInterface.(map[string]interface{})
	if !ok {
		return floorDetail
	}

	var matchingModelGroup map[string]interface{}
	floorRule := getString(bidFloors, "floorrule")
	floorRuleValue := getFloat64(bidFloors, "floorrulevalue")

	for _, modelGroup := range modelGroups {
		modelgroup := modelGroup.(map[string]interface{})
		values := getMap(modelgroup, "values")

		if floorValue, exists := values[floorRule]; exists && floorValue == floorRuleValue {
			matchingModelGroup = modelgroup
			break
		}
	}

	if matchingModelGroup == nil {
		return floorDetail
	}

	floorDetail.FloorProvider = getString(floorData, "floorprovider")
	floorDetail.ModelVersion = getString(matchingModelGroup, "modelversion")
	floorDetail.SkipRate = getInt64(matchingModelGroup, "skiprate")
	floorDetail.FetchStatus = getString(floors, "fetchstatus")
	floorDetail.Location = getString(floors, "location")
	floorDetail.Skipped = getBool(floors, "skipped")

	return floorDetail
}

func ExtractAdunitCodes(requestExt map[string]interface{}) []string {
	var adunitCodes []string
	imps, ok := requestExt["imp"].([]interface{})
	if !ok {
		return adunitCodes
	}

	for _, imp := range imps {
		if impMap, ok := imp.(map[string]interface{}); ok {
			if id, ok := impMap["id"].(string); ok {
				adunitCodes = append(adunitCodes, id)
			}
		}
	}

	return adunitCodes
}
func UnmarshalExtensions(ao *LogObject) (map[string]interface{}, map[string]interface{}, error) {
	var requestExt map[string]interface{}
	var responseExt map[string]interface{}

	if ao.RequestWrapper == nil {
		return nil, nil, errors.New("request wrapper is nil")
	}

	data, err := jsonutil.Marshal(ao.RequestWrapper)
	if err != nil {
		glog.Errorf("[pubxai] Error unmarshalling extensions: %v", err)
		return nil, nil, err
	}

	err = jsonutil.Unmarshal(data, &requestExt)
	if err != nil {
		glog.Errorf("[pubxai] Error unmarshalling extensions: %v", err)
		return nil, nil, err
	}

	err = jsonutil.Unmarshal(ao.Response.Ext, &responseExt)
	if err != nil {
		glog.Errorf("[pubxai] Error unmarshalling extensions: %v", err)
		return requestExt, nil, nil
	}

	return requestExt, responseExt, nil
}

func ProcessBidResponses(bidResponses []map[string]interface{}, auctionId string, startTime int64, requestExt, responseExt map[string]interface{}, floorDetail FloorDetail) ([]Bid, []Bid) {
	var auctionBids []Bid
	var winningBids []Bid

	for _, bidData := range bidResponses {
		bidderName := bidData["bidder"].(string)

		bidExt, impExt, err := unmarshalBidAndImpExt(bidData)
		if err != nil {
			glog.Errorf("[pubxai] Error unmarshalling ext: %v", err)
			continue
		}
		bid := bidData["bid"].(openrtb2.Bid)
		imp := bidData["imp"].(openrtb2.Imp)
		bidderResponsetimeInterface, ok := nestedMapLookup(responseExt, "responsetimemillis", bidderName)
		if !ok {
			return nil, nil
		}
		bidderResponsetime, ok := bidderResponsetimeInterface.(float64)
		if !ok {
			return nil, nil
		}

		bidObj := createBidObject(&bid, bidExt, imp, impExt, auctionId, bidderName, startTime, bidderResponsetime)

		auctionBids = append(auctionBids, bidObj)

		if isWinningBid(bidderName, bidExt) {
			winningBidObj := createWinningBidObject(bidObj, impExt, bidExt, bidderName, floorDetail)
			winningBids = append(winningBids, winningBidObj)
		}
	}

	return auctionBids, winningBids
}

func AppendTimeoutBids(auctionBids []Bid, impsById map[string]openrtb2.Imp, ao *LogObject) []Bid {

	requestExt, _, err := UnmarshalExtensions(ao)
	if err != nil {
		return auctionBids
	}
	imp, ok := requestExt["imp"].([]interface{})
	if !ok {
		return auctionBids
	} else if len(imp) == 0 {
		return auctionBids
	}

	for id, imp := range impsById {
		var impExt map[string]interface{}
		err := jsonutil.Unmarshal(imp.Ext, &impExt)
		if err != nil {
			continue
		}
		bidderInterface, ok := nestedMapLookup(impExt, "prebid", "bidder")
		if !ok {
			continue
		}
		bidders, _ := bidderInterface.(map[string]interface{})

		for bidder := range bidders {
			if !hasBidResponse(auctionBids, bidder, id) {
				auctionBids = append(auctionBids, createTimedOutBid(imp, impExt, requestExt, ao, bidder))
			}
		}

	}
	return auctionBids
}
func extractFloorData(bidExt map[string]interface{}) map[string]interface{} {
	bidFloors, ok := nestedMapLookup(bidExt, "prebid", "floors")

	if !ok {
		return nil
	}
	floorData, ok := bidFloors.(map[string]interface{})

	if !ok {
		return nil
	}
	return floorData
}
func unmarshalBidAndImpExt(bidData map[string]interface{}) (map[string]interface{}, map[string]interface{}, error) {
	var bidExt map[string]interface{}
	var impExt map[string]interface{}

	bid, bidOk := bidData["bid"].(openrtb2.Bid)
	imp, impOk := bidData["imp"].(openrtb2.Imp)

	if !bidOk || !impOk {
		return nil, nil, errors.New("invalid bidData")
	}
	err := jsonutil.Unmarshal(imp.Ext, &impExt)
	if err != nil {
		return nil, nil, err
	}

	err = jsonutil.Unmarshal(bid.Ext, &bidExt)
	if err != nil {
		return nil, nil, err
	}

	return bidExt, impExt, nil
}
func getAdunitCodeAndGptSlot(impExt map[string]interface{}) (string, string) {
	adUnitCode, ok := nestedMapLookup(impExt, "data", "pbadslot")
	if !ok {
		adUnitCode = ""
	}

	gptSlot, ok := nestedMapLookup(impExt, "data", "adserver", "adslot")
	if !ok {
		gptSlot = ""
	}
	return adUnitCode.(string), gptSlot.(string)
}

func getResponseBidInfo(bid *openrtb2.Bid) (string, string, int64, bool, string) {
	if bid == nil {
		return "", "", 3, false, "Bid Timeout"
	}
	return bid.ID, bid.CrID, 2, true, "Bid available"

}
func createBidObject(bid *openrtb2.Bid, bidExt map[string]interface{}, imp openrtb2.Imp, impExt map[string]interface{}, auctionId, bidderName string, startTime int64, bidderResponsetime float64) Bid {
	adUnitCode, gptSlot := getAdunitCodeAndGptSlot(impExt)
	bidId, creativeId, bidType, netRevenue, statusMessage := getResponseBidInfo(bid)
	cpm, _ := bidExt["origbidcpm"].(float64)
	currency, _ := bidExt["origbidcur"].(string)
	tid, _ := impExt["tid"].(string)

	bidObj := Bid{
		AdUnitCode:        adUnitCode,
		BidId:             bidId,
		GptSlotCode:       gptSlot,
		AuctionId:         auctionId,
		BidderCode:        bidderName,
		Cpm:               cpm,
		CreativeId:        creativeId,
		Currency:          currency,
		FloorData:         extractFloorData(bidExt),
		NetRevenue:        netRevenue,
		RequestTimestamp:  startTime,
		ResponseTimestamp: startTime + int64(bidderResponsetime),
		Status:            "targetingSet",
		StatusMessage:     statusMessage,
		TimeToRespond:     int64(bidderResponsetime),
		TransactionId:     tid,
		BidType:           bidType,
	}

	for _, format := range imp.Banner.Format {
		bidObj.Sizes = append(bidObj.Sizes, []int64{format.W, format.H})
	}

	return bidObj
}

func createWinningBidObject(bidObj Bid, impExt, bidExt map[string]interface{}, bidderName string, floorDetail FloorDetail) Bid {
	if placementInterface, ok := nestedMapLookup(impExt, "prebid", "bidder", bidderName, "placement_id"); ok {
		bidObj.PlacementId, _ = placementInterface.(float64)
	}

	if renderSizeInterface, ok := nestedMapLookup(bidExt, "prebid", "targeting", "hb_size"); ok {
		bidObj.RenderedSize, _ = renderSizeInterface.(string)
	}

	bidObj.IsWinningBid = true
	bidObj.BidType = 4
	bidObj.Status = "rendered"
	bidObj.FloorProvider = floorDetail.FloorProvider
	bidObj.FloorFetchStatus = floorDetail.FetchStatus
	bidObj.FloorLocation = floorDetail.Location
	bidObj.FloorModelVersion = floorDetail.ModelVersion
	bidObj.FloorSkipRate = floorDetail.SkipRate
	bidObj.IsFloorSkipped = floorDetail.Skipped

	return bidObj
}

// if hb_pb is present in bidExt.prebid.targeting and bidderName matches with hb_bidder
func isWinningBid(bidderName string, bidExt map[string]interface{}) bool {
	prebid, ok := bidExt["prebid"].(map[string]interface{})
	if !ok {
		return false
	}

	targeting, ok := prebid["targeting"].(map[string]interface{})
	if !ok {
		return false
	}

	hbPb, hbPbOk := targeting["hb_pb"].(string)
	hbBidder, hbBidderOk := targeting["hb_bidder"].(string)

	return hbPbOk && hbPb != "" && hbBidderOk && bidderName == hbBidder
}

func createTimedOutBid(imp openrtb2.Imp, impExt map[string]interface{}, requestExt map[string]interface{}, ao *LogObject, bidder string) Bid {
	return createBidObject(nil, nil, imp, impExt, requestExt["id"].(string), bidder, ao.StartTime.UTC().UnixMilli(), 0.0)
}

func hasBidResponse(auctionBids []Bid, bidder string, adunitCode string) bool {

	for _, bid := range auctionBids {
		if bid.AdUnitCode == adunitCode && bid.BidderCode == bidder {
			return true
		}
	}
	return false
}
func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if val, ok := m[key].(map[string]interface{}); ok {
		return val
	}
	return nil
}

func getSlice(m map[string]interface{}, key string) []interface{} {
	if val, ok := m[key].([]interface{}); ok {
		return val
	}
	return nil
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getInt64(m map[string]interface{}, key string) int64 {
	if val, ok := m[key].(int64); ok {
		return val
	}
	return 0
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return false
}

func nestedMapLookup(m map[string]interface{}, keys ...string) (interface{}, bool) {
	current := interface{}(m)
	for _, key := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			if value, ok := m[key]; ok {
				current = value
			} else {
				return nil, false
			}
		} else {
			return nil, false
		}
	}
	return current, true
}
