package huaweiads

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mxmCherry/openrtb/v15/native1"
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
const DefaultTimeZone = "+0200"
const DefaultModelName = "HUAWEI"
const DefaultTrackingEndpoint = "https://events-dra.op.hicloud.com/contserver/tracker/action"

// creative type
const (
	text                   int32 = 1
	bigPicture             int32 = 2
	bigPicture2            int32 = 3
	gif                    int32 = 4
	videoText              int32 = 6
	smallPicture           int32 = 7
	threeSmallPicturesText int32 = 8
	video                  int32 = 9
	iconText               int32 = 10
	videoWithPicturesText  int32 = 11
)

// ads type
const (
	banner int32 = 8
	native int32 = 3
	roll   int32 = 60
)

type HuaweiAdsRequest struct {
	Version           string     `json:"version"`
	Multislot         []Adslot30 `json:"multislot"`
	App               App        `json:"app"`
	Device            Device     `json:"device"`
	Network           Network    `json:"network,omitempty"`
	ClientAdRequestId string     `json:"clientAdRequestId,omitempty"`
	ParentCtrlUser    int32      `json:"parentCtrlUser,omitempty"`
	NonPersonalizedAd int32      `json:"nonPersonalizedAd,omitempty"`
	Regs              Regs       `json:"regs,omitempty"`
	Geo               Geo        `json:"geo,omitempty"`
	Consent           string     `json:"consent,omitempty"`
}

type Adslot30 struct {
	Slotid                   string   `json:"slotid"`
	Adtype                   int32    `json:"adtype"`
	Test                     int32    `json:"test"`
	TotalDuration            int32    `json:"totalDuration,omitempty"`
	Orientation              int32    `json:"orientation,omitempty"`
	W                        int64    `json:"w,omitempty"`
	H                        int64    `json:"h,omitempty"`
	Format                   []Format `json:"format,omitempty"`
	DetailedCreativeTypeList []string `json:"detailedCreativeTypeList,omitempty"`
}

type Format struct {
	W int64 `json:"w,omitempty"`
	H int64 `json:"h,omitempty"`
}

type App struct {
	Version string `json:"version,omitempty"`
	Name    string `json:"name,omitempty"`
	Pkgname string `json:"pkgname"`
	Lang    string `json:"lang,omitempty"`
	Country string `json:"country,omitempty"`
}

type Device struct {
	Type                int32   `json:"type,omitempty"`
	Useragent           string  `json:"useragent,omitempty"`
	Os                  string  `json:"os,omitempty"`
	Version             string  `json:"version,omitempty"`
	Maker               string  `json:"maker,omitempty"`
	Model               string  `json:"model,omitempty"`
	Width               int32   `json:"width,omitempty"`
	Height              int32   `json:"height,omitempty"`
	Language            string  `json:"language,omitempty"`
	BuildVersion        string  `json:"buildVersion,omitempty"`
	Dpi                 int32   `json:"dpi,omitempty"`
	Pxratio             float32 `json:"pxratio,omitempty"`
	Imei                string  `json:"imei,omitempty"`
	Oaid                string  `json:"oaid,omitempty"`
	IsTrackingEnabled   string  `json:"isTrackingEnabled,omitempty"`
	EmuiVer             string  `json:"emuiVer,omitempty"`
	LocaleCountry       string  `json:"localeCountry"`
	SimCountryIso       string  `json:"simCountryIso,omitempty"`
	BelongCountry       string  `json:"belongCountry"`
	GaidTrackingEnabled string  `json:"gaidTrackingEnabled,omitempty"`
	Gaid                string  `json:"gaid,omitempty"`
	VerCodeOfHms        string  `json:"verCodeOfHms,omitempty"`
	ClientTime          string  `json:"clientTime"`
	VerCodeOfAG         string  `json:"verCodeOfAG,omitempty"`
	VendorCountry       string  `json:"vendorCountry,omitempty"`
	RoLocaleCountry     string  `json:"roLocaleCountry,omitempty"`
	AgCountryCode       string  `json:"agCountryCode,omitempty"`
	RouterCountry       string  `json:"routerCountry,omitempty"`
	RoLocale            string  `json:"roLocale,omitempty"`
	Ip                  string  `json:"ip,omitempty"`
}

type Network struct {
	Type     int32      `json:"type"`
	Carrier  int32      `json:"carrier,omitempty"`
	CellInfo []CellInfo `json:"cellInfo,omitempty"`
}

