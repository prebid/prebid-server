package huaweiads

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	nativeRequests "github.com/mxmCherry/openrtb/v15/native1/request"
	nativeResponse "github.com/mxmCherry/openrtb/v15/native1/response"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const HuaweiAdxApiVersion string = "3.4"
const DefaultBidRequestArrayLength int = 1
const DefaultCountryName = "ZA"
const DefaultUnknownNetworkType = 0
const TimeFormat = "2006-01-02 15:04:05.000"
const DefaultModelName = "HUAWEI"

// creative type
const (
	text        int32 = 1
	bigPicture  int32 = 2
	bigPicture2 int32 = 3
	gif         int32 = 4
	// used for magazine lock screen
	noSpecificCreativeType int32 = 5
	videoText              int32 = 6
	smallPicture           int32 = 7
	threeSmallPicturesText int32 = 8
	video                  int32 = 9
	iconText               int32 = 10
	// RewardedVideo only
	videoWithPicturesText int32 = 11
	windowAdvertisement   int32 = 13
)

// ads events
const (
	click         string = "click"
	imp           string = "imp"
	userClose     string = "userclose"
	download      string = "download"
	install       string = "install"
	downloadstart string = "downloadstart"
	playStart     string = "playStart"
	playEnd       string = "playEnd"
	playResume    string = "playResume"
	playPause     string = "playPause"
	appOpen       string = "appOpen"
)

// ads type
const (
	banner        int32 = 8
	native        int32 = 3
	rewardedVideo int32 = 7
	splash        int32 = 1
	interstitial  int32 = 12
	roll          int32 = 60
)

type HuaweiAdsAdapter struct {
	endpoint string
}

func (a *HuaweiAdsAdapter) MakeRequests(openRTBRequest *openrtb2.BidRequest,
	reqInfo *adapters.ExtraRequestInfo) (requestsToBidder []*adapters.RequestData, errs []error) {
	if openRTBRequest == nil {
		errs = append(errs, &errortypes.BadInput{
			Message: "MakeRequests: openRTBRequest is nil",
		})
		return nil, errs
	}

	numRequests := len(openRTBRequest.Imp)
	if numRequests == 0 || openRTBRequest.Imp == nil {
		errs = append(errs, &errortypes.BadInput{
			Message: "MakeRequests: No impression in the bid request",
		})
		return nil, errs
	}

	var request HuaweiAdsRequest
	var header http.Header
	var multislot = make([]Adslot30, 0, numRequests)

	var huaweiAdsImpExt *openrtb_ext.ExtImpHuaweiAds
	var err1 error
	for index := 0; index < numRequests; index++ {
		huaweiAdsImpExt, err1 = unmarshalExtImpHuaweiAds(&openRTBRequest.Imp[index])

		if err1 != nil {
			errs = append(errs, err1)
			return nil, errs
		}

		adslot30, err := getHuaweiAdsReqAdslot30(huaweiAdsImpExt, &openRTBRequest.Imp[index], openRTBRequest)
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}

		multislot = append(multislot, adslot30)
	}
	request.Multislot = multislot

	if err := getHuaweiAdsReqJson(&request, openRTBRequest, huaweiAdsImpExt); err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	//	our request header's Authorization is changing by time, cannot verify by a certain string,
	//	use isAddAuthorization = false only when run testcase,
	//	isAddAuthorization = true when we request for a new ad from Huawei Ads ADX.
	var isAddAuthorization = true
	if huaweiAdsImpExt.IsAddAuthorization == "false" {
		isAddAuthorization = false
	}
	header = getHeaders(huaweiAdsImpExt, openRTBRequest, isAddAuthorization)
	bidRequest := &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: header,
	}

	var bidRequestArray = make([]*adapters.RequestData, 0, DefaultBidRequestArrayLength)
	bidRequestArray = append(bidRequestArray, bidRequest)

	return bidRequestArray, errs
}

func (a *HuaweiAdsAdapter) MakeBids(openRTBRequest *openrtb2.BidRequest, requestToBidder *adapters.RequestData,
	bidderRawResponse *adapters.ResponseData) (bidderResponse *adapters.BidderResponse, errs []error) {
	if bidderRawResponse.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	httpStatusError := checkRespStatusCode(bidderRawResponse)
	if httpStatusError != nil {
		return nil, []error{httpStatusError}
	}

	var huaweiAdsResponse HuaweiAdsResponse
	if err := json.Unmarshal(bidderRawResponse.Body, &huaweiAdsResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Bad Server Response",
		}}
	}

	huaweiAdsResponseError := checkHuaweiAdsResponseRetcode(huaweiAdsResponse)
	if huaweiAdsResponseError != nil {
		return nil, []error{huaweiAdsResponseError}
	}

	var err error
	bidderResponse, err = convertHuaweiAdsResp2BidderResp(&huaweiAdsResponse, openRTBRequest)
	if err != nil {
		return nil, []error{err}
	}

	return bidderResponse, nil
}

