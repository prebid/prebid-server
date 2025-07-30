package huaweiads

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prebid/openrtb/v20/native1"
	nativeRequests "github.com/prebid/openrtb/v20/native1/request"
	nativeResponse "github.com/prebid/openrtb/v20/native1/response"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

const huaweiAdxApiVersion = "3.4"
const defaultCountryName = "ZA"
const defaultUnknownNetworkType = 0
const timeFormat = "2006-01-02 15:04:05.000"
const defaultTimeZone = "+0200"
const defaultModelName = "HUAWEI"
const chineseSiteEndPoint = "https://acd.op.hicloud.com/ppsadx/getResult"
const europeanSiteEndPoint = "https://adx-dre.op.hicloud.com/ppsadx/getResult"
const asianSiteEndPoint = "https://adx-dra.op.hicloud.com/ppsadx/getResult"
const russianSiteEndPoint = "https://adx-drru.op.hicloud.com/ppsadx/getResult"

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

// interaction type
const (
	appPromotion int32 = 3
)

// ads type
const (
	banner       int32 = 8
	native       int32 = 3
	roll         int32 = 60
	interstitial int32 = 12
	rewarded     int32 = 7
	splash       int32 = 1
	magazinelock int32 = 2
	audio        int32 = 17
)

type huaweiAdsRequest struct {
	Version           string     `json:"version"`
	Multislot         []adslot30 `json:"multislot"`
	App               app        `json:"app"`
	Device            device     `json:"device"`
	Network           network    `json:"network,omitempty"`
	Regs              regs       `json:"regs,omitempty"`
	Geo               geo        `json:"geo,omitempty"`
	Consent           string     `json:"consent,omitempty"`
	ClientAdRequestId string     `json:"clientAdRequestId,omitempty"`
}

type adslot30 struct {
	Slotid                   string   `json:"slotid"`
	Adtype                   int32    `json:"adtype"`
	Test                     int32    `json:"test"`
	TotalDuration            int32    `json:"totalDuration,omitempty"`
	Orientation              int32    `json:"orientation,omitempty"`
	W                        int64    `json:"w,omitempty"`
	H                        int64    `json:"h,omitempty"`
	Format                   []format `json:"format,omitempty"`
	DetailedCreativeTypeList []string `json:"detailedCreativeTypeList,omitempty"`
}

type format struct {
	W int64 `json:"w,omitempty"`
	H int64 `json:"h,omitempty"`
}

type app struct {
	Version string `json:"version,omitempty"`
	Name    string `json:"name,omitempty"`
	Pkgname string `json:"pkgname"`
	Lang    string `json:"lang,omitempty"`
	Country string `json:"country,omitempty"`
}

type device struct {
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
	BelongCountry       string  `json:"belongCountry"`
	GaidTrackingEnabled string  `json:"gaidTrackingEnabled,omitempty"`
	Gaid                string  `json:"gaid,omitempty"`
	ClientTime          string  `json:"clientTime"`
	Ip                  string  `json:"ip,omitempty"`
}

type network struct {
	Type     int32      `json:"type"`
	Carrier  int32      `json:"carrier,omitempty"`
	CellInfo []cellInfo `json:"cellInfo,omitempty"`
}

type regs struct {
	Coppa int32 `json:"coppa,omitempty"`
}

type geo struct {
	Lon      float32 `json:"lon,omitempty"`
	Lat      float32 `json:"lat,omitempty"`
	Accuracy int32   `json:"accuracy,omitempty"`
	Lastfix  int32   `json:"lastfix,omitempty"`
}

type cellInfo struct {
	Mcc string `json:"mcc,omitempty"`
	Mnc string `json:"mnc,omitempty"`
}

type huaweiAdsResponse struct {
	Retcode int32  `json:"retcode"`
	Reason  string `json:"reason"`
	Multiad []ad30 `json:"multiad"`
}

type ad30 struct {
	AdType    int32     `json:"adtype"`
	Slotid    string    `json:"slotid"`
	Retcode30 int32     `json:"retcode30"`
	Content   []content `json:"content"`
}

type content struct {
	Contentid       string    `json:"contentid"`
	Interactiontype int32     `json:"interactiontype"`
	Creativetype    int32     `json:"creativetype"`
	MetaData        metaData  `json:"metaData"`
	Monitor         []monitor `json:"monitor"`
	Cur             string    `json:"cur"`
	Price           float64   `json:"price"`
}

type metaData struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	ImageInfo   []imageInfo `json:"imageInfo"`
	Icon        []icon      `json:"icon"`
	ClickUrl    string      `json:"clickUrl"`
	Intent      string      `json:"intent"`
	VideoInfo   videoInfo   `json:"videoInfo"`
	ApkInfo     apkInfo     `json:"apkInfo"`
	Duration    int64       `json:"duration"`
	MediaFile   mediaFile   `json:"mediaFile"`
	Cta         string      `json:"cta"`
}

type imageInfo struct {
	Url       string `json:"url"`
	Height    int64  `json:"height"`
	FileSize  int64  `json:"fileSize"`
	Sha256    string `json:"sha256"`
	ImageType string `json:"imageType"`
	Width     int64  `json:"width"`
}

type icon struct {
	Url       string `json:"url"`
	Height    int64  `json:"height"`
	FileSize  int64  `json:"fileSize"`
	Sha256    string `json:"sha256"`
	ImageType string `json:"imageType"`
	Width     int64  `json:"width"`
}

type videoInfo struct {
	VideoDownloadUrl string  `json:"videoDownloadUrl"`
	VideoDuration    int32   `json:"videoDuration"`
	VideoFileSize    int32   `json:"videoFileSize"`
	Sha256           string  `json:"sha256"`
	VideoRatio       float32 `json:"videoRatio"`
	Width            int32   `json:"width"`
	Height           int32   `json:"height"`
}

type apkInfo struct {
	Url         string `json:"url"`
	FileSize    int64  `json:"fileSize"`
	Sha256      string `json:"sha256"`
	PackageName string `json:"packageName"`
	SecondUrl   string `json:"secondUrl"`
	AppName     string `json:"appName"`
	VersionName string `json:"versionName"`
	AppDesc     string `json:"appDesc"`
	AppIcon     string `json:"appIcon"`
}

type mediaFile struct {
	Mime     string `json:"mime"`
	Width    int64  `json:"width"`
	Height   int64  `json:"height"`
	FileSize int64  `json:"fileSize"`
	Url      string `json:"url"`
	Sha256   string `json:"sha256"`
}

type monitor struct {
	EventType string   `json:"eventType"`
	Url       []string `json:"url"`
}

type adapter struct {
	endpoint  string
	extraInfo ExtraInfo
}

type ExtraInfo struct {
	PkgNameConvert              []pkgNameConvert `json:"pkgNameConvert,omitempty"`
	CloseSiteSelectionByCountry string           `json:"closeSiteSelectionByCountry,omitempty"`
}