type Regs struct {
	Coppa    int32  `json:"coppa,omitempty"`
	Tfua     int32  `json:"tfua,omitempty"`
	AdRating string `json:"adRating,omitempty"`
}

type Geo struct {
	Lon      float32 `json:"lon,omitempty"`
	Lat      float32 `json:"lat,omitempty"`
	Accuracy int32   `json:"accuracy,omitempty"`
	Lastfix  int32   `json:"lastfix,omitempty"`
}

type CellInfo struct {
	Mcc string `json:"mcc,omitempty"`
	Mnc string `json:"mnc,omitempty"`
}

type HuaweiAdsResponse struct {
	Retcode int32  `json:"retcode"`
	Reason  string `json:"reason"`
	Multiad []Ad30 `json:"multiad"`
}

type Ad30 struct {
	AdType    int32     `json:"adtype"`
	Slotid    string    `json:"slotid"`
	Retcode30 int32     `json:"retcode30"`
	Content   []Content `json:"content"`
}

type Content struct {
	Contentid       string          `json:"contentid"`
	Endtime         int64           `json:"endtime"`
	Interactiontype int32           `json:"interactiontype"`
	Creativetype    int32           `json:"creativetype"`
	MetaData        MetaData        `json:"metaData"`
	Starttime       int64           `json:"starttime"`
	KeyWords        []string        `json:"keyWords"`
	KeyWordsType    []string        `json:"keyWordsType"`
	Monitor         []Monitor       `json:"monitor"`
	Paramfromserver Paramfromserver `json:"paramfromserver"`
	RewardItem      RewardItem      `json:"rewardItem"`
	Cur             string          `json:"cur"`
	Price           float64         `json:"price"`
}

type Paramfromserver struct {
	A   string `json:"a"`
	Sig string `json:"sig"`
	T   string `json:"t"`
}

type MetaData struct {
	Title             string      `json:"title"`
	Description       string      `json:"description"`
	ImageInfo         []ImageInfo `json:"imageInfo"`
	Icon              []Icon      `json:"icon"`
	ClickUrl          string      `json:"clickUrl"`
	Label             string      `json:"label"`
	Intent            string      `json:"intent"`
	VideoInfo         VideoInfo   `json:"videoInfo"`
	ApkInfo           ApkInfo     `json:"apkInfo"`
	Duration          int64       `json:"duration"`
	MediaFile         MediaFile   `json:"mediaFile"`
	RewardCriterion   string      `json:"rewardCriterion"`
	ScreenOrientation string      `json:"screenOrientation"`
	PrivacyUrl        string      `json:"privacyUrl"`
}

type ImageInfo struct {
	Url       string `json:"url"`
	Height    int64  `json:"height"`
	FileSize  int64  `json:"fileSize"`
	Sha256    string `json:"sha256"`
	ImageType string `json:"imageType"`
	Width     int64  `json:"width"`
}

type Icon struct {
	Url       string `json:"url"`
	Height    int64  `json:"height"`
	FileSize  int64  `json:"fileSize"`
	Sha256    string `json:"sha256"`
	ImageType string `json:"imageType"`
	Width     int64  `json:"width"`
}

type VideoInfo struct {
	VideoDownloadUrl string  `json:"videoDownloadUrl"`
	VideoDuration    int32   `json:"videoDuration"`
	VideoFileSize    int32   `json:"videoFileSize"`
	Sha256           string  `json:"sha256"`
	VideoRatio       float32 `json:"videoRatio"`
	Width            int32   `json:"width"`
	Height           int32   `json:"height"`
}

type ApkInfo struct {
	Url           string       `json:"url"`
	FileSize      int64        `json:"fileSize"`
	Sha256        string       `json:"sha256"`
	PackageName   string       `json:"packageName"`
	SecondUrl     string       `json:"secondUrl"`
	AppName       string       `json:"appName"`
	Permissions   []Permission `json:"permissions"`
	VersionName   string       `json:"versionName"`
	AppDesc       string       `json:"appDesc"`
	AppIcon       string       `json:"appIcon"`
	DeveloperName string       `json:"developerName"`
}

type MediaFile struct {
	Mime     string `json:"mime"`
	Width    int64  `json:"width"`
	Height   int64  `json:"height"`
	FileSize int64  `json:"fileSize"`
	Url      string `json:"url"`
	Sha256   string `json:"sha256"`
}