// Builder builds a new instance of the HuaweiAds adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &HuaweiAdsAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

// get request header
func getHeaders(huaweiAdsImpExt *openrtb_ext.ExtImpHuaweiAds, request *openrtb2.BidRequest, isAddAuthorization bool) http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	if isAddAuthorization {
		headers.Add("Authorization", getDigestAuthorization(huaweiAdsImpExt))
	}

	if request.Device != nil && len(request.Device.UA) > 0 {
		headers.Add("User-Agent", request.Device.UA)
	}

	return headers
}

// get body json for HuaweiAds request
func getHuaweiAdsReqJson(request *HuaweiAdsRequest, openRTBRequest *openrtb2.BidRequest, huaweiAdsImpExt *openrtb_ext.ExtImpHuaweiAds) error {
	request.Version = HuaweiAdxApiVersion
	var err error
	if err = getHuaweiAdsReqAppInfo(request, openRTBRequest); err != nil {
		return err
	}
	if err = getHuaweiAdsReqDeviceInfo(request, openRTBRequest, huaweiAdsImpExt); err != nil {
		return err
	}
	getHuaweiAdsReqNetWorkInfo(request, openRTBRequest)
	getHuaweiAdsReqRegsInfo(request, openRTBRequest)
	getHuaweiAdsReqGeoInfo(request, openRTBRequest)
	return nil
}

// get adslot30
func getHuaweiAdsReqAdslot30(huaweiAdsImpExt *openrtb_ext.ExtImpHuaweiAds,
	openRTBImp *openrtb2.Imp, openRTBRequest *openrtb2.BidRequest) (Adslot30, error) {
	adtypeLower := strings.ToLower(huaweiAdsImpExt.Adtype)
	var adslot30 = Adslot30{
		Slotid: huaweiAdsImpExt.SlotId,
		Adtype: convertAdtypeString2Integer(adtypeLower),
		Test:   int32(openRTBRequest.Test),
	}

	if adtypeLower == "roll" {
		if openRTBImp.Video != nil {
			adslot30.TotalDuration = int32(openRTBImp.Video.MaxDuration)
		} else {
			return adslot30, errors.New("GetHuaweiAdsReqAdslot30: MaxDuration is empty when adtype is roll.")
		}
	}
	return adslot30, nil
}

// convert adtype String -> Integer
func convertAdtypeString2Integer(adtypeLower string) int32 {
	if adtypeLower == "banner" {
		return 8
	} else if adtypeLower == "native" {
		return 3
	} else if adtypeLower == "rewarded" {
		return 7
	} else if adtypeLower == "splash" {
		return 1
	} else if adtypeLower == "interstitial" {
		return 12
	} else if adtypeLower == "roll" {
		return 60
	} else {
		return 8
	}
}

// get app information for HuaweiAds request
func getHuaweiAdsReqAppInfo(request *HuaweiAdsRequest, openRTBRequest *openrtb2.BidRequest) error {
	var app App
	if openRTBRequest.App != nil {
		if openRTBRequest.App.Ver != "" {
			app.Version = openRTBRequest.App.Ver
		}
		if openRTBRequest.App.Name != "" {
			app.Name = openRTBRequest.App.Name
		}

		// bundle cannot be empty, we need package name.
		if openRTBRequest.App.Bundle != "" {
			app.Pkgname = openRTBRequest.App.Bundle
		} else {
			return errors.New("HuaweiAdsReqApp: Pkgname is empty.")
		}

		if openRTBRequest.App.Content != nil && openRTBRequest.App.Content.Language != "" {
			app.Lang = openRTBRequest.App.Content.Language
		} else {
			app.Lang = "en"
		}
	}

	if openRTBRequest.User != nil && openRTBRequest.User.Geo != nil && openRTBRequest.User.Geo.Country != "" {
		app.Country = openRTBRequest.User.Geo.Country
	} else if openRTBRequest.Device != nil && openRTBRequest.Device.Geo != nil && openRTBRequest.Device.Geo.Country != "" {
		app.Country = openRTBRequest.Device.Geo.Country
	} else {
		app.Country = DefaultCountryName
	}

	request.App = app
	return nil
}

// get field clientTime, format: 2006-01-02 15:04:05.000+2000
func getClientTime(clientTime string) (newClientTime string) {
	zone, _ := time.Now().Local().Zone()
	if isMatched, _ := regexp.MatchString("[+-]{1}\\d{2}", zone); isMatched {
		zone = zone + "00"
	} else {
		zone = "+0200"
	}

	if clientTime == "" {
		return time.Now().Format(TimeFormat) + zone
	}

	if isMatched1, _ := regexp.MatchString("^\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}.\\d{3}[+-]{1}\\d{4}$", clientTime); isMatched1 {
		return clientTime
	}
	if isMatched2, _ := regexp.MatchString("^\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}.\\d{3}$", clientTime); isMatched2 {
		return clientTime + "+0200"
	}
	return time.Now().Format(TimeFormat) + zone
}

