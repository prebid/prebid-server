package utils

import (
	"net/url"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/prebid/prebid-server/v2/util/jsonutil"
	useragentutil "github.com/prebid/prebid-server/v2/util/useragentutil"
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
	AdUnitCode        string      `json:"adUnitCode"`
	GptSlotCode       string      `json:"gptSlotCode"`
	AuctionId         string      `json:"auctionId"`
	BidderCode        string      `json:"bidderCode"`
	Cpm               float64     `json:"cpm"`
	CreativeId        string      `json:"creativeId"`
	Currency          string      `json:"currency"`
	FloorData         FloorDetail `json:"floorData"`
	NetRevenue        bool        `json:"netRevenue"`
	RequestTimestamp  int64       `json:"requestTimestamp"`
	ResponseTimestamp int64       `json:"responseTimestamp"`
	Status            string      `json:"status"`
	StatusMessage     string      `json:"statusMessage"`
	TimeToRespond     int64       `json:"timeToRespond"`
	TransactionId     string      `json:"transactionId"`
	BidId             string      `json:"bidId"`
	RenderStatus      int64       `json:"renderStatus"`
	Sizes             [][]int64   `json:"sizes"`
	FloorProvider     string      `json:"floorProvider"`
	FloorFetchStatus  string      `json:"floorFetchStatus"`
	FloorLocation     string      `json:"floorLocation"`
	FloorModelVersion string      `json:"floorModelVersion"`
	FloorSkipRate     int64       `json:"floorSkipRate"`
	IsFloorSkipped    bool        `json:"isFloorSkipped"`
	IsWinningBid      bool        `json:"isWinningBid"`
	PlacementId       float64     `json:"placementId"`
	RenderedSize      string      `json:"renderedSize"`
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

type UtilsService interface {
	ExtractUserIds(requestExt map[string]interface{}) UserDetail
	ExtractConsentTypes(requestExt map[string]interface{}) ConsentDetail
	ExtractDeviceData(requestExt map[string]interface{}) DeviceDetail
	ExtractPageData(requestExt map[string]interface{}) PageDetail
	ExtractFloorData(requestExt map[string]interface{}, bidResponse map[string]interface{}) FloorDetail
	ExtractAdunitCodes(requestExt map[string]interface{}) []string
	UnmarshalExtensions(ao *LogObject) (map[string]interface{}, map[string]interface{}, error)
	unmarshalBidAndImpExt(bid openrtb2.Bid, imp openrtb2.Imp) (map[string]interface{}, map[string]interface{}, error)
	ProcessBidResponses(bidResponses []map[string]interface{}, auctionId string, startTime int64, requestExt, responseExt map[string]interface{}, floorDetail FloorDetail) ([]Bid, []Bid)
	createBidObject(bid openrtb2.Bid, bidExt map[string]interface{}, imp openrtb2.Imp, impExt map[string]interface{}, auctionId, bidderName string, bidderInfo map[string]interface{}, startTime int64, bidderResponsetime float64, floorDetail FloorDetail) Bid
	createWinningBidObject(bidObj Bid, impExt, bidExt map[string]interface{}, bidderName string, floorDetail FloorDetail) Bid
	isWinningBid(bidderName string, bidExt map[string]interface{}) bool
}

type UtilsServiceImpl struct {
	publisherId string
}

func NewUtilsService(publisherId string) UtilsService {
	return &UtilsServiceImpl{
		publisherId: publisherId,
	}
}

func (u *UtilsServiceImpl) ExtractUserIds(requestExt map[string]interface{}) UserDetail {
	userExt, ok := requestExt["user"].(map[string]interface{})
	if !ok {
		return UserDetail{}
	}

	ext, ok := userExt["ext"].(map[string]interface{})
	if !ok {
		return UserDetail{}
	}

	eids, ok := ext["eids"].([]interface{})
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

func (u *UtilsServiceImpl) ExtractConsentTypes(requestExt map[string]interface{}) ConsentDetail {
	regs, ok := requestExt["regs"].(map[string]interface{})
	if !ok {
		return ConsentDetail{}
	}

	consent, ok := regs["ext"].(map[string]interface{})
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

func (u *UtilsServiceImpl) ExtractDeviceData(requestExt map[string]interface{}) DeviceDetail {
	var deviceDetail DeviceDetail

	deviceExt, ok := requestExt["device"].(map[string]interface{})
	if !ok {
		return deviceDetail
	}

	userAgent, ok := deviceExt["ua"].(string)
	if !ok {
		return deviceDetail
	}

	deviceDetail.Browser = useragentutil.GetBrowser(userAgent)
	deviceDetail.DeviceOS = useragentutil.GetOS(userAgent)
	deviceDetail.DeviceType = useragentutil.GetDeviceType(userAgent)

	return deviceDetail
}

func (u *UtilsServiceImpl) ExtractPageData(requestExt map[string]interface{}) PageDetail {
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

func (u *UtilsServiceImpl) ExtractFloorData(requestExt map[string]interface{}, bidResponse map[string]interface{}) FloorDetail {
	floorDetail := FloorDetail{}

	ext := getMap(requestExt, "ext")
	prebidExt := getMap(ext, "prebid")
	floors := getMap(prebidExt, "floors")
	floorData := getMap(floors, "data")
	modelGroups := getSlice(floorData, "modelgroups")

	bidExt, _, err := u.unmarshalBidAndImpExt(bidResponse["bid"].(openrtb2.Bid), bidResponse["imp"].(openrtb2.Imp))
	if err != nil {
		return floorDetail
	}

	bidPrebid := getMap(bidExt, "prebid")
	bidFloors := getMap(bidPrebid, "floors")

	var matchingModelGroup map[string]interface{}
	floorRule := getString(bidFloors, "floorRule")
	floorRuleValue := getFloat64(bidFloors, "floorRuleValue")

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

func (u *UtilsServiceImpl) ExtractAdunitCodes(requestExt map[string]interface{}) []string {
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
func (u *UtilsServiceImpl) UnmarshalExtensions(ao *LogObject) (map[string]interface{}, map[string]interface{}, error) {
	var requestExt map[string]interface{}
	var responseExt map[string]interface{}

	data, err := jsonutil.Marshal(ao.RequestWrapper)
	if err != nil {
		return nil, nil, err
	}

	err = jsonutil.Unmarshal(data, &requestExt)
	if err != nil {
		return nil, nil, err
	}

	err = jsonutil.Unmarshal(ao.Response.Ext, &responseExt)
	if err != nil {
		return nil, nil, err
	}

	return requestExt, responseExt, nil
}

func (u *UtilsServiceImpl) ProcessBidResponses(bidResponses []map[string]interface{}, auctionId string, startTime int64, requestExt, responseExt map[string]interface{}, floorDetail FloorDetail) ([]Bid, []Bid) {
	var auctionBids []Bid
	var winningBids []Bid

	for _, bidData := range bidResponses {
		bidderName := bidData["bidder"].(string)
		bid := bidData["bid"].(openrtb2.Bid)
		imp := bidData["imp"].(openrtb2.Imp)

		bidExt, impExt, err := u.unmarshalBidAndImpExt(bid, imp)
		if err != nil {
			glog.Errorf("[pubxai] Error unmarshalling ext: %v", err)
			continue
		}

		bidderInfo := bidExt[bidderName].(map[string]interface{})
		bidderResponsetime := responseExt["responsetimemillis"].(map[string]interface{})[bidderName].(float64)

		bidObj := u.createBidObject(bid, bidExt, imp, impExt, auctionId, bidderName, bidderInfo, startTime, bidderResponsetime, floorDetail)

		auctionBids = append(auctionBids, bidObj)

		if u.isWinningBid(bidderName, bidExt) {
			winningBidObj := u.createWinningBidObject(bidObj, impExt, bidExt, bidderName, floorDetail)
			winningBids = append(winningBids, winningBidObj)
		}
	}

	return auctionBids, winningBids
}

func (u *UtilsServiceImpl) unmarshalBidAndImpExt(bid openrtb2.Bid, imp openrtb2.Imp) (map[string]interface{}, map[string]interface{}, error) {
	var bidExt map[string]interface{}
	var impExt map[string]interface{}

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

func (u *UtilsServiceImpl) createBidObject(bid openrtb2.Bid, bidExt map[string]interface{}, imp openrtb2.Imp, impExt map[string]interface{}, auctionId, bidderName string, bidderInfo map[string]interface{}, startTime int64, bidderResponsetime float64, floorDetail FloorDetail) Bid {
	gptSlotCode, ok := impExt["gpid"].(string)
	if !ok {
		gptSlotCode = ""
	}
	bidObj := Bid{
		AdUnitCode:        bid.ImpID,
		BidId:             bid.ID,
		GptSlotCode:       gptSlotCode,
		AuctionId:         auctionId,
		BidderCode:        bidderName,
		Cpm:               bidExt["origbidcpm"].(float64),
		CreativeId:        bid.CrID,
		Currency:          bidExt["origbidcur"].(string),
		FloorData:         floorDetail,
		NetRevenue:        true,
		RequestTimestamp:  startTime,
		ResponseTimestamp: startTime + int64(bidderResponsetime),
		Status:            "targetingSet",
		StatusMessage:     "Bid available",
		TimeToRespond:     int64(bidderResponsetime),
		TransactionId:     impExt["tid"].(string),
		RenderStatus:      2,
	}

	for _, format := range imp.Banner.Format {
		bidObj.Sizes = append(bidObj.Sizes, []int64{format.W, format.H})
	}

	return bidObj
}

func (u *UtilsServiceImpl) createWinningBidObject(bidObj Bid, impExt, bidExt map[string]interface{}, bidderName string, floorDetail FloorDetail) Bid {
	bidObj.IsWinningBid = true
	bidObj.RenderStatus = 4
	bidObj.Status = "rendered"
	bidObj.PlacementId = impExt["prebid"].(map[string]interface{})["bidder"].(map[string]interface{})[bidderName].(map[string]interface{})["placement_id"].(float64)
	bidObj.RenderedSize = bidExt["prebid"].(map[string]interface{})["targeting"].(map[string]interface{})["hb_size"].(string)

	bidObj.FloorProvider = floorDetail.FloorProvider
	bidObj.FloorFetchStatus = floorDetail.FetchStatus
	bidObj.FloorLocation = floorDetail.Location
	bidObj.FloorModelVersion = floorDetail.ModelVersion
	bidObj.FloorSkipRate = floorDetail.SkipRate
	bidObj.IsFloorSkipped = floorDetail.Skipped

	return bidObj
}

// if hb_pb is present in bidExt.prebid.targeting and bidderName matches with hb_bidder
func (u *UtilsServiceImpl) isWinningBid(bidderName string, bidExt map[string]interface{}) bool {
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

	if hbPbOk && hbBidderOk && hbPb != "" && bidderName == hbBidder {
		return true
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