type pkgNameConvert struct {
	ConvertedPkgName           string   `json:"convertedPkgName,omitempty"`
	UnconvertedPkgNames        []string `json:"unconvertedPkgNames,omitempty"`
	UnconvertedPkgNameKeyWords []string `json:"unconvertedPkgNameKeyWords,omitempty"`
	UnconvertedPkgNamePrefixs  []string `json:"unconvertedPkgNamePrefixs,omitempty"`
	ExceptionPkgNames          []string `json:"exceptionPkgNames,omitempty"`
}

type empty struct {
}

func (a *adapter) MakeRequests(openRTBRequest *openrtb2.BidRequest,
	reqInfo *adapters.ExtraRequestInfo) (requestsToBidder []*adapters.RequestData, errs []error) {
	// the upstream code already confirms that there is a non-zero number of impressions
	numRequests := len(openRTBRequest.Imp)
	var request huaweiAdsRequest
	var header http.Header
	var multislot = make([]adslot30, 0, numRequests)

	var huaweiAdsImpExt *openrtb_ext.ExtImpHuaweiAds
	for index := 0; index < numRequests; index++ {
		var err1 error
		huaweiAdsImpExt, err1 = unmarshalExtImpHuaweiAds(&openRTBRequest.Imp[index])
		if err1 != nil {
			return nil, []error{err1}
		}

		if huaweiAdsImpExt == nil {
			return nil, []error{errors.New("Unmarshal ExtImpHuaweiAds failed: huaweiAdsImpExt is nil.")}
		}

		adslot30, err := getReqAdslot30(huaweiAdsImpExt, &openRTBRequest.Imp[index])
		if err != nil {
			return nil, []error{err}
		}

		multislot = append(multislot, adslot30)
	}
	request.Multislot = multislot
	request.ClientAdRequestId = openRTBRequest.ID

	countryCode, err := getReqJson(&request, openRTBRequest, a.extraInfo)
	if err != nil {
		return nil, []error{err}
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	//	our request header's Authorization is changing by time, cannot verify by a certain string,
	//	use isTestAuthorization = true only when run testcase
	var isTestAuthorization = false
	if huaweiAdsImpExt != nil && huaweiAdsImpExt.IsTestAuthorization == "true" {
		isTestAuthorization = true
	}
	header = getHeaders(huaweiAdsImpExt, openRTBRequest, isTestAuthorization)
	bidRequest := &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     getFinalEndPoint(countryCode, a.endpoint, a.extraInfo),
		Body:    reqJSON,
		Headers: header,
		ImpIDs:  openrtb_ext.GetImpIDs(openRTBRequest.Imp),
	}

	return []*adapters.RequestData{bidRequest}, nil
}

// countryCode is alpha2, choose the corresponding site end point
func getFinalEndPoint(countryCode string, defaultEndpoint string, extraInfo ExtraInfo) string {
	// closeSiteSelectionByCountry == 1, close site selection, use the defaultEndpoint
	if "1" == extraInfo.CloseSiteSelectionByCountry {
		return defaultEndpoint
	}

	if countryCode == "" || len(countryCode) > 2 {
		return defaultEndpoint
	}
	var europeanSiteCountryCodeGroup = map[string]empty{"AX": {}, "AL": {}, "AD": {}, "AU": {}, "AT": {}, "BE": {},
		"BA": {}, "BG": {}, "CA": {}, "HR": {}, "CY": {}, "CZ": {}, "DK": {}, "EE": {}, "FO": {}, "FI": {},
		"FR": {}, "DE": {}, "GI": {}, "GR": {}, "GL": {}, "GG": {}, "VA": {}, "HU": {}, "IS": {}, "IE": {},
		"IM": {}, "IL": {}, "IT": {}, "JE": {}, "YK": {}, "LV": {}, "LI": {}, "LT": {}, "LU": {}, "MT": {},
		"MD": {}, "MC": {}, "ME": {}, "NL": {}, "AN": {}, "NZ": {}, "NO": {}, "PL": {}, "PT": {}, "RO": {},
		"MF": {}, "VC": {}, "SM": {}, "RS": {}, "SX": {}, "SK": {}, "SI": {}, "ES": {}, "SE": {}, "CH": {},
		"TR": {}, "UA": {}, "GB": {}, "US": {}, "MK": {}, "SJ": {}, "BQ": {}, "PM": {}, "CW": {}}
	var russianSiteCountryCodeGroup = map[string]empty{"RU": {}}
	var chineseSiteCountryCodeGroup = map[string]empty{"CN": {}}
	// choose site
	if _, exists := chineseSiteCountryCodeGroup[countryCode]; exists {
		return chineseSiteEndPoint
	} else if _, exists := russianSiteCountryCodeGroup[countryCode]; exists {
		return russianSiteEndPoint
	} else if _, exists := europeanSiteCountryCodeGroup[countryCode]; exists {
		return europeanSiteEndPoint
	} else {
		return asianSiteEndPoint
	}
}

func (a *adapter) MakeBids(openRTBRequest *openrtb2.BidRequest, requestToBidder *adapters.RequestData,
	bidderRawResponse *adapters.ResponseData) (bidderResponse *adapters.BidderResponse, errs []error) {
	httpStatusError := checkRespStatusCode(bidderRawResponse)
	if httpStatusError != nil {
		return nil, []error{httpStatusError}
	}

	var huaweiAdsResponse huaweiAdsResponse
	if err := jsonutil.Unmarshal(bidderRawResponse.Body, &huaweiAdsResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Unable to parse server response",
		}}
	}

	if err := checkHuaweiAdsResponseRetcode(huaweiAdsResponse); err != nil {
		return nil, []error{err}
	}

	bidderResponse, err := a.convertHuaweiAdsRespToBidderResp(&huaweiAdsResponse, openRTBRequest)
	if err != nil {
		return nil, []error{err}
	}

	return bidderResponse, nil
}

// Builder builds a new instance of the HuaweiAds adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
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

func getExtraInfo(v string) (ExtraInfo, error) {
	var extraInfo ExtraInfo
	if len(v) == 0 {
		return extraInfo, nil
	}

	if err := jsonutil.Unmarshal([]byte(v), &extraInfo); err != nil {
		return extraInfo, fmt.Errorf("invalid extra info: %v , pls check", err)
	}

	for _, convert := range extraInfo.PkgNameConvert {
		if convert.ConvertedPkgName == "" {
			return extraInfo, fmt.Errorf("invalid extra info: ConvertedPkgName is empty, pls check")
		}

		if convert.UnconvertedPkgNameKeyWords != nil {
			for _, keyword := range convert.UnconvertedPkgNameKeyWords {
				if keyword == "" {
					return extraInfo, fmt.Errorf("invalid extra info: UnconvertedPkgNameKeyWords has a empty keyword, pls check")
				}
			}
		}

		if convert.UnconvertedPkgNamePrefixs != nil {
			for _, prefix := range convert.UnconvertedPkgNamePrefixs {
				if prefix == "" {
					return extraInfo, fmt.Errorf("invalid extra info: UnconvertedPkgNamePrefixs has a empty value, pls check")
				}
			}
		}
	}
	return extraInfo, nil
}