// get device information for HuaweiAds request
func getHuaweiAdsReqDeviceInfo(request *HuaweiAdsRequest, openRTBRequest *openrtb2.BidRequest, huaweiAdsImpExt *openrtb_ext.ExtImpHuaweiAds) (err error) {
	var device Device
	if openRTBRequest.Device != nil {
		device.Type = int32(openRTBRequest.Device.DeviceType)
		device.Useragent = openRTBRequest.Device.UA
		device.Os = openRTBRequest.Device.OS
		device.Version = openRTBRequest.Device.OSV
		device.Maker = openRTBRequest.Device.Make
		device.Model = openRTBRequest.Device.Model
		if device.Model == "" {
			device.Model = DefaultModelName
		}
		device.Height = int32(openRTBRequest.Device.H)
		device.Width = int32(openRTBRequest.Device.W)
		device.Language = openRTBRequest.Device.Language
		device.Pxratio = float32(openRTBRequest.Device.PxRatio)
		device.ClientTime = getClientTime(huaweiAdsImpExt.ClientTime)

		// oaid  IsTrackingEnabled = 1 - DNT
		if device.Oaid != "" && openRTBRequest.Device.DNT != nil {
			device.IsTrackingEnabled = strconv.Itoa(1 - int(*openRTBRequest.Device.DNT))
		}
		if device.Gaid != "" && openRTBRequest.Device.DNT != nil {
			device.GaidTrackingEnabled = strconv.Itoa(1 - int(*openRTBRequest.Device.DNT))
		}

		if openRTBRequest.User != nil && openRTBRequest.User.Geo != nil && openRTBRequest.User.Geo.Country != "" {
			device.BelongCountry = openRTBRequest.User.Geo.Country
			device.LocaleCountry = openRTBRequest.User.Geo.Country
		} else if openRTBRequest.Device != nil && openRTBRequest.Device.Geo != nil && openRTBRequest.Device.Geo.Country != "" {
			device.BelongCountry = openRTBRequest.Device.Geo.Country
			device.LocaleCountry = openRTBRequest.Device.Geo.Country
		} else {
			device.BelongCountry = DefaultCountryName
			device.LocaleCountry = DefaultCountryName
		}
		device.Ip = openRTBRequest.Device.IP
	}

	// get oaid gaid imei in openRTBRequest.User.Ext.Data
	if err = getDeviceID(&device, openRTBRequest); err != nil {
		return err
	}

	request.Device = device
	return nil
}

// get device id  include oaid gaid imei.
func getDeviceID(device *Device, openRTBRequest *openrtb2.BidRequest) (err error) {
	if openRTBRequest.User == nil {
		return errors.New("getDeviceID: openRTBRequest.User is nil.")
	}
	if openRTBRequest.User.Ext == nil {
		return errors.New("getDeviceID: openRTBRequest.User.Ext is nil.")
	}
	var extUserDataHuaweiAds openrtb_ext.ExtUserDataHuaweiAds
	if err := json.Unmarshal(openRTBRequest.User.Ext, &extUserDataHuaweiAds); err != nil {
		return errors.New("Unmarshal: openRTBRequest.User.Ext -> extUserDataHuaweiAds failed")
	}
	var deviceId = extUserDataHuaweiAds.Data
	if len(deviceId.Imei) == 0 && len(deviceId.Gaid) == 0 && len(deviceId.Oaid) == 0 {
		return errors.New("getDeviceID: Imei ,Oaid, Gaid are all empty.")
	}
	if len(deviceId.Oaid) > 0 {
		device.Oaid = deviceId.Oaid[0]
	}
	if len(deviceId.Gaid) > 0 {
		device.Gaid = deviceId.Gaid[0]
	}
	if len(deviceId.Imei) > 0 {
		device.Imei = deviceId.Imei[0]
	}
	return nil
}

// get network information for HuaweiAds request
func getHuaweiAdsReqNetWorkInfo(request *HuaweiAdsRequest, openRTBRequest *openrtb2.BidRequest) {
	var network Network
	if openRTBRequest.Device != nil && openRTBRequest.Device.ConnectionType != nil {
		network.Type = int32(*openRTBRequest.Device.ConnectionType)
	} else {
		network.Type = DefaultUnknownNetworkType
	}
	request.Network = network
}

// get regs information for HuaweiAds request
func getHuaweiAdsReqRegsInfo(request *HuaweiAdsRequest, openRTBRequest *openrtb2.BidRequest) {
	var regs Regs
	if openRTBRequest.Regs != nil {
		regs.Coppa = int32(openRTBRequest.Regs.COPPA)
	}
	request.Regs = regs
}