type Monitor struct {
	EventType string   `json:"eventType"`
	Url       []string `json:"url"`
}

type Permission struct {
	PermissionLabel string `json:"permissionLabel"`
	GroupDesc       string `json:"groupDesc"`
	TargetSDK       string `json:"targetSDK"`
}

type RewardItem struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

type adapter struct {
	endpoint  string
	extraInfo ExtraInfo
}

type ExtraInfo struct {
	TrackingUrl string `json:"trackingUrl,omitempty"`
}

func (a *adapter) MakeRequests(openRTBRequest *openrtb2.BidRequest,
	reqInfo *adapters.ExtraRequestInfo) (requestsToBidder []*adapters.RequestData, errs []error) {
	// the upstream code already confirms that there is a non-zero number of impressions
	numRequests := len(openRTBRequest.Imp)
	var request HuaweiAdsRequest
	var header http.Header
	var multislot = make([]Adslot30, 0, numRequests)

	var huaweiAdsImpExt *openrtb_ext.ExtImpHuaweiAds
	var err1 error
	for index := 0; index < numRequests && numRequests > 0; index++ {
		huaweiAdsImpExt, err1 = unmarshalExtImpHuaweiAds(&openRTBRequest.Imp[index])
		if err1 != nil || huaweiAdsImpExt == nil {
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
	//	use isAddAuthorization = false only when run testcase
	var isAddAuthorization = true
	if huaweiAdsImpExt != nil && huaweiAdsImpExt.IsAddAuthorization == "false" {
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

func (a *adapter) MakeBids(openRTBRequest *openrtb2.BidRequest, requestToBidder *adapters.RequestData,
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
	bidderResponse, err = a.convertHuaweiAdsResp2BidderResp(&huaweiAdsResponse, openRTBRequest)
	if err != nil {
		return nil, []error{err}
	}

	return bidderResponse, nil
}

// Builder builds a new instance of the HuaweiAds adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	extraInfo, err := getExtraInfo(config.ExtraAdapterInfo)
	if err != nil {
		return nil, err
	}

	bidder := &adapter{
		endpoint:  config.Endpoint,
		extraInfo: extraInfo,
	}
	return bidder, nil
}

func getExtraInfo(huaweiAdsExtraInfo string) (ExtraInfo, error) {
	if huaweiAdsExtraInfo == "" {
		return ExtraInfo{
			TrackingUrl: DefaultTrackingEndpoint,
		}, nil
	}

	var extraInfo ExtraInfo
	if err := json.Unmarshal([]byte(huaweiAdsExtraInfo), &extraInfo); err != nil {
		return extraInfo, fmt.Errorf("HuaweiAds ExtraInfo: invalid extra info: %v", err)
	}

	if extraInfo.TrackingUrl == "" {
		extraInfo.TrackingUrl = DefaultTrackingEndpoint
	}
	return extraInfo, nil
}

// get request header
func getHeaders(huaweiAdsImpExt *openrtb_ext.ExtImpHuaweiAds, request *openrtb2.BidRequest, isAddAuthorization bool) http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	if huaweiAdsImpExt == nil {
		return headers
	}

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

	if openRTBImp.Banner != nil {
		if openRTBImp.Banner.W != nil && openRTBImp.Banner.H != nil {
			adslot30.W = *openRTBImp.Banner.W
			adslot30.H = *openRTBImp.Banner.H
		}
		if len(openRTBImp.Banner.Format) != 0 {
			var formats = make([]Format, 0, len(openRTBImp.Banner.Format))
			for _, f := range openRTBImp.Banner.Format {
				if f.H != 0 && f.W != 0 {
					formats = append(formats, Format{f.W, f.H})
				}
			}
			adslot30.Format = formats
		}
	} else if openRTBImp.Native != nil {
		if err := getNativeFormat(&adslot30, openRTBImp); err != nil {
			return adslot30, err
		}
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

func getNativeFormat(adslot30 *Adslot30, openRTBImp *openrtb2.Imp) error {
	if openRTBImp.Native.Request == "" {
		return errors.New("extractAdmNative: imp.Native.Request is empty")
	}

	var nativePayload nativeRequests.Request
	if err := json.Unmarshal(json.RawMessage(openRTBImp.Native.Request), &nativePayload); err != nil {
		return err
	}

	var numImage = 0
	var numVideo = 0
	var width int64
	var height int64
	for _, asset := range nativePayload.Assets {
		if asset.Video != nil {
			numVideo++
		}
		// every image has the same W, H.
		if asset.Img != nil {
			numImage++
			if asset.Img.H != 0 && asset.Img.W != 0 {
				width = asset.Img.W
				height = asset.Img.H
			} else if asset.Img.WMin != 0 && asset.Img.HMin != 0 {
				width = asset.Img.WMin
				height = asset.Img.HMin
			}
		}
	}
	adslot30.W = width
	adslot30.H = height

	var detailedCreativeTypeList = make([]string, 0, 2)
	if numVideo >= 1 {
		detailedCreativeTypeList = append(detailedCreativeTypeList, "903")
	} else if numImage > 1 {
		detailedCreativeTypeList = append(detailedCreativeTypeList, "904")
	} else if numImage == 1 {
		detailedCreativeTypeList = append(detailedCreativeTypeList, "909")
	} else {
		detailedCreativeTypeList = append(detailedCreativeTypeList, "913", "914")
	}
	adslot30.DetailedCreativeTypeList = detailedCreativeTypeList
	return nil
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
	app.Country = getCountryCode(openRTBRequest)
	request.App = app
	return nil
}

// get field clientTime, format: 2006-01-02 15:04:05.000+0200
func getClientTime(clientTime string) (newClientTime string) {
	var zone = DefaultTimeZone
	t := time.Now().Local().Format(time.RFC822Z)
	index := strings.IndexAny(t, "-+")
	if index > 0 && len(t)-index == 5 {
		zone = t[index:]
	}
	if clientTime == "" {
		return time.Now().Format(TimeFormat) + zone
	}
	if isMatched, _ := regexp.MatchString("^\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}.\\d{3}[+-]{1}\\d{4}$", clientTime); isMatched {
		return clientTime
	}
	if isMatched, _ := regexp.MatchString("^\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}.\\d{3}$", clientTime); isMatched {
		return clientTime + zone
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
		var country = getCountryCode(openRTBRequest)
		device.BelongCountry = country
		device.LocaleCountry = country
		device.Ip = openRTBRequest.Device.IP
	}
	// get oaid gaid imei in openRTBRequest.User.Ext.Data
	if err = getDeviceID(&device, openRTBRequest); err != nil {
		return err
	}

	request.Device = device
	return nil
}

// get CountryCode ISO 3166-1 Alpha2
func getCountryCode(openRTBRequest *openrtb2.BidRequest) string {
	if openRTBRequest.Device != nil && openRTBRequest.Device.Geo != nil && openRTBRequest.Device.Geo.Country != "" {
		return convertCountryCode(openRTBRequest.Device.Geo.Country)
	} else if openRTBRequest.User != nil && openRTBRequest.User.Geo != nil && openRTBRequest.User.Geo.Country != "" {
		return convertCountryCode(openRTBRequest.User.Geo.Country)
	} else {
		return DefaultCountryName
	}
}

//ISO 3166-1 Alpha3 -> Alpha2, Some countries may use
func convertCountryCode(country string) (out string) {
	if country == "" {
		return DefaultCountryName
	}
	var MapCountryCodeAlpha3ToAlpha2 = map[string]string{
		"CHL": "CL", "CHN": "CN", "ARE": "AE"}
	if MapCountryCodeAlpha3ToAlpha2[country] != "" {
		return MapCountryCodeAlpha3ToAlpha2[country]
	} else if len(country) >= 3 {
		return country[0:2]
	} else {
		return DefaultCountryName
	}
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
	if len(deviceId.ClientTime) > 0 {
		device.ClientTime = getClientTime(deviceId.ClientTime[0])
	}
	// IsTrackingEnabled = 1 - DNT
	if openRTBRequest.Device != nil && openRTBRequest.Device.DNT != nil {
		if device.Oaid != "" {
			device.IsTrackingEnabled = strconv.Itoa(1 - int(*openRTBRequest.Device.DNT))
		}
		if device.Gaid != "" {
			device.GaidTrackingEnabled = strconv.Itoa(1 - int(*openRTBRequest.Device.DNT))
		}
	}
	return nil
}

// get network information for HuaweiAds request
func getHuaweiAdsReqNetWorkInfo(request *HuaweiAdsRequest, openRTBRequest *openrtb2.BidRequest) {
	var network Network
	if openRTBRequest.Device != nil {
		if openRTBRequest.Device.ConnectionType != nil {
			network.Type = int32(*openRTBRequest.Device.ConnectionType)
		} else {
			network.Type = DefaultUnknownNetworkType
		}

		var cellInfos []CellInfo
		if openRTBRequest.Device.MCCMNC != "" {
			var arr = strings.Split(openRTBRequest.Device.MCCMNC, "-")
			network.Carrier = 0
			if len(arr) >= 2 {
				cellInfos = append(cellInfos, CellInfo{
					Mcc: arr[0],
					Mnc: arr[1],
				})
				var str = arr[0] + arr[1]
				if str == "46000" || str == "46002" || str == "46007" {
					network.Carrier = 2
				} else if str == "46001" || str == "46006" {
					network.Carrier = 1
				} else if str == "46003" || str == "46005" || str == "46011" {
					network.Carrier = 3
				} else {
					network.Carrier = 99
				}
			}
		}
		network.CellInfo = cellInfos
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
func (a *adapter) convertHuaweiAdsResp2BidderResp(huaweiAdsResponse *HuaweiAdsResponse, openRTBRequest *openrtb2.BidRequest) (bidderResponse *adapters.BidderResponse, err error) {
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
			bid.CrID = content.Contentid
			// All currencies should be the same
			if content.Cur != "" {
				bidderResponse.Currency = content.Cur
			}

			bid.AdM, bid.W, bid.H, err = a.handleHuaweiAdsContent(ad30.AdType, &content, mapSlotid2MediaType[ad30.Slotid], mapSlotid2Imp[ad30.Slotid])
			if err != nil {
				return nil, err
			}
			bid.ADomain = append(bid.ADomain, "huaweiads")
			bid.NURL = getNurl(&content)
			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: mapSlotid2MediaType[ad30.Slotid],
			})
		}
	}
	return bidderResponse, nil
}

// get nurl when win auction.
func getNurl(content *Content) string {
	if len(content.Monitor) == 0 {
		return ""
	}
	for _, monitor := range content.Monitor {
		if monitor.EventType == "win" && len(monitor.Url) != 0 {
			return monitor.Url[0]
		}
	}
	return ""
}

// get field Adm, Width, Height
func (a *adapter) handleHuaweiAdsContent(adType int32, content *Content, bidType openrtb_ext.BidType, imp openrtb2.Imp) (
	adm string, adWidth int64, adHeight int64, err error) {
	// v1: only support banner, native
	switch bidType {
	case openrtb_ext.BidTypeBanner:
		adm, adWidth, adHeight, err = a.extractAdmBanner(adType, content, bidType, imp)
	case openrtb_ext.BidTypeNative:
		adm, adWidth, adHeight, err = a.extractAdmNative(adType, content, bidType, imp)
	case openrtb_ext.BidTypeVideo:
		adm, adWidth, adHeight, err = a.extractAdmVideo(adType, content, bidType, imp)
	default:
		return "", 0, 0, fmt.Errorf("no support bidtype: audio")
	}

	if err != nil {
		return "", 0, 0, fmt.Errorf("getAdmFromHuaweiAdsContent failed: %s", err)
	}
	return adm, adWidth, adHeight, nil
}

// banner ad
func (a *adapter) extractAdmBanner(adType int32, content *Content, bidType openrtb_ext.BidType, imp openrtb2.Imp) (adm string,
	adWidth int64, adHeight int64, err error) {
	if adType != banner {
		return "", 0, 0, fmt.Errorf("extractAdmBanner: huaweiads response is not a banner ad")
	}
	var creativeType = content.Creativetype
	if content.Creativetype > 100 {
		creativeType = creativeType - 100
	}
	if creativeType == text || creativeType == bigPicture || creativeType == bigPicture2 ||
		creativeType == smallPicture || creativeType == threeSmallPicturesText ||
		creativeType == iconText || creativeType == gif {
		return a.extractAdmPicture(content)
	} else if creativeType == videoText || creativeType == video || creativeType == videoWithPicturesText {
		return a.extractAdmVideo(adType, content, bidType, imp)
	} else {
		return "", 0, 0, fmt.Errorf("no banner support creativetype")
	}
}

// native ad
func (a *adapter) extractAdmNative(adType int32, content *Content, bidType openrtb_ext.BidType, openrtb2Imp openrtb2.Imp) (adm string,
	adWidth int64, adHeight int64, err error) {
	if adType != native {
		return "", 0, 0, fmt.Errorf("extractAdmNative: response is not a native ad")
	}
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
	if content.MetaData.ClickUrl != "" {
		linkObject.URL = content.MetaData.ClickUrl
	} else if content.MetaData.Intent != "" {
		linkObject.URL, _ = getDecodeValue(content.MetaData.Intent)
	}

	nativeResult.Assets = make([]nativeResponse.Asset, 0, len(nativePayload.Assets))
	var imgIndex = 0
	var iconIndex = 0
	for _, asset := range nativePayload.Assets {
		var responseAsset nativeResponse.Asset
		if asset.Title != nil {
			var titleObject nativeResponse.Title
			titleObject.Text, _ = getDecodeValue(content.MetaData.Title)
			titleObject.Len = int64(len(titleObject.Text))
			responseAsset.Title = &titleObject
		} else if asset.Video != nil {
			var videoObject nativeResponse.Video
			var err error
			if videoObject.VASTTag, adWidth, adHeight, err = a.extractAdmVideo(adType, content, bidType, openrtb2Imp); err != nil {
				return "", 0, 0, err
			}
			responseAsset.Video = &videoObject
		} else if asset.Img != nil {
			var imgObject nativeResponse.Image
			imgObject.URL = ""
			imgObject.Type = asset.Img.Type
			if asset.Img.Type == native1.ImageAssetTypeIcon {
				if len(content.MetaData.Icon) > iconIndex {
					imgObject.URL = content.MetaData.Icon[iconIndex].Url
					imgObject.W = content.MetaData.Icon[iconIndex].Width
					imgObject.H = content.MetaData.Icon[iconIndex].Height
					iconIndex++
				}
			} else {
				if len(content.MetaData.ImageInfo) > imgIndex {
					imgObject.URL = content.MetaData.ImageInfo[imgIndex].Url
					imgObject.W = content.MetaData.ImageInfo[imgIndex].Width
					imgObject.H = content.MetaData.ImageInfo[imgIndex].Height
					imgIndex++
				}
			}
			if adHeight == 0 && adWidth == 0 {
				adHeight = imgObject.H
				adWidth = imgObject.W
			}
			responseAsset.Img = &imgObject
		} else if asset.Data != nil {
			var dataObject nativeResponse.Data
			dataObject.Label = ""
			dataObject.Value = ""
			if asset.Data.Type == native1.DataAssetTypeDesc || asset.Data.Type == native1.DataAssetTypeDesc2 {
				dataObject.Label = "desc"
				dataObject.Value, _ = getDecodeValue(content.MetaData.Description)
			}
			responseAsset.Data = &dataObject
		}
		var id = asset.ID
		responseAsset.ID = &id
		nativeResult.Assets = append(nativeResult.Assets, responseAsset)
	}

	// dsp imp click tracking + imp click tracking
	if content.Monitor != nil {
		for _, monitor := range content.Monitor {
			if len(monitor.Url) == 0 {
				continue
			}
			if monitor.EventType == "click" {
				for _, trackerUrl := range monitor.Url {
					linkObject.ClickTrackers = append(linkObject.ClickTrackers, trackerUrl)
				}
			}
			if monitor.EventType == "imp" {
				for _, trackerUrl := range monitor.Url {
					nativeResult.ImpTrackers = append(nativeResult.ImpTrackers, trackerUrl)
				}
			}
		}
	}
	nativeResult.Link = linkObject
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

func getDecodeValue(str string) (decodeValue string, err error) {
	if str == "" {
		return "", nil
	}
	if decodeValue, err = url.QueryUnescape(str); err == nil {
		return decodeValue, nil
	} else {
		return "", err
	}
}

// JSON custom
func JSON(nativeResult nativeResponse.Response) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(nativeResult)
	return buffer.Bytes(), err
}

// For banner single picture
func (a *adapter) extractAdmPicture(content *Content) (adm string, adWidth int64, adHeight int64, err error) {
	if content == nil {
		return "", 0, 0, fmt.Errorf("extractAdmPicture: content is empty")
	}

	var clickUrl = ""
	if content.MetaData.ClickUrl != "" {
		clickUrl = content.MetaData.ClickUrl
	} else if content.MetaData.Intent != "" {
		clickUrl, _ = getDecodeValue(content.MetaData.Intent)
	}

	var imageInfoUrl string
	if content.MetaData.ImageInfo != nil {
		imageInfoUrl = content.MetaData.ImageInfo[0].Url
		adHeight = content.MetaData.ImageInfo[0].Height
		adWidth = content.MetaData.ImageInfo[0].Width
	} else {
		return "", 0, 0, fmt.Errorf("coontent.MetaData.ImageInfo is empty")
	}

	var imageTitle = ""
	imageTitle, _ = getDecodeValue(content.MetaData.Title)
	// dspImp, Imp, dspClick, Click tracking all can be found in MonitorUrl(imp ,click)
	dspImpTrackings, dspClickTrackings := getDspImpClickTrackings(content)
	var dspImpTrackings2StrImg strings.Builder
	for i := 0; i < len(dspImpTrackings); i++ {
		dspImpTrackings2StrImg.WriteString("<img height=\"1\" width=\"1\" src='" + dspImpTrackings[i] + "' >  ")
	}

	adm = "<style> html, body  " +
		"{ margin: 0; padding: 0; width: 100%; height: 100%; vertical-align: middle; }  " +
		"html  " +
		"{ display: table; }  " +
		"body { display: table-cell; vertical-align: middle; text-align: center; -webkit-text-size-adjust: none; }  " +
		"</style> " +
		"<span class=\"title-link advertiser_label\">" + imageTitle + "</span> " +
		"<a href='" + clickUrl + "' style=\"text-decoration:none\" " +
		"onclick=sendGetReq()> " +
		"<img src='" + imageInfoUrl + "' width='" + strconv.Itoa(int(adWidth)) + "' height='" + strconv.Itoa(int(adHeight)) + "'/> " +
		"</a> " +
		dspImpTrackings2StrImg.String() +
		"<script type=\"text/javascript\">" +
		"var dspClickTrackings = [" + dspClickTrackings + "];" +
		"function sendGetReq() {" +
		"sendSomeGetReq(dspClickTrackings)" +
		"}" +
		"function sendOneGetReq(url) {" +
		"var req = new XMLHttpRequest();" +
		"req.open('GET', url, true);" +
		"req.send(null);" +
		"}" +
		"function sendSomeGetReq(urls) {" +
		"for (var i = 0; i < urls.length; i++) {" +
		"sendOneGetReq(urls[i]);" +
		"}" +
		"}" +
		"</script>"
	return adm, adWidth, adHeight, nil
}

func getDspImpClickTrackings(content *Content) (dspImpTrackings []string, dspClickTrackings string) {
	if content.Monitor != nil {
		for _, monitor := range content.Monitor {
			if monitor.EventType == "imp" && len(monitor.Url) != 0 {
				dspImpTrackings = monitor.Url
			}
			if monitor.EventType == "click" && len(monitor.Url) != 0 {
				dspClickTrackings = getStrings(monitor.Url)
			}
		}
	}
	return dspImpTrackings, dspClickTrackings
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
func (a *adapter) extractAdmVideo(adType int32, content *Content, bidType openrtb_ext.BidType, opentrb2Imp openrtb2.Imp) (adm string,
	adWidth int64, adHeight int64, err error) {
	if content == nil {
		return "", 0, 0, fmt.Errorf("extractAdmVideo: content is empty")
	}

	var clickUrl = ""
	if content.MetaData.ClickUrl != "" {
		clickUrl = content.MetaData.ClickUrl
	} else if content.MetaData.Intent != "" {
		clickUrl, _ = getDecodeValue(content.MetaData.Intent)
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
			return "", 0, 0, errors.New("extractAdmVideo: Content.MetaData.MediaFile.Url is empty")
		}
	} else {
		if content.MetaData.VideoInfo.VideoDownloadUrl != "" {
			resourceUrl = content.MetaData.VideoInfo.VideoDownloadUrl
		} else {
			return "", 0, 0, errors.New("extractAdmVideo: content.MetaData.VideoInfo.VideoDownloadUrl is empty")
		}
		if content.MetaData.VideoInfo.Width != 0 && content.MetaData.VideoInfo.Height != 0 {
			adWidth = int64(content.MetaData.VideoInfo.Width)
			adHeight = int64(content.MetaData.VideoInfo.Height)
		} else if bidType == openrtb_ext.BidTypeVideo {
			if opentrb2Imp.Video != nil && opentrb2Imp.Video.W != 0 && opentrb2Imp.Video.H != 0 {
				adWidth = opentrb2Imp.Video.W
				adHeight = opentrb2Imp.Video.H
			}
		} else {
			return "", 0, 0, errors.New("extractAdmVideo: cannot get width, height")
		}
	}

	var adTitle, _ = getDecodeValue(content.MetaData.Title)
	var adId = content.Contentid
	var creativeId = content.Contentid
	var duration = getDuration(content.MetaData.Duration)
	var trackingEvents strings.Builder
	var dspImpTracking2Str = ""
	var dspClickTracking2Str = ""
	var errorTracking2Str = ""
	if content.Monitor != nil {
		for _, monitor := range content.Monitor {
			if len(monitor.Url) == 0 {
				continue
			}
			var event = ""
			switch monitor.EventType {
			case "vastError":
				errorTracking2Str = getVastImpClickErrorTrackingUrls(monitor.Url, "vastError")
			case "imp":
				dspImpTracking2Str = getVastImpClickErrorTrackingUrls(monitor.Url, "imp")
			case "click":
				dspClickTracking2Str = getVastImpClickErrorTrackingUrls(monitor.Url, "click")
			case "userclose":
				event = "skip&closeLinear"
			case "playStart":
				event = "start"
			case "playEnd":
				event = "complete"
			case "playResume":
				event = "resume"
			case "playPause":
				event = "pause"
			case "soundClickOff":
				event = "mute"
			case "soundClickOn":
				event = "unmute"
			default:
			}
			if event != "" {
				if event != "skip&closeLinear" {
					trackingEvents.WriteString(getVastEventTrackingUrls(monitor.Url, event))
				} else {
					trackingEvents.WriteString(getVastEventTrackingUrls(monitor.Url, "skip&closeLinear"))
				}
			}
		}
	}

	adm = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>" +
		"<VAST version=\"3.0\">" +
		"<Ad id=\"" + adId + "\"><InLine>" +
		"<AdSystem>HuaweiAds</AdSystem>" +
		"<AdTitle>" + adTitle + "</AdTitle>" +
		errorTracking2Str + dspImpTracking2Str +
		"<Creatives>" +
		"<Creative adId=\"" + adId + "\" id=\"" + creativeId + "\">" +
		"<Linear>" +
		"<Duration>" + duration + "</Duration>" +
		"<TrackingEvents>" + trackingEvents.String() + "</TrackingEvents>" +
		"<VideoClicks>" +
		"<ClickThrough><![CDATA[" + clickUrl + "]]></ClickThrough>" +
		dspClickTracking2Str +
		"</VideoClicks>" +
		"<MediaFiles>" +
		"<MediaFile delivery=\"progressive\" type=\"" + mime + "\" width=\"" + strconv.Itoa(int(adWidth)) + "\" " +
		"height=\"" + strconv.Itoa(int(adHeight)) + "\" scalable=\"true\" maintainAspectRatio=\"true\"> " +
		"<![CDATA[" + resourceUrl + "]]>" +
		"</MediaFile>" +
		"</MediaFiles>" +
		"</Linear>" +
		"</Creative>" +
		"</Creatives>" +
		"</InLine>" +
		"</Ad>" +
		"</VAST>"
	return adm, adWidth, adHeight, nil
}

func getVastImpClickErrorTrackingUrls(urls []string, eventType string) (result string) {
	var trackingUrls strings.Builder
	for _, url := range urls {
		if eventType == "click" {
			trackingUrls.WriteString("<ClickTracking><![CDATA[" + url + "]]></ClickTracking>")
		} else if eventType == "imp" {
			trackingUrls.WriteString("<Impression><![CDATA[" + url + "]]></Impression>")
		} else if eventType == "vastError" {
			trackingUrls.WriteString("<Error><![CDATA[" + url + "&et=[ERRORCODE]]]></Error>")
		}
	}
	return trackingUrls.String()
}

func getVastEventTrackingUrls(urls []string, eventType string) (result string) {
	var trackingUrls strings.Builder
	for _, eventUrl := range urls {
		if eventType == "skip&closeLinear" {
			trackingUrls.WriteString("<Tracking event=\"skip\"><![CDATA[" + eventUrl + "]]>" +
				"</Tracking><Tracking event=\"closeLinear\"><![CDATA[" + eventUrl + "]]></Tracking>")
		} else {
			trackingUrls.WriteString("<Tracking event=\"" + eventType + "\"><![CDATA[" + eventUrl + "]]></Tracking>")
		}
	}
	return trackingUrls.String()
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