// getHeaders: get request header, Authorization -> digest
func getHeaders(huaweiAdsImpExt *openrtb_ext.ExtImpHuaweiAds, request *openrtb2.BidRequest, isTestAuthorization bool) http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	if huaweiAdsImpExt == nil {
		return headers
	}
	headers.Add("Authorization", getDigestAuthorization(huaweiAdsImpExt, isTestAuthorization))

	if request.Device != nil && len(request.Device.UA) > 0 {
		headers.Add("User-Agent", request.Device.UA)
	}
	return headers
}

// getReqJson: get body json for HuaweiAds request
func getReqJson(request *huaweiAdsRequest, openRTBRequest *openrtb2.BidRequest, extraInfo ExtraInfo) (countryCode string, err error) {
	request.Version = huaweiAdxApiVersion
	if countryCode, err = getReqAppInfo(request, openRTBRequest, extraInfo); err != nil {
		return "", err
	}
	if err = getReqDeviceInfo(request, openRTBRequest); err != nil {
		return "", err
	}
	getReqNetWorkInfo(request, openRTBRequest)
	getReqRegsInfo(request, openRTBRequest)
	getReqGeoInfo(request, openRTBRequest)
	getReqConsentInfo(request, openRTBRequest)
	return countryCode, nil
}

func getReqAdslot30(huaweiAdsImpExt *openrtb_ext.ExtImpHuaweiAds,
	openRTBImp *openrtb2.Imp) (adslot30, error) {
	adtype := convertAdtypeStringToInteger(strings.ToLower(huaweiAdsImpExt.Adtype))
	testStatus := 0
	if huaweiAdsImpExt.IsTestAuthorization == "true" {
		testStatus = 1
	}
	var adslot30 = adslot30{
		Slotid: huaweiAdsImpExt.SlotId,
		Adtype: adtype,
		Test:   int32(testStatus),
	}
	if err := checkAndExtractOpenrtbFormat(&adslot30, adtype, huaweiAdsImpExt.Adtype, openRTBImp); err != nil {
		return adslot30, err
	}
	return adslot30, nil
}

// opentrb :  huawei adtype
// banner <-> banner, interstitial
// native <-> native
// video  <->  banner, roll, interstitial, rewarded
func checkAndExtractOpenrtbFormat(adslot30 *adslot30, adtype int32, yourAdtype string, openRTBImp *openrtb2.Imp) error {
	if openRTBImp.Banner != nil {
		if adtype != banner && adtype != interstitial {
			return errors.New("check openrtb format: request has banner, doesn't correspond to huawei adtype " + yourAdtype)
		}
		getBannerFormat(adslot30, openRTBImp)
	} else if openRTBImp.Native != nil {
		if adtype != native {
			return errors.New("check openrtb format: request has native, doesn't correspond to huawei adtype " + yourAdtype)
		}
		if err := getNativeFormat(adslot30, openRTBImp); err != nil {
			return err
		}
	} else if openRTBImp.Video != nil {
		if adtype != banner && adtype != interstitial && adtype != rewarded && adtype != roll {
			return errors.New("check openrtb format: request has video, doesn't correspond to huawei adtype " + yourAdtype)
		}
		if err := getVideoFormat(adslot30, adtype, openRTBImp); err != nil {
			return err
		}
	} else if openRTBImp.Audio != nil {
		return errors.New("check openrtb format: request has audio, not currently supported")
	} else {
		return errors.New("check openrtb format: please choose one of our supported type banner, native, or video")
	}
	return nil
}

func getBannerFormat(adslot30 *adslot30, openRTBImp *openrtb2.Imp) {
	if openRTBImp.Banner.W != nil && openRTBImp.Banner.H != nil {
		adslot30.W = *openRTBImp.Banner.W
		adslot30.H = *openRTBImp.Banner.H
	}
	if len(openRTBImp.Banner.Format) != 0 {
		var formats = make([]format, 0, len(openRTBImp.Banner.Format))
		for _, f := range openRTBImp.Banner.Format {
			if f.H != 0 && f.W != 0 {
				formats = append(formats, format{f.W, f.H})
			}
		}
		adslot30.Format = formats
	}
}

func getNativeFormat(adslot30 *adslot30, openRTBImp *openrtb2.Imp) error {
	if openRTBImp.Native.Request == "" {
		return errors.New("extract openrtb native failed: imp.Native.Request is empty")
	}

	var nativePayload nativeRequests.Request
	if err := jsonutil.Unmarshal(json.RawMessage(openRTBImp.Native.Request), &nativePayload); err != nil {
		return err
	}

	//popular size for native ads
	popularSizes := []format{{W: 225, H: 150}, {W: 1080, H: 607}, {W: 300, H: 250}, {W: 1080, H: 1620}, {W: 1280, H: 720}, {W: 640, H: 360}, {W: 1080, H: 1920}, {W: 720, H: 1280}}

	// only compute the main image number, type = native1.ImageAssetTypeMain
	var numMainImage = 0
	var numVideo = 0
	var formats = make([]format, 0)
	var numFormat = 0
	var detailedCreativeTypeList = make([]string, 0, 2)

	//number of the requested image size
	for _, asset := range nativePayload.Assets {
		if numFormat > 1 {
			break
		}
		if asset.Img != nil {
			if asset.Img.Type == native1.ImageAssetTypeMain {
				numFormat++
			}
		}
	}

	sizeMap := make(map[format]struct{})
	for _, size := range popularSizes {
		sizeMap[size] = struct{}{}
	}

	for _, asset := range nativePayload.Assets {
		// Only one of the {title,img,video,data} objects should be present in each object.
		if asset.Video != nil {
			numVideo++
			formats = popularSizes

			w := ptrutil.ValueOrDefault(asset.Video.W)
			h := ptrutil.ValueOrDefault(asset.Video.H)

			_, ok := sizeMap[format{W: w, H: h}]
			if (w != 0 && h != 0) && !ok {
				formats = append(formats, format{w, h})
			}
		}
		// every image has the same W, H.
		if asset.Img != nil {
			if asset.Img.Type == native1.ImageAssetTypeMain {
				numMainImage++
				if numFormat > 1 && asset.Img.H != 0 && asset.Img.W != 0 && asset.Img.WMin != 0 && asset.Img.HMin != 0 {
					formats = append(formats, format{asset.Img.W, asset.Img.H})
				}
				if numFormat == 1 && asset.Img.H != 0 && asset.Img.W != 0 && asset.Img.WMin != 0 && asset.Img.HMin != 0 {
					result := filterPopularSizes(popularSizes, asset.Img.W, asset.Img.H, "ratio")
					formats = append(formats, result...)
				}
				if numFormat == 1 && asset.Img.H == 0 && asset.Img.W == 0 && asset.Img.WMin != 0 && asset.Img.HMin != 0 {
					result := filterPopularSizes(popularSizes, asset.Img.WMin, asset.Img.HMin, "range")
					formats = append(formats, result...)
				}
			}
		}
		adslot30.Format = formats
	}
	if numVideo >= 1 {
		detailedCreativeTypeList = append(detailedCreativeTypeList, "903")
	}
	if numMainImage >= 1 {
		detailedCreativeTypeList = append(detailedCreativeTypeList, "901", "904", "905")
	}
	adslot30.DetailedCreativeTypeList = detailedCreativeTypeList
	return nil
}