// get geo information for HuaweiAds request
func getHuaweiAdsReqGeoInfo(request *HuaweiAdsRequest, openRTBRequest *openrtb2.BidRequest) {
	var geo Geo
	if openRTBRequest.Device != nil && openRTBRequest.Device.Geo != nil {
		geo.Lon = float32(openRTBRequest.Device.Geo.Lon)
		geo.Lat = float32(openRTBRequest.Device.Geo.Lat)
		geo.Accuracy = int32(openRTBRequest.Device.Geo.Accuracy)
		geo.Lastfix = int32(openRTBRequest.Device.Geo.LastFix)
	}
	request.Geo = geo
}

func unmarshalExtImpHuaweiAds(openRTBImp *openrtb2.Imp) (*openrtb_ext.ExtImpHuaweiAds, error) {
	var bidderExt adapters.ExtImpBidder
	var huaweiAdsImpExt openrtb_ext.ExtImpHuaweiAds
	if err := json.Unmarshal(openRTBImp.Ext, &bidderExt); err != nil {
		return nil, errors.New("Unmarshal: openRTBImp.Ext -> bidderExt failed")
	}
	if err := json.Unmarshal(bidderExt.Bidder, &huaweiAdsImpExt); err != nil {
		return nil, errors.New("Unmarshal: bidderExt.Bidder -> huaweiAdsImpExt failed")
	}
	if huaweiAdsImpExt.SlotId == "" {
		return nil, errors.New("ExtImpHuaweiAds: slotid is empty.")
	}
	if huaweiAdsImpExt.Adtype == "" {
		return nil, errors.New("ExtImpHuaweiAds: adtype is empty.")
	}
	if huaweiAdsImpExt.PublisherId == "" {
		return nil, errors.New("ExtHuaweiAds: publisherid is empty.")
	}
	if huaweiAdsImpExt.SignKey == "" {
		return nil, errors.New("ExtHuaweiAds: signkey is empty.")
	}
	if huaweiAdsImpExt.KeyId == "" {
		return nil, errors.New("ExtImpHuaweiAds: keyid is empty.")
	}
	return &huaweiAdsImpExt, nil
}

// check response status code
func checkRespStatusCode(response *adapters.ResponseData) error {
	if response.StatusCode == http.StatusNoContent {
		return nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: [ %d ]", response.StatusCode),
		}
	}

	if response.StatusCode == http.StatusServiceUnavailable {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Something went wrong, please contact your Account Manager. Status Code: [ %d ] ", response.StatusCode),
		}
	}

	if response.StatusCode != http.StatusOK {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: [ %d ]. Run with request.debug = 1 for more info", response.StatusCode),
		}
	}

	if response.Body == nil {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("bidderRawResponse body is empty"),
		}
	}
	return nil
}

// check HuaweiAds response retcode
func checkHuaweiAdsResponseRetcode(response HuaweiAdsResponse) error {
	if response.Retcode == 200 || response.Retcode == 204 || response.Retcode == 206 {
		return nil
	}

	if (response.Retcode < 600 && response.Retcode >= 400) || (response.Retcode < 300 && response.Retcode > 200) {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("HuaweiAdsResponse retcode: %d , reason: %s", response.Retcode, response.Reason),
		}
	}
	return nil
}

// convert HuaweiAds' response into bidder's response
func convertHuaweiAdsResp2BidderResp(huaweiAdsResponse *HuaweiAdsResponse, openRTBRequest *openrtb2.BidRequest) (bidderResponse *adapters.BidderResponse, err error) {
	if len(huaweiAdsResponse.Multiad) == 0 {
		return nil, errors.New("convertHuaweiAdsResp2BidderResp: multiad length is 0, get no ads from huawei side.")
	}
	bidderResponse = adapters.NewBidderResponseWithBidsCapacity(len(huaweiAdsResponse.Multiad))
	// Default Currency: CNY
	bidderResponse.Currency = "CNY"

	// record request Imp (slotid->imp, slotid->openrtb_ext.bidtype)
	mapSlotid2Imp := make(map[string]openrtb2.Imp, len(openRTBRequest.Imp))
	mapSlotid2MediaType := make(map[string]openrtb_ext.BidType, len(openRTBRequest.Imp))
	for _, imp := range openRTBRequest.Imp {
		huaweiAdsExt, err := unmarshalExtImpHuaweiAds(&imp)
		if err != nil {
			continue
		}
		mapSlotid2Imp[huaweiAdsExt.SlotId] = imp

		var mediaType = openrtb_ext.BidTypeBanner
		if imp.Video != nil {
			mediaType = openrtb_ext.BidTypeVideo
		} else if imp.Native != nil {
			mediaType = openrtb_ext.BidTypeNative
		} else if imp.Audio != nil {
			mediaType = openrtb_ext.BidTypeAudio
		}
		mapSlotid2MediaType[huaweiAdsExt.SlotId] = mediaType
	}

	if len(mapSlotid2MediaType) < 1 || len(mapSlotid2Imp) < 1 {
		return nil, errors.New("convertHuaweiAdsResp2BidderResp: openRTBRequest.imp is nil")
	}
	if huaweiAdsResponse.Multiad == nil {
		return nil, errors.New("convertHuaweiAdsResp2BidderResp: huaweiAdsResponse.Multiad is nil")
	}

	for _, ad30 := range huaweiAdsResponse.Multiad {
		if mapSlotid2Imp[ad30.Slotid].ID == "" {
			continue
		}

		if ad30.Retcode30 != 200 {
			continue
		}

		if ad30.Content == nil {
			continue
		}

		for _, content := range ad30.Content {
			var bid openrtb2.Bid
			bid.ID = mapSlotid2Imp[ad30.Slotid].ID
			bid.ImpID = mapSlotid2Imp[ad30.Slotid].ID
			// The bidder has already helped us automatically convert the currency price, here only the CNY price is filled in
			bid.Price = content.Price
			// Advertising creative ID, used for logging and behavior tracking
			bid.CrID = content.Contentid
			// All currencies should be the same
			if content.Cur != "" {
				bidderResponse.Currency = content.Cur
			}

			bid.AdM, bid.W, bid.H, err = getAdmFromHuaweiAdsContent(ad30.AdType, &content, mapSlotid2MediaType[ad30.Slotid], mapSlotid2Imp[ad30.Slotid])
			if err != nil {
				return nil, err
			}

			bid.ADomain = append(bid.ADomain, "huaweiads")
			// TODO
			bid.NURL = ""
			bid.LURL = ""
			bid.IURL = ""
			bid.BURL = ""

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: mapSlotid2MediaType[ad30.Slotid],
			})
		}
	}
	return bidderResponse, nil
}

// get field Adm
func getAdmFromHuaweiAdsContent(adType int32, content *Content, bidType openrtb_ext.BidType, imp openrtb2.Imp) (adm string, adWidth int64, adHeight int64, err error) {
	var creativeType = content.Creativetype
	if content.Creativetype > 100 {
		creativeType = creativeType - 100
	}

	if bidType == openrtb_ext.BidTypeVideo {
		if creativeType == videoText || creativeType == video || creativeType == videoWithPicturesText {
			adm, adWidth, adHeight, err = extractAdmVideo(adType, content, bidType, imp)
		} else {
			return "", 0, 0, fmt.Errorf("huaweiads response dose not have video")
		}
	} else if bidType == openrtb_ext.BidTypeNative {
		if adType == native {
			adm, adWidth, adHeight, err = extractAdmNative(adType, content, bidType, imp)
		} else {
			return "", 0, 0, fmt.Errorf("huaweiads response is not a native ad")
		}
	} else if bidType == openrtb_ext.BidTypeBanner {
		if creativeType == text || creativeType == bigPicture || creativeType == bigPicture2 ||
			creativeType == smallPicture || creativeType == threeSmallPicturesText ||
			creativeType == iconText || creativeType == gif {
			adm, adWidth, adHeight, err = extractAdmPicture(content)
		} else if creativeType == videoText || creativeType == video || creativeType == videoWithPicturesText {
			adm, adWidth, adHeight, err = extractAdmVideo(adType, content, bidType, imp)
		} else {
			return "", 0, 0, fmt.Errorf("no support creativetype: " + strconv.Itoa(int(creativeType)))
		}
	} else if bidType == openrtb_ext.BidTypeAudio {
		return "", 0, 0, fmt.Errorf("no support bidtype: audio")
	}

	if err != nil {
		return "", 0, 0, fmt.Errorf("getAdmFromHuaweiAdsContent failed: %s", err)
	}
	return adm, adWidth, adHeight, nil
}