// filter popular size by range or ratio to append format array
func filterPopularSizes(sizes []format, width int64, height int64, byWhat string) []format {

	filtered := []format{}
	for _, size := range sizes {
		w := size.W
		h := size.H

		if byWhat == "ratio" {
			ratio := float64(width) / float64(height)
			diff := math.Abs(float64(w)/float64(h) - ratio)
			if diff <= 0.5 {
				filtered = append(filtered, size)
			}
		}
		if byWhat == "range" && w > width && h > height {
			filtered = append(filtered, size)
		}
	}
	return filtered
}

// roll ad need TotalDuration
func getVideoFormat(adslot30 *adslot30, adtype int32, openRTBImp *openrtb2.Imp) error {
	adslot30.W = ptrutil.ValueOrDefault(openRTBImp.Video.W)
	adslot30.H = ptrutil.ValueOrDefault(openRTBImp.Video.H)

	if adtype == roll {
		if openRTBImp.Video.MaxDuration == 0 {
			return errors.New("extract openrtb video failed: MaxDuration is empty when huaweiads adtype is roll.")
		}
		adslot30.TotalDuration = int32(openRTBImp.Video.MaxDuration)
	}
	return nil
}

func convertAdtypeStringToInteger(adtypeLower string) int32 {
	switch adtypeLower {
	case "banner":
		return banner
	case "native":
		return native
	case "rewarded":
		return rewarded
	case "interstitial":
		return interstitial
	case "roll":
		return roll
	case "splash":
		return splash
	case "magazinelock":
		return magazinelock
	case "audio":
		return audio
	default:
		return banner
	}
}

// getReqAppInfo: get app information for HuaweiAds request
func getReqAppInfo(request *huaweiAdsRequest, openRTBRequest *openrtb2.BidRequest, extraInfo ExtraInfo) (countryCode string, err error) {
	var app app
	if openRTBRequest.App != nil {
		if openRTBRequest.App.Ver != "" {
			app.Version = openRTBRequest.App.Ver
		}
		if openRTBRequest.App.Name != "" {
			app.Name = openRTBRequest.App.Name
		}

		// bundle cannot be empty, we need package name.
		if openRTBRequest.App.Bundle != "" {
			app.Pkgname = getFinalPkgName(openRTBRequest.App.Bundle, extraInfo)
		} else {
			return "", errors.New("generate HuaweiAds AppInfo failed: openrtb BidRequest.App.Bundle is empty.")
		}

		if openRTBRequest.App.Content != nil && openRTBRequest.App.Content.Language != "" {
			app.Lang = openRTBRequest.App.Content.Language
		} else {
			app.Lang = "en"
		}
	}
	countryCode = getCountryCode(openRTBRequest)
	app.Country = countryCode
	request.App = app
	return countryCode, nil
}

// when it has pkgNameConvert (include different rules)
// 1. when bundleName in ExceptionPkgNames, finalPkgname = bundleName
// 2. when bundleName conform UnconvertedPkgNames, finalPkgname = ConvertedPkgName
// 3. when bundleName conform keyword, finalPkgname = ConvertedPkgName
// 4. when bundleName conform prefix, finalPkgname = ConvertedPkgName
func getFinalPkgName(bundleName string, extraInfo ExtraInfo) string {
	for _, convert := range extraInfo.PkgNameConvert {
		if convert.ConvertedPkgName == "" {
			continue
		}

		for _, name := range convert.ExceptionPkgNames {
			if name == bundleName {
				return bundleName
			}
		}

		for _, name := range convert.UnconvertedPkgNames {
			if name == bundleName || name == "*" {
				return convert.ConvertedPkgName
			}
		}

		for _, keyword := range convert.UnconvertedPkgNameKeyWords {
			if strings.Index(bundleName, keyword) > 0 {
				return convert.ConvertedPkgName
			}
		}

		for _, prefix := range convert.UnconvertedPkgNamePrefixs {
			if strings.HasPrefix(bundleName, prefix) {
				return convert.ConvertedPkgName
			}
		}
	}
	return bundleName
}

// getClientTime: get field clientTime, format: 2006-01-02 15:04:05.000+0200
// If this parameter is not passed, the server time is used
func getClientTime(clientTime string) (newClientTime string) {
	var zone = defaultTimeZone
	t := time.Now().Local().Format(time.RFC822Z)
	index := strings.IndexAny(t, "-+")
	if index > 0 && len(t)-index == 5 {
		zone = t[index:]
	}
	if clientTime == "" {
		return time.Now().Format(timeFormat) + zone
	}
	if isMatched, _ := regexp.MatchString("^\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}\\.\\d{3}[+-]{1}\\d{4}$", clientTime); isMatched {
		return clientTime
	}
	if isMatched, _ := regexp.MatchString("^\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}\\.\\d{3}$", clientTime); isMatched {
		return clientTime + zone
	}
	return time.Now().Format(timeFormat) + zone
}