// For native ad
func extractAdmNative(adType int32, content *Content, bidType openrtb_ext.BidType, openrtb2Imp openrtb2.Imp) (adm string,
	adWidth int64, adHeight int64, err error) {
	if openrtb2Imp.Native == nil {
		return "", 0, 0, fmt.Errorf("extractAdmNative: imp.Native is nil")
	}

	if openrtb2Imp.Native.Request == "" {
		return "", 0, 0, fmt.Errorf("extractAdmNative: imp.Native.Request is empty")
	}

	var nativePayload nativeRequests.Request
	if err := json.Unmarshal(json.RawMessage(openrtb2Imp.Native.Request), &nativePayload); err != nil {
		return "", 0, 0, err
	}

	var nativeResult nativeResponse.Response
	var linkObject nativeResponse.Link
	linkObject.URL = content.MetaData.ClickUrl
	if content.MetaData.ClickUrl == "" {
		return "", 0, 0, fmt.Errorf("extractAdmNative: content.MetaData.ClickUrl is empty")
	}

	nativeResult.Assets = make([]nativeResponse.Asset, 0, len(nativePayload.Assets))
	var ImgIndex = 0
	for _, asset := range nativePayload.Assets {
		var responseAsset nativeResponse.Asset
		if asset.Title != nil {
			var titleObject nativeResponse.Title
			titleObject.Text = ""
			if content.MetaData.Title != "" {
				if decodeTitle, err := url.QueryUnescape(content.MetaData.Title); err == nil {
					titleObject.Text = decodeTitle
				}
			}
			responseAsset.Title = &titleObject
		} else if asset.Video != nil {
			var vastXml string
			var err error
			if vastXml, adWidth, adHeight, err = extractAdmVideo(adType, content, bidType, openrtb2Imp); err != nil {
				return "", 0, 0, err
			}
			var videoObject nativeResponse.Video
			videoObject.VASTTag = vastXml
			responseAsset.Video = &videoObject
		} else if asset.Img != nil {
			var imgObject nativeResponse.Image
			if len(content.MetaData.ImageInfo) > ImgIndex {
				imgObject.URL = content.MetaData.ImageInfo[ImgIndex].Url
				imgObject.W = content.MetaData.ImageInfo[ImgIndex].Width
				imgObject.H = content.MetaData.ImageInfo[ImgIndex].Height
			} else {
				imgObject.URL = ""
				imgObject.W = 0
				imgObject.H = 0
			}
			ImgIndex++
			responseAsset.Img = &imgObject
		} else if asset.Data != nil {
			var dataObject nativeResponse.Data
			dataObject.Label = ""
			dataObject.Value = ""
			responseAsset.Data = &dataObject
		}
		var id = asset.ID
		responseAsset.ID = &id
		nativeResult.Assets = append(nativeResult.Assets, responseAsset)
	}

	if content.Monitor != nil {
		for _, monitor := range content.Monitor {
			if len(monitor.Url) == 0 {
				continue
			}
			if monitor.EventType == click {
				for _, trackerUrl := range monitor.Url {
					linkObject.ClickTrackers = append(linkObject.ClickTrackers, trackerUrl)
				}
				nativeResult.Link = linkObject
			}
			if monitor.EventType == imp {
				for _, trackerUrl := range monitor.Url {
					nativeResult.ImpTrackers = append(nativeResult.ImpTrackers, trackerUrl)
				}
			}
		}
	}
	nativeResult.Ver = "1.1"
	if nativePayload.Ver != "" {
		nativeResult.Ver = nativePayload.Ver
	}

	var result []byte
	if result, err = JSON(nativeResult); err != nil {
		return "", 0, 0, err
	}
	return strings.Replace(string(result), "\n", "", -1), adWidth, adHeight, nil
}

// JSON custom
func JSON(nativeResult nativeResponse.Response) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(nativeResult)
	return buffer.Bytes(), err
}

// For creative picture
func extractAdmPicture(content *Content) (adm string, adWidth int64, adHeight int64, err error) {
	if content == nil {
		return "", 0, 0, fmt.Errorf("extractAdmPicture: content is empty")
	}

	var clickUrl = ""
	if content.MetaData.ClickUrl != "" {
		clickUrl = content.MetaData.ClickUrl
	}

	// now handle one picture, maybe two three picture, wait to do.
	// TODO two three picture
	var imageInfoUrl string
	var height int64
	var width int64
	if content.MetaData.ImageInfo != nil {
		imageInfoUrl = content.MetaData.ImageInfo[0].Url
		height = content.MetaData.ImageInfo[0].Height
		width = content.MetaData.ImageInfo[0].Width
	} else {
		return "", 0, 0, fmt.Errorf("coontent.MetaData.ImageInfo is empty")
	}
	adWidth = width
	adHeight = height

	var impMonitorUrls string = ""
	var clickMonitorUrls string = ""
	var userCloseMonitorUrls string = ""
	if content.Monitor != nil {
		for _, monitor := range content.Monitor {
			if len(monitor.Url) == 0 {
				continue
			}
			if monitor.EventType == imp {
				impMonitorUrls = getStrings(monitor.Url)
			}
			if monitor.EventType == click {
				clickMonitorUrls = getStrings(monitor.Url)
			}
			if monitor.EventType == userClose {
				userCloseMonitorUrls = getStrings(monitor.Url)
			}
		}
	}

	// handle ads event
	var script = "<script type=\"text/javascript\">" +
		"(function () {var impMonitorUrls = [" + impMonitorUrls + "];" +
		"var clickMonitorUrls = [" + clickMonitorUrls + "];" +
		"var userCloseMonitorUrls = [" + userCloseMonitorUrls + "];" +
		"function visitUrl(url) {var img = new Image();img.src = url;return img;}" +
		"function visitAllUrls(urls) {" +
		"for (var i = 0; i < urls.length; i++) {visitUrl(urls[i]);}}" +
		"function addEventListener(node, event, func, useCapture) {" +
		"node = node || document;useCapture = useCapture || false;" +
		"if (node.addEventListener) {node.addEventListener(event, func, useCapture);} else {node.attachEvent('on' + event, func);}}" +
		"function init() {var imgLink = document.getElementById('img_link');" +
		"if (imgLink) {addEventListener(imgLink, 'click', function () {visitAllUrls(clickMonitorUrls);}, false);}" +
		"var close = document.getElementById('close');" +
		"if (close) {addEventListener(close, 'click', function () {visitAllUrls(userCloseMonitorUrls)}, false);}}" +
		"window.onload = function () {visitAllUrls(impMonitorUrls);init();}})();" +
		"</script>"

	adm = "<head><meta charset=\"UTF-8\">" +
		"<meta http-equiv=\"Content-Type\" content=\"text/html\">" +
		"<meta http-equiv=\"X-UA-Compatible\" content=\"IE=edge,chrome=1\">" + script + "</head>" +
		"<body><div class=\"product\">" +
		"<a class=\"img_area\" id=\"img_link\" href=\"" + clickUrl + "\" target=\"_blank\">" +
		"<img src=\"" + imageInfoUrl + "\" width=\"" + strconv.Itoa(int(width)) + "\" height=\"" + strconv.Itoa(int(height)) + "\" alt=\"\"/>" +
		"</a></div>" +
		"</body>"

	return adm, adWidth, adHeight, nil
}

// Add url for monitor
func getStrings(eles []string) string {
	if len(eles) == 0 {
		return ""
	}

	var strs strings.Builder
	for i := 0; i < len(eles); i++ {
		strs.WriteString("\"" + eles[i] + "\"")
		if i < len(eles)-1 {
			strs.WriteString(",")
		}
	}
	return strs.String()
}

// get duration, format: 00:00:00.000
func getDuration(duration int64) string {
	// duration millisecond
	if duration == 0 {
		return "00:00:00.000"
	}
	var totalSec = int(duration) / 1000
	var mmm = int(duration) % 1000
	var hour = totalSec / 3600
	var tmp = totalSec % 3600
	var min = tmp / 60
	var sec = tmp % 60
	var result string
	result += addZero(strconv.Itoa(hour), ":", false)
	result += addZero(strconv.Itoa(min), ":", false)
	result += addZero(strconv.Itoa(sec), ".", false)
	result += addZero(strconv.Itoa(mmm), "", true)
	return result
}

func addZero(str string, dot string, isMmm bool) (result string) {
	if isMmm == false {
		if len(str) < 2 {
			result = "0" + str + dot
		} else {
			result = str + dot
		}
	} else {
		if len(str) == 1 {
			result = "00" + str
		} else if len(str) == 2 {
			result = "0" + str
		} else {
			result = str
		}
	}
	return result
}