// getReqDeviceInfo: get device information for HuaweiAds request
func getReqDeviceInfo(request *huaweiAdsRequest, openRTBRequest *openrtb2.BidRequest) (err error) {
	var device device
	if openRTBRequest.Device != nil {
		device.Type = int32(openRTBRequest.Device.DeviceType)
		device.Useragent = openRTBRequest.Device.UA
		device.Os = openRTBRequest.Device.OS
		device.Version = openRTBRequest.Device.OSV
		device.Maker = openRTBRequest.Device.Make
		device.Model = openRTBRequest.Device.Model
		if device.Model == "" {
			device.Model = defaultModelName
		}
		device.Height = int32(openRTBRequest.Device.H)
		device.Width = int32(openRTBRequest.Device.W)
		device.Language = openRTBRequest.Device.Language
		device.Pxratio = float32(openRTBRequest.Device.PxRatio)
		var country = getCountryCode(openRTBRequest)
		device.BelongCountry = country
		device.LocaleCountry = country
		device.Ip = openRTBRequest.Device.IP
		device.Gaid = openRTBRequest.Device.IFA
		device.ClientTime = getClientTime("")
	}

	// get oaid gaid imei in openRTBRequest.User.Ext.Data
	if err = getDeviceIDFromUserExt(&device, openRTBRequest); err != nil {
		return err
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

	request.Device = device
	return nil
}

func getCountryCode(openRTBRequest *openrtb2.BidRequest) string {
	if openRTBRequest.Device != nil && openRTBRequest.Device.Geo != nil && openRTBRequest.Device.Geo.Country != "" {
		return convertCountryCode(openRTBRequest.Device.Geo.Country)
	} else if openRTBRequest.User != nil && openRTBRequest.User.Geo != nil && openRTBRequest.User.Geo.Country != "" {
		return convertCountryCode(openRTBRequest.User.Geo.Country)
	} else if openRTBRequest.Device != nil && openRTBRequest.Device.MCCMNC != "" {
		return getCountryCodeFromMCC(openRTBRequest.Device.MCCMNC)
	} else {
		return defaultCountryName
	}
}

// convertCountryCode: ISO 3166-1 Alpha3 -> Alpha2, Some countries may use
func convertCountryCode(country string) (out string) {
	if country == "" {
		return defaultCountryName
	}
	var mapCountryCodeAlpha3ToAlpha2 = map[string]string{"AND": "AD", "AGO": "AO", "AUT": "AT", "BGD": "BD",
		"BLR": "BY", "CAF": "CF", "TCD": "TD", "CHL": "CL", "CHN": "CN", "COG": "CG", "COD": "CD", "DNK": "DK",
		"GNQ": "GQ", "EST": "EE", "GIN": "GN", "GNB": "GW", "GUY": "GY", "IRQ": "IQ", "IRL": "IE", "ISR": "IL",
		"KAZ": "KZ", "LBY": "LY", "MDG": "MG", "MDV": "MV", "MEX": "MX", "MNE": "ME", "MOZ": "MZ", "PAK": "PK",
		"PNG": "PG", "PRY": "PY", "POL": "PL", "PRT": "PT", "SRB": "RS", "SVK": "SK", "SVN": "SI", "SWE": "SE",
		"TUN": "TN", "TUR": "TR", "TKM": "TM", "UKR": "UA", "ARE": "AE", "URY": "UY"}
	if mappedCountry, exists := mapCountryCodeAlpha3ToAlpha2[country]; exists {
		return mappedCountry
	}

	if len(country) >= 2 {
		return country[0:2]
	}

	return defaultCountryName
}

func getCountryCodeFromMCC(MCC string) (out string) {
	var countryMCC = strings.Split(MCC, "-")[0]
	intVar, err := strconv.Atoi(countryMCC)

	if err != nil {
		return defaultCountryName
	} else {
		if result, found := MccList[intVar]; found {
			return strings.ToUpper(result)
		} else {
			return defaultCountryName
		}
	}
}

// getDeviceID include oaid gaid imei. In prebid mobile, use TargetingParams.addUserData("imei", "imei-test");
// When ifa: gaid exists, other device id can be passed by TargetingParams.addUserData("oaid", "oaid-test");
func getDeviceIDFromUserExt(device *device, openRTBRequest *openrtb2.BidRequest) (err error) {
	var userObjExist = true
	if openRTBRequest.User == nil || openRTBRequest.User.Ext == nil {
		userObjExist = false
	}
	if userObjExist {
		var extUserDataHuaweiAds openrtb_ext.ExtUserDataHuaweiAds
		if err := jsonutil.Unmarshal(openRTBRequest.User.Ext, &extUserDataHuaweiAds); err != nil {
			return errors.New("get gaid from openrtb Device.IFA failed, and get device id failed: Unmarshal openRTBRequest.User.Ext -> extUserDataHuaweiAds. Error: " + err.Error())
		}

		var deviceId = extUserDataHuaweiAds.Data
		isValidDeviceId := false

		if len(deviceId.Oaid) > 0 {
			device.Oaid = deviceId.Oaid[0]
			isValidDeviceId = true
		}
		if len(deviceId.Gaid) > 0 {
			device.Gaid = deviceId.Gaid[0]
			isValidDeviceId = true
		}
		if len(device.Gaid) > 0 {
			isValidDeviceId = true
		}
		if len(deviceId.Imei) > 0 {
			device.Imei = deviceId.Imei[0]
			isValidDeviceId = true
		}

		if !isValidDeviceId {
			return errors.New("getDeviceID: Imei ,Oaid, Gaid are all empty.")
		}
		if len(deviceId.ClientTime) > 0 {
			device.ClientTime = getClientTime(deviceId.ClientTime[0])
		}
	} else {
		if len(device.Gaid) == 0 {
			return errors.New("getDeviceID: openRTBRequest.User.Ext is nil and device.Gaid is not specified.")
		}
	}
	return nil
}

// getReqNetWorkInfo: for HuaweiAds request, include Carrier, Mcc, Mnc
func getReqNetWorkInfo(request *huaweiAdsRequest, openRTBRequest *openrtb2.BidRequest) {
	if openRTBRequest.Device != nil {
		var network network
		if openRTBRequest.Device.ConnectionType != nil {
			network.Type = int32(*openRTBRequest.Device.ConnectionType)
		} else {
			network.Type = defaultUnknownNetworkType
		}

		var cellInfos []cellInfo
		if openRTBRequest.Device.MCCMNC != "" {
			var arr = strings.Split(openRTBRequest.Device.MCCMNC, "-")
			network.Carrier = 0
			if len(arr) >= 2 {
				cellInfos = append(cellInfos, cellInfo{
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
		request.Network = network
	}
}

// getReqRegsInfo: get regs information for HuaweiAds request, include Coppa
func getReqRegsInfo(request *huaweiAdsRequest, openRTBRequest *openrtb2.BidRequest) {
	if openRTBRequest.Regs != nil && openRTBRequest.Regs.COPPA >= 0 {
		var regs regs
		regs.Coppa = int32(openRTBRequest.Regs.COPPA)
		request.Regs = regs
	}
}

// getReqGeoInfo: get geo information for HuaweiAds request, include Lon, Lat, Accuracy, Lastfix
func getReqGeoInfo(request *huaweiAdsRequest, openRTBRequest *openrtb2.BidRequest) {
	if openRTBRequest.Device != nil && openRTBRequest.Device.Geo != nil {
		request.Geo = geo{
			Lon:      float32(ptrutil.ValueOrDefault(openRTBRequest.Device.Geo.Lon)),
			Lat:      float32(ptrutil.ValueOrDefault(openRTBRequest.Device.Geo.Lat)),
			Accuracy: int32(openRTBRequest.Device.Geo.Accuracy),
			Lastfix:  int32(openRTBRequest.Device.Geo.LastFix),
		}
	}
}

// getReqGeoInfo: get GDPR consent
func getReqConsentInfo(request *huaweiAdsRequest, openRTBRequest *openrtb2.BidRequest) {
	if openRTBRequest.User != nil && openRTBRequest.User.Ext != nil {
		var extUser openrtb_ext.ExtUser
		if err := jsonutil.Unmarshal(openRTBRequest.User.Ext, &extUser); err != nil {
			return
		}
		request.Consent = extUser.Consent
	}
}

func unmarshalExtImpHuaweiAds(openRTBImp *openrtb2.Imp) (*openrtb_ext.ExtImpHuaweiAds, error) {
	var bidderExt adapters.ExtImpBidder
	var huaweiAdsImpExt openrtb_ext.ExtImpHuaweiAds
	if err := jsonutil.Unmarshal(openRTBImp.Ext, &bidderExt); err != nil {
		return nil, errors.New("Unmarshal: openRTBImp.Ext -> bidderExt failed")
	}
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &huaweiAdsImpExt); err != nil {
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

func checkRespStatusCode(response *adapters.ResponseData) error {
	if response.StatusCode == http.StatusNoContent {
		return nil
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
		return errors.New("bidderRawResponse body is empty")
	}
	return nil
}

func checkHuaweiAdsResponseRetcode(response huaweiAdsResponse) error {
	if response.Retcode == 200 || response.Retcode == 206 {
		return nil
	}
	if response.Retcode == 204 {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("HuaweiAdsResponse retcode: %d , reason: The request packet is correct, but no advertisement was found for this request.", response.Retcode),
		}
	}
	if (response.Retcode < 600 && response.Retcode >= 400) || (response.Retcode < 300 && response.Retcode > 200) {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("HuaweiAdsResponse retcode: %d , reason: %s", response.Retcode, response.Reason),
		}
	}
	return nil
}

// convertHuaweiAdsRespToBidderResp: convert HuaweiAds' response into bidder's response
func (a *adapter) convertHuaweiAdsRespToBidderResp(huaweiAdsResponse *huaweiAdsResponse, openRTBRequest *openrtb2.BidRequest) (bidderResponse *adapters.BidderResponse, err error) {
	if len(huaweiAdsResponse.Multiad) == 0 {
		return nil, errors.New("convert huaweiads response to bidder response failed: multiad length is 0, get no ads from huawei side.")
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
		return nil, errors.New("convert huaweiads response to bidder response failed: openRTBRequest.imp is nil")
	}

	for _, ad30 := range huaweiAdsResponse.Multiad {
		if mapSlotid2Imp[ad30.Slotid].ID == "" {
			continue
		}

		if ad30.Retcode30 != 200 {
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
			bid.NURL = getNurl(content)
			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: mapSlotid2MediaType[ad30.Slotid],
			})
		}
	}
	return bidderResponse, nil
}

func getNurl(content content) string {
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

// handleHuaweiAdsContent: get field Adm, Width, Height
func (a *adapter) handleHuaweiAdsContent(adType int32, content *content, bidType openrtb_ext.BidType, imp openrtb2.Imp) (
	adm string, adWidth int64, adHeight int64, err error) {
	switch bidType {
	case openrtb_ext.BidTypeBanner:
		adm, adWidth, adHeight, err = a.extractAdmBanner(adType, content, bidType, imp)
	case openrtb_ext.BidTypeNative:
		adm, adWidth, adHeight, err = a.extractAdmNative(adType, content, bidType, imp)
	case openrtb_ext.BidTypeVideo:
		adm, adWidth, adHeight, err = a.extractAdmVideo(adType, content, bidType, imp)
	default:
		return "", 0, 0, errors.New("no support bidtype: audio")
	}

	if err != nil {
		return "", 0, 0, fmt.Errorf("generate Adm field from HuaweiAds response failed: %s", err)
	}
	return adm, adWidth, adHeight, nil
}

// extractAdmBanner: banner ad
func (a *adapter) extractAdmBanner(adType int32, content *content, bidType openrtb_ext.BidType, imp openrtb2.Imp) (adm string,
	adWidth int64, adHeight int64, err error) {
	// support openrtb: banner  <=> huawei adtype: banner, interstitial
	if adType != banner && adType != interstitial {
		return "", 0, 0, errors.New("openrtb banner should correspond to huaweiads adtype: banner or interstitial")
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
		return "", 0, 0, errors.New("no banner support creativetype")
	}
}

// extractAdmNative: native ad
func (a *adapter) extractAdmNative(adType int32, content *content, bidType openrtb_ext.BidType, openrtb2Imp openrtb2.Imp) (adm string,
	adWidth int64, adHeight int64, err error) {
	if adType != native {
		return "", 0, 0, errors.New("extract Adm for Native ad: huaweiads response is not a native ad")
	}
	if openrtb2Imp.Native == nil {
		return "", 0, 0, errors.New("extract Adm for Native ad: imp.Native is nil")
	}
	if openrtb2Imp.Native.Request == "" {
		return "", 0, 0, errors.New("extract Adm for Native ad: imp.Native.Request is empty")
	}

	var nativePayload nativeRequests.Request
	if err := jsonutil.Unmarshal(json.RawMessage(openrtb2Imp.Native.Request), &nativePayload); err != nil {
		return "", 0, 0, err
	}

	var nativeResult nativeResponse.Response
	var linkObject nativeResponse.Link
	linkObject.URL, err = a.getClickUrl(content)
	if err != nil {
		return "", 0, 0, err
	}

	nativeResult.Assets = make([]nativeResponse.Asset, 0, len(nativePayload.Assets))
	var imgIndex = 0
	var iconIndex = 0
	for _, asset := range nativePayload.Assets {
		var responseAsset nativeResponse.Asset
		if asset.Title != nil {
			var titleObject nativeResponse.Title
			titleObject.Text = getDecodeValue(content.MetaData.Title)
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
			if len(content.MetaData.ImageInfo) == imgIndex && asset.Img.Type == native1.ImageAssetTypeMain {
				continue
			}
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
				dataObject.Value = getDecodeValue(content.MetaData.Description)
			}

			if asset.Data.Type == native1.DataAssetTypeCTAText {
				dataObject.Type = native1.DataAssetTypeCTAText
				dataObject.Value = getDecodeValue(content.MetaData.Cta)
			}
			responseAsset.Data = &dataObject
		}
		var id = asset.ID
		responseAsset.ID = &id
		nativeResult.Assets = append(nativeResult.Assets, responseAsset)
	}

	// dsp imp click tracking + imp click tracking
	var eventTrackers []nativeResponse.EventTracker
	if content.Monitor != nil {
		for _, monitor := range content.Monitor {
			if len(monitor.Url) == 0 {
				continue
			}
			if monitor.EventType == "click" {
				linkObject.ClickTrackers = append(linkObject.ClickTrackers, monitor.Url...)
			}
			if monitor.EventType == "imp" {
				for i := range monitor.Url {
					var eventTracker nativeResponse.EventTracker
					eventTracker.Event = native1.EventTypeImpression
					eventTracker.Method = native1.EventTrackingMethodImage
					eventTracker.URL = monitor.Url[i]
					eventTrackers = append(eventTrackers, eventTracker)
				}
			}
		}
	}
	nativeResult.EventTrackers = eventTrackers
	nativeResult.Link = linkObject
	nativeResult.Ver = "1.1"
	if nativePayload.Ver != "" {
		nativeResult.Ver = nativePayload.Ver
	}

	var result []byte
	if result, err = jsonEncode(nativeResult); err != nil {
		return "", 0, 0, err
	}
	return strings.Replace(string(result), "\n", "", -1), adWidth, adHeight, nil
}

func getDecodeValue(str string) string {
	if str == "" {
		return ""
	}
	if decodeValue, err := url.QueryUnescape(str); err == nil {
		return decodeValue
	} else {
		return ""
	}
}

func jsonEncode(nativeResult nativeResponse.Response) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(nativeResult)
	return buffer.Bytes(), err
}

// extractAdmPicture: For banner single picture
func (a *adapter) extractAdmPicture(content *content) (adm string, adWidth int64, adHeight int64, err error) {
	if content == nil {
		return "", 0, 0, errors.New("extract Adm failed: content is empty")
	}

	var clickUrl = ""
	clickUrl, err = a.getClickUrl(content)
	if err != nil {
		return "", 0, 0, err
	}

	var imageInfoUrl string
	if content.MetaData.ImageInfo != nil {
		imageInfoUrl = content.MetaData.ImageInfo[0].Url
		adHeight = content.MetaData.ImageInfo[0].Height
		adWidth = content.MetaData.ImageInfo[0].Width
	} else {
		return "", 0, 0, errors.New("content.MetaData.ImageInfo is empty")
	}

	var imageTitle = ""
	imageTitle = getDecodeValue(content.MetaData.Title)
	// dspImp, Imp, dspClick, Click tracking all can be found in MonitorUrl(imp ,click)
	dspImpTrackings, dspClickTrackings := getDspImpClickTrackings(content)
	var dspImpTrackings2StrImg strings.Builder
	for i := 0; i < len(dspImpTrackings); i++ {
		dspImpTrackings2StrImg.WriteString(`<img height="1" width="1" src='`)
		dspImpTrackings2StrImg.WriteString(dspImpTrackings[i])
		dspImpTrackings2StrImg.WriteString(`' >  `)
	}

	adm = "<style> html, body  " +
		"{ margin: 0; padding: 0; width: 100%; height: 100%; vertical-align: middle; }  " +
		"html  " +
		"{ display: table; }  " +
		"body { display: table-cell; vertical-align: middle; text-align: center; -webkit-text-size-adjust: none; }  " +
		"</style> " +
		`<span class="title-link advertiser_label">` + imageTitle + "</span> " +
		"<a href='" + clickUrl + `' style="text-decoration:none" ` +
		"onclick=sendGetReq()> " +
		"<img src='" + imageInfoUrl + "' width='" + strconv.Itoa(int(adWidth)) + "' height='" + strconv.Itoa(int(adHeight)) + "'/> " +
		"</a> " +
		dspImpTrackings2StrImg.String() +
		`<script type="text/javascript">` +
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

// for Interactiontype == appPromotion, clickUrl is intent
func (a *adapter) getClickUrl(content *content) (clickUrl string, err error) {
	if content.Interactiontype == appPromotion {
		if content.MetaData.Intent != "" {
			clickUrl = getDecodeValue(content.MetaData.Intent)
		} else {
			return "", errors.New("content.MetaData.Intent in huaweiads resopnse is empty when interactiontype is appPromotion")
		}
	} else {
		if content.MetaData.ClickUrl != "" {
			clickUrl = content.MetaData.ClickUrl
		} else if content.MetaData.Intent != "" {
			clickUrl = getDecodeValue(content.MetaData.Intent)
		}
	}
	return clickUrl, nil
}

func getDspImpClickTrackings(content *content) (dspImpTrackings []string, dspClickTrackings string) {
	for _, monitor := range content.Monitor {
		if len(monitor.Url) != 0 {
			switch monitor.EventType {
			case "imp":
				dspImpTrackings = monitor.Url
			case "click":
				dspClickTrackings = getStrings(monitor.Url)
			}
		}
	}
	return dspImpTrackings, dspClickTrackings
}

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

// getDuration: millisecond -> format: 00:00:00.000
func getDuration(duration int64) string {
	var dur time.Duration = time.Duration(duration) * time.Millisecond
	t := time.Time{}.Add(dur)
	return t.Format("15:04:05.000")
}

// extractAdmVideo: get field adm for video, vast 3.0
func (a *adapter) extractAdmVideo(adType int32, content *content, bidType openrtb_ext.BidType, opentrb2Imp openrtb2.Imp) (adm string,
	adWidth int64, adHeight int64, err error) {
	if content == nil {
		return "", 0, 0, errors.New("extract Adm for video failed: content is empty")
	}

	var clickUrl = ""
	clickUrl, err = a.getClickUrl(content)
	if err != nil {
		return "", 0, 0, err
	}

	var mime = "video/mp4"
	var resourceUrl = ""
	var duration = ""
	if adType == roll {
		// roll ad get information from mediafile
		if content.MetaData.MediaFile.Mime != "" {
			mime = content.MetaData.MediaFile.Mime
		}
		adWidth = content.MetaData.MediaFile.Width
		adHeight = content.MetaData.MediaFile.Height
		if content.MetaData.MediaFile.Url != "" {
			resourceUrl = content.MetaData.MediaFile.Url
		} else {
			return "", 0, 0, errors.New("extract Adm for video failed: Content.MetaData.MediaFile.Url is empty")
		}
		duration = getDuration(content.MetaData.Duration)
	} else {
		if content.MetaData.VideoInfo.VideoDownloadUrl != "" {
			resourceUrl = content.MetaData.VideoInfo.VideoDownloadUrl
		} else {
			return "", 0, 0, errors.New("extract Adm for video failed: content.MetaData.VideoInfo.VideoDownloadUrl is empty")
		}
		if content.MetaData.VideoInfo.Width != 0 && content.MetaData.VideoInfo.Height != 0 {
			adWidth = int64(content.MetaData.VideoInfo.Width)
			adHeight = int64(content.MetaData.VideoInfo.Height)
		} else if bidType == openrtb_ext.BidTypeVideo {
			if opentrb2Imp.Video != nil && opentrb2Imp.Video.W != nil && *opentrb2Imp.Video.W != 0 && opentrb2Imp.Video.H != nil && *opentrb2Imp.Video.H != 0 {
				adWidth = *opentrb2Imp.Video.W
				adHeight = *opentrb2Imp.Video.H
			}
		} else {
			return "", 0, 0, errors.New("extract Adm for video failed: cannot get video width, height")
		}
		duration = getDuration(int64(content.MetaData.VideoInfo.VideoDuration))
	}

	var adTitle = getDecodeValue(content.MetaData.Title)
	var adId = content.Contentid
	var creativeId = content.Contentid
	var trackingEvents strings.Builder
	var dspImpTracking2Str = ""
	var dspClickTracking2Str = ""
	var errorTracking2Str = ""
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

	// Only for rewarded video
	var rewardedVideoPart = ""
	var isAddRewardedVideoPart = true
	if adType == rewarded {
		var staticImageUrl = ""
		var staticImageHeight = ""
		var staticImageWidth = ""
		var staticImageType = "image/png"
		if len(content.MetaData.Icon) > 0 && content.MetaData.Icon[0].Url != "" {
			staticImageUrl = content.MetaData.Icon[0].Url
			if content.MetaData.Icon[0].Height > 0 && content.MetaData.Icon[0].Width > 0 {
				staticImageHeight = strconv.Itoa(int(content.MetaData.Icon[0].Height))
				staticImageWidth = strconv.Itoa(int(content.MetaData.Icon[0].Width))
			} else {
				staticImageHeight = strconv.Itoa(int(adHeight))
				staticImageWidth = strconv.Itoa(int(adWidth))
			}
		} else if len(content.MetaData.ImageInfo) > 0 && content.MetaData.ImageInfo[0].Url != "" {
			staticImageUrl = content.MetaData.ImageInfo[0].Url
			if content.MetaData.ImageInfo[0].Height > 0 && content.MetaData.ImageInfo[0].Width > 0 {
				staticImageHeight = strconv.Itoa(int(content.MetaData.ImageInfo[0].Height))
				staticImageWidth = strconv.Itoa(int(content.MetaData.ImageInfo[0].Width))
			} else {
				staticImageHeight = strconv.Itoa(int(adHeight))
				staticImageWidth = strconv.Itoa(int(adWidth))
			}
		} else {
			isAddRewardedVideoPart = false
		}
		if isAddRewardedVideoPart {
			rewardedVideoPart = `<Creative adId="` + adId + `" id="` + creativeId + `">` +
				"<CompanionAds>" +
				`<Companion width="` + staticImageWidth + `" height="` + staticImageHeight + `">` +
				`<StaticResource creativeType="` + staticImageType + `"><![CDATA[` + staticImageUrl + `]]></StaticResource>` +
				"<CompanionClickThrough><![CDATA[" + clickUrl + "]]></CompanionClickThrough>" +
				"</Companion>" +
				"</CompanionAds>" +
				"</Creative>"
		}
	}

	adm = `<?xml version="1.0" encoding="UTF-8"?>` +
		`<VAST version="3.0">` +
		`<Ad id="` + adId + `"><InLine>` +
		"<AdSystem>HuaweiAds</AdSystem>" +
		"<AdTitle>" + adTitle + "</AdTitle>" +
		errorTracking2Str + dspImpTracking2Str +
		"<Creatives>" +
		`<Creative adId="` + adId + `" id="` + creativeId + `">` +
		"<Linear>" +
		"<Duration>" + duration + "</Duration>" +
		"<TrackingEvents>" + trackingEvents.String() + "</TrackingEvents>" +
		"<VideoClicks>" +
		"<ClickThrough><![CDATA[" + clickUrl + "]]></ClickThrough>" +
		dspClickTracking2Str +
		"</VideoClicks>" +
		"<MediaFiles>" +
		`<MediaFile delivery="progressive" type="` + mime + `" width="` + strconv.Itoa(int(adWidth)) + `" ` +
		`height="` + strconv.Itoa(int(adHeight)) + `" scalable="true" maintainAspectRatio="true"> ` +
		"<![CDATA[" + resourceUrl + "]]>" +
		"</MediaFile>" +
		"</MediaFiles>" +
		"</Linear>" +
		"</Creative>" + rewardedVideoPart +
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
			trackingUrls.WriteString("<ClickTracking><![CDATA[")
			trackingUrls.WriteString(url)
			trackingUrls.WriteString("]]></ClickTracking>")
		} else if eventType == "imp" {
			trackingUrls.WriteString("<Impression><![CDATA[")
			trackingUrls.WriteString(url)
			trackingUrls.WriteString("]]></Impression>")
		} else if eventType == "vastError" {
			trackingUrls.WriteString("<Error><![CDATA[")
			trackingUrls.WriteString(url)
			trackingUrls.WriteString("&et=[ERRORCODE]]]></Error>")
		}
	}
	return trackingUrls.String()
}

func getVastEventTrackingUrls(urls []string, eventType string) (result string) {
	var trackingUrls strings.Builder
	for _, eventUrl := range urls {
		if eventType == "skip&closeLinear" {
			trackingUrls.WriteString(`<Tracking event="skip"><![CDATA[`)
			trackingUrls.WriteString(eventUrl)
			trackingUrls.WriteString(`]]></Tracking><Tracking event="closeLinear"><![CDATA[`)
			trackingUrls.WriteString(eventUrl)
			trackingUrls.WriteString("]]></Tracking>")
		} else {
			trackingUrls.WriteString(`<Tracking event="`)
			trackingUrls.WriteString(eventType)
			trackingUrls.WriteString(`"><![CDATA[`)
			trackingUrls.WriteString(eventUrl)
			trackingUrls.WriteString("]]></Tracking>")
		}
	}
	return trackingUrls.String()
}

func computeHmacSha256(message string, signKey string) string {
	h := hmac.New(sha256.New, []byte(signKey))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// getDigestAuthorization: get digest authorization for request header
func getDigestAuthorization(huaweiAdsImpExt *openrtb_ext.ExtImpHuaweiAds, isTestAuthorization bool) string {
	var nonce = strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
	// this is for test case, time 2021/8/20 19:30
	if isTestAuthorization {
		nonce = "1629473330823"
	}
	publisher_id := strings.TrimSpace(huaweiAdsImpExt.PublisherId)
	sign_key := strings.TrimSpace(huaweiAdsImpExt.SignKey)
	key_id := strings.TrimSpace(huaweiAdsImpExt.KeyId)

	var apiKey = publisher_id + ":ppsadx/getResult:" + sign_key
	return "Digest username=" + publisher_id + "," +
		"realm=ppsadx/getResult," +
		"nonce=" + nonce + "," +
		"response=" + computeHmacSha256(nonce+":POST:/ppsadx/getResult", apiKey) + "," +
		"algorithm=HmacSHA256,usertype=1,keyid=" + key_id
}