// get field adm for video
func extractAdmVideo(adType int32, content *Content, bidType openrtb_ext.BidType, opentrb2Imp openrtb2.Imp) (adm string,
	adWidth int64, adHeight int64, err error) {
	if content == nil {
		return "", 0, 0, fmt.Errorf("extractAdmVideo: content is empty")
	}

	var clickUrl = ""
	if content.MetaData.ClickUrl != "" {
		clickUrl = content.MetaData.ClickUrl
	} else {
		return "", 0, 0, fmt.Errorf("extractAdmVideo: Content.MetaData.Clickurl is empty")
	}

	var mime = "video/mp4"
	var resourceUrl = ""
	if adType == roll {
		if content.MetaData.MediaFile.Mime != "" {
			mime = content.MetaData.MediaFile.Mime
		}
		adWidth = content.MetaData.MediaFile.Width
		adHeight = content.MetaData.MediaFile.Height
		if content.MetaData.MediaFile.Url != "" {
			resourceUrl = content.MetaData.MediaFile.Url
		} else {
			return "", 0, 0, fmt.Errorf("extractAdmVideo: Content.MetaData.MediaFile.Url is empty")
		}
	} else {
		if content.MetaData.VideoInfo.VideoDownloadUrl != "" {
			resourceUrl = content.MetaData.VideoInfo.VideoDownloadUrl
		} else {
			return "", 0, 0, fmt.Errorf("extractAdmVideo: content.MetaData.VideoInfo.VideoDownloadUrl is empty")
		}
		adWidth, adHeight, err = getWidthAndHeightForVideo(adType, content, bidType, opentrb2Imp)
		if err != nil {
			return "", 0, 0, err
		}
	}

	var adTitle = ""
	if content.MetaData.Title != "" {
		decodeTitle, err := url.QueryUnescape(content.MetaData.Title)
		if err != nil {
			adTitle = decodeTitle
		}
	}

	var adId = content.Contentid
	var impressionTrackingEvent = ""
	var duration = getDuration(content.MetaData.Duration)
	var trackingEvents strings.Builder
	var clickTrackingEvent = ""
	if content.Monitor != nil {
		for _, monitor := range content.Monitor {
			if len(monitor.Url) == 0 {
				continue
			}
			var event = ""
			if monitor.EventType == imp {
				impressionTrackingEvent = "<Impression><![CDATA[" + strings.Join(monitor.Url, ";") + "]]></Impression>"
			} else if monitor.EventType == click {
				clickTrackingEvent = "<ClickTracking><![CDATA[" + strings.Join(monitor.Url, ";") + "]]></ClickTracking>"
			} else if monitor.EventType == userClose {
				event = "skip"
			} else if monitor.EventType == playStart {
				event = "start"
			} else if monitor.EventType == playEnd {
				event = "complete"
			} else if monitor.EventType == playResume {
				event = "resume"
			} else if monitor.EventType == playPause {
				event = "pause"
			}
			if event != "" {
				trackingEvents.WriteString("<Tracking event=\"" + event + "\"><![CDATA[" + strings.Join(monitor.Url, ";") + "]]></Tracking>")
			}
		}
	}

	// vast 3.0 -> adm
	adm = "<?xml version=\"1.0\" encoding=\"UTF-8\"?><VAST version=\"3.0\">" +
		"<Ad id=\"" + adId + "\"><InLine><AdSystem>HuaweiAds</AdSystem>" +
		"<AdTitle><![CDATA[" + adTitle + "]]></AdTitle>" + impressionTrackingEvent +
		"<Creatives><Creative><Linear><Duration>" + duration + "</Duration>" +
		"<TrackingEvents>" + trackingEvents.String() + "</TrackingEvents>" +
		"<VideoClicks><ClickThrough><![CDATA[" + clickUrl + "]]></ClickThrough>" + clickTrackingEvent + "</VideoClicks>" +
		"<MediaFiles><MediaFile bitrate=\"\" delivery=\"progressive\" height=\"" + strconv.Itoa(int(adHeight)) + "\" type=\"" + mime + "\" width=\"" + strconv.Itoa(int(adWidth)) + "\">" +
		"<![CDATA[" + resourceUrl + "]]>" +
		"</MediaFile></MediaFiles>" +
		"</Linear></Creative></Creatives>" +
		"</InLine></Ad></VAST>"
	return adm, adWidth, adHeight, nil
}

func getWidthAndHeightForVideo(adType int32, content *Content, bidType openrtb_ext.BidType, openrtb2Imp openrtb2.Imp) (
	adWidth int64, adHeight int64, err error) {
	if openrtb2Imp.Video == nil {
		if bidType == openrtb_ext.BidTypeVideo {
			return 0, 0, fmt.Errorf("getWidthAndHeightForVideo: openrtb2Imp.Video is nil")
		}
	} else {
		if openrtb2Imp.Video.W != 0 && openrtb2Imp.Video.H != 0 {
			return openrtb2Imp.Video.W, openrtb2Imp.Video.H, nil
		}
	}

	if content.MetaData.ImageInfo == nil && content.MetaData.Icon == nil {
		return 0, 0, fmt.Errorf("can not get width, height for video")
	}

	if content.MetaData.ImageInfo != nil && len(content.MetaData.ImageInfo) > 0 {
		adWidth = content.MetaData.ImageInfo[0].Width
		adHeight = content.MetaData.ImageInfo[0].Height
	} else if content.MetaData.Icon != nil && len(content.MetaData.Icon) > 0 {
		adWidth = content.MetaData.Icon[0].Width
		adHeight = content.MetaData.Icon[0].Height
	}

	if adWidth == 0 || adHeight == 0 {
		return 0, 0, fmt.Errorf("can not get width, height for video")
	}
	return adWidth, adHeight, nil
}

// compute HmacSha256
func computeHmacSha256(message string, signKey string) string {
	h := hmac.New(sha256.New, []byte(signKey))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// get digest authorization for request header, huawei ads ppsadx API
func getDigestAuthorization(huaweiAdsImpExt *openrtb_ext.ExtImpHuaweiAds) string {
	var nonce = strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
	var apiKey = huaweiAdsImpExt.PublisherId + ":ppsadx/getResult:" + huaweiAdsImpExt.SignKey
	return "Digest username=" + huaweiAdsImpExt.PublisherId + "," +
		"realm=ppsadx/getResult," +
		"nonce=" + nonce + "," +
		"response=" + computeHmacSha256(nonce+":POST:/ppsadx/getResult", apiKey) + "," +
		"algorithm=HmacSHA256,usertype=1,keyid=" + huaweiAdsImpExt.KeyId
}
