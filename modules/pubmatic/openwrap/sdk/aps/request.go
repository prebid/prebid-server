package aps

import (
	"slices"

	"github.com/buger/jsonparser"
	jsoniter "github.com/json-iterator/go"
	adcom1 "github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/metrics"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/sdk/sdkutils"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

const (
	apsAdFormatBanner       = "banner"
	apsAdFormatMrec         = "mrec"
	apsAdFormatInterstitial = "interstitial"
	apsAdFormatRewarded     = "rewarded"

	apsMrecWidth  int64 = 300
	apsMrecHeight int64 = 250
)

type apsVideoFields struct {
	W           *int64
	H           *int64
	Pos         *adcom1.PlacementPosition
	BAttr       []adcom1.CreativeAttribute
	CompanionAd []openrtb2.Banner
}

type apsBannerFields struct {
	W      *int64
	H      *int64
	Format []openrtb2.Format
	Pos    *adcom1.PlacementPosition
}

// apsImpMediaFields holds impression media captured from the incoming APS request
// before signal merge mutates banner/video objects.
type apsImpMediaFields struct {
	video        *apsVideoFields
	banner       *apsBannerFields
	videoMissing bool
}

var jsoniterator = jsoniter.ConfigCompatibleWithStandardLibrary

type Aps struct {
	metricsEngine metrics.MetricsEngine
	publisherId   string
	profileId     string
}

func NewAPS(metricsEngine metrics.MetricsEngine) *Aps {
	return &Aps{
		metricsEngine: metricsEngine,
	}
}

// ModifyRequestWithAPSParams merges APS signal into the request. rctx is a pointer so the decoded
// signal bid request can be stored on rCtx.SignalRequest for PubMatic-only EDS at before_validation.
func (a *Aps) ModifyRequestWithAPSParams(requestBody []byte, rctx *models.RequestCtx) []byte {
	if len(requestBody) == 0 {
		return requestBody
	}
	request := &openrtb2.BidRequest{}
	if err := jsoniterator.Unmarshal(requestBody, request); err != nil {
		return requestBody
	}

	// Capture ad format and impression media from the incoming APS request before any
	// modifications; signal merge replaces banner/video objects later in the pipeline.
	adFormat := ""
	var apsMedia apsImpMediaFields
	if len(request.Imp) > 0 {
		adFormat = determineAdFormat(&request.Imp[0])
		apsMedia = captureApsImpMediaFields(&request.Imp[0])
	}

	// modify request with static data
	a.modifyRequestWithStaticData(request)
	// Set publisher id
	if request.App != nil && request.App.Publisher != nil {
		a.publisherId = request.App.Publisher.ID
	}

	// Set profile id
	if profileID, _, _, _ := jsonparser.Get(request.Ext, "prebid", "bidderparams", "pubmatic", "wrapper", "profileid"); profileID != nil {
		a.profileId = string(profileID)
	}

	// modify request with signal data
	a.modifyRequestWithSignalData(request, rctx, adFormat, apsMedia)
	modifiedRequest, err := jsoniterator.Marshal(request)
	if err != nil {
		return requestBody
	}
	return modifiedRequest
}

func (a *Aps) modifyRequestWithStaticData(request *openrtb2.BidRequest) {
	if request == nil {
		return
	}

	if len(request.Imp) > 0 {
		// Set rwdd as 1 when video.ext.videotype is rewarded
		if request.Imp[0].Video != nil && request.Imp[0].Video.Ext != nil {
			reward, err := jsonparser.GetString(request.Imp[0].Video.Ext, "videotype")
			if reward == "rewarded" && err == nil {
				request.Imp[0].Rwdd = 1
				// remove banner
				request.Imp[0].Banner = nil
			}
		}

		// Always set secure to 1
		request.Imp[0].Secure = ptrutil.ToPtr(int8(1))

		// Remove native from request
		request.Imp[0].Native = nil

	}

}

func (a *Aps) modifyRequestWithSignalData(request *openrtb2.BidRequest, rctx *models.RequestCtx, adFormat string, apsMedia apsImpMediaFields) {
	if request == nil || request.User == nil {
		return
	}

	signal := request.User.BuyerUID
	if signal == "" {
		a.metricsEngine.RecordSignalDataStatus(a.publisherId, a.profileId, models.MissingSignal)
		return
	}

	var signalRequest *openrtb2.BidRequest
	if err := jsoniterator.Unmarshal([]byte(signal), &signalRequest); err != nil || signalRequest == nil {
		a.metricsEngine.RecordSignalDataStatus(a.publisherId, a.profileId, models.InvalidSignal)
		return
	}

	// Keep decoded signal for EDS; signal ext.eds is not merged onto the shared request ext.
	if rctx != nil {
		rctx.SignalRequest = signalRequest
	}

	updateImpressionWithSignalAndApsMedia(request, signalRequest.Imp, adFormat, apsMedia)

	updateRegs(request, signalRequest.Regs)
	updateApp(request, signalRequest.App)
	updateDevice(request, signalRequest.Device)
	updateUser(request, signalRequest.User)
	updateSource(request, signalRequest.Source)

	// Request Ext
	request.Ext, _ = sdkutils.CopyPath(signalRequest.Ext, request.Ext, "wrapper", "clientconfig")

	applyAdFormatModifications(request, adFormat, signalRequest.Ext)

	// Embedded signal lives in buyeruid; merged device/app/imp/user/source/regs stay—do not forward raw JSON to partners.
	if request.User != nil {
		request.User.BuyerUID = ""
	}
}

func determineAdFormat(imp *openrtb2.Imp) string {
	if imp == nil {
		return ""
	}

	if isRewardedImp(imp) {
		return apsAdFormatRewarded
	}

	if imp.Instl == 1 {
		return apsAdFormatInterstitial
	}

	if isMrecBanner(imp.Banner) || isMrecVideo(imp.Video) {
		return apsAdFormatMrec
	}

	if isBannerFormat(imp.Banner) {
		return apsAdFormatBanner
	}

	return ""
}

func isMrecBanner(banner *openrtb2.Banner) bool {
	if banner == nil {
		return false
	}

	if hasMrecDimensions(banner.W, banner.H) {
		return true
	}

	for _, format := range banner.Format {
		if format.W == apsMrecWidth && format.H == apsMrecHeight {
			return true
		}
	}

	return false
}

func isMrecVideo(video *openrtb2.Video) bool {
	if video == nil {
		return false
	}
	return hasMrecDimensions(video.W, video.H)
}

func hasMrecDimensions(w, h *int64) bool {
	return w != nil && h != nil && *w == apsMrecWidth && *h == apsMrecHeight
}

func isBannerFormat(banner *openrtb2.Banner) bool {
	if banner == nil {
		return false
	}
	return true
}

func isRewardedImp(imp *openrtb2.Imp) bool {
	if imp.Rwdd == 1 {
		return true
	}

	if imp.Video != nil && imp.Video.Ext != nil {
		if reward, err := jsonparser.GetString(imp.Video.Ext, "videotype"); err == nil && reward == models.TypeRewarded {
			return true
		}
	}

	return false
}

func captureApsImpMediaFields(imp *openrtb2.Imp) apsImpMediaFields {
	if imp == nil {
		return apsImpMediaFields{}
	}

	return apsImpMediaFields{
		videoMissing: imp.Video == nil,
		video:        captureApsVideoFields(imp.Video),
		banner:       captureApsBannerFields(imp.Banner),
	}
}

func captureApsVideoFields(video *openrtb2.Video) *apsVideoFields {
	if video == nil {
		return nil
	}

	fields := &apsVideoFields{
		W:     video.W,
		H:     video.H,
		Pos:   video.Pos,
		BAttr: slices.Clone(video.BAttr),
	}

	if len(video.CompanionAd) > 0 {
		fields.CompanionAd = make([]openrtb2.Banner, len(video.CompanionAd))
		for i, companion := range video.CompanionAd {
			fields.CompanionAd[i].Pos = companion.Pos
			if len(companion.Format) > 0 {
				fields.CompanionAd[i].Format = slices.Clone(companion.Format)
			}
		}
	}

	return fields
}

func captureApsBannerFields(banner *openrtb2.Banner) *apsBannerFields {
	if banner == nil {
		return nil
	}

	fields := &apsBannerFields{
		W:      banner.W,
		H:      banner.H,
		Pos:    banner.Pos,
		Format: slices.Clone(banner.Format),
	}
	return fields
}

func isMrecOrInterstitialAdFormat(adFormat string) bool {
	return adFormat == apsAdFormatMrec || adFormat == apsAdFormatInterstitial
}

func isVideoSupportingAdFormat(adFormat string) bool {
	return adFormat == apsAdFormatMrec || adFormat == apsAdFormatInterstitial || adFormat == apsAdFormatRewarded
}

// createBannerFromApsVideoIfMissing creates a banner from captured APS video fields when the imp has none.
// Only used for mrec and interstitial ad formats.
func createBannerFromApsVideoIfMissing(imp *openrtb2.Imp, adFormat string, apsVideo *apsVideoFields) {
	if imp == nil || !isMrecOrInterstitialAdFormat(adFormat) || imp.Banner != nil || apsVideo == nil {
		return
	}

	imp.Banner = &openrtb2.Banner{}
	imp.Banner.W = ptrutil.Clone(apsVideo.W)
	imp.Banner.H = ptrutil.Clone(apsVideo.H)
	imp.Banner.Pos = apsVideo.Pos
	if len(apsVideo.CompanionAd) > 0 {
		imp.Banner.Format = slices.Clone(apsVideo.CompanionAd[0].Format)
	}
}

// applyApsBannerFieldsToVideo applies captured APS banner sizing to the video object, creating video if absent.
// Used for mrec, interstitial, and rewarded when the original APS request had no video.
func applyApsBannerFieldsToVideo(imp *openrtb2.Imp, adFormat string, apsBanner *apsBannerFields) {
	if imp == nil || apsBanner == nil || !isVideoSupportingAdFormat(adFormat) {
		return
	}

	if imp.Video == nil {
		imp.Video = &openrtb2.Video{}
	}
	video := imp.Video
	video.W = ptrutil.Clone(apsBanner.W)
	video.H = ptrutil.Clone(apsBanner.H)
	companion := ensureCompanionAd(video)
	companion.Format = slices.Clone(apsBanner.Format)
	//don't set pos for mrec
	if adFormat != apsAdFormatMrec {
		companion.Pos = apsBanner.Pos
		video.Pos = apsBanner.Pos
	}
}

// updateImpressionWithSignalAndApsMedia merges signal impression data and reconciles mrec/interstitial/rewarded
// banner/video objects with captured APS request fields. Keep these steps together; order matters:
//  1. Create banner from APS video fields when missing (mrec/interstitial only; must run before signal banner merge).
//  2. Merge signal impression fields (banner mimes/api, full video object from signal).
//  3. Create or overlay video from APS banner fields when the original request had no video (mrec/interstitial/rewarded).
func updateImpressionWithSignalAndApsMedia(request *openrtb2.BidRequest, signalImps []openrtb2.Imp, adFormat string, apsMedia apsImpMediaFields) {
	if len(request.Imp) == 0 {
		return
	}

	createBannerFromApsVideoIfMissing(&request.Imp[0], adFormat, apsMedia.video)
	updateImpressionWithSignal(request, signalImps, adFormat, apsMedia.video)
	if apsMedia.videoMissing {
		applyApsBannerFieldsToVideo(&request.Imp[0], adFormat, apsMedia.banner)
	}
}

func updateImpressionWithSignal(request *openrtb2.BidRequest, signalImps []openrtb2.Imp, adFormat string, apsVideo *apsVideoFields) {
	if len(request.Imp) == 0 || len(signalImps) == 0 {
		return
	}

	// set instl to 1 for interstitial and rewarded; 0 for others
	if adFormat == apsAdFormatInterstitial || adFormat == apsAdFormatRewarded {
		request.Imp[0].Instl = 1
	} else if adFormat != "" {
		request.Imp[0].Instl = 0
	}

	if signalImps[0].Exp > 0 {
		request.Imp[0].Exp = signalImps[0].Exp
	}

	if signalImps[0].DisplayManager != "" {
		request.Imp[0].DisplayManager = signalImps[0].DisplayManager
	}

	if signalImps[0].DisplayManagerVer != "" {
		request.Imp[0].DisplayManagerVer = signalImps[0].DisplayManagerVer
	}

	if signalImps[0].ClickBrowser != nil {
		request.Imp[0].ClickBrowser = signalImps[0].ClickBrowser
	}

	// modify banner with signal banner
	sdkutils.MergeBanner(request.Imp[0].Banner, signalImps[0].Banner)

	// Create video object from signal if adformat is not banner; restore APS video fields w/h/pos/companion and battr
	if signalImps[0].Video != nil && adFormat != apsAdFormatBanner {
		request.Imp[0].Video = signalImps[0].Video
		restoreApsVideoFields(request.Imp[0].Video, apsVideo, adFormat)
	}

	request.Imp[0].Ext = updateImpExtension(request.Imp[0].Ext, signalImps[0].Ext)
}

func getExtendedSignalForFormat(signalExt []byte, adFormat string) []byte {
	if len(signalExt) == 0 || adFormat == "" {
		return nil
	}

	extSignal, _, _, err := jsonparser.Get(signalExt, "extendedsignal", adFormat)
	if err != nil || len(extSignal) == 0 {
		return nil
	}

	return extSignal
}

func applyAdFormatModifications(request *openrtb2.BidRequest, adFormat string, signalExt []byte) {
	if len(request.Imp) == 0 || adFormat == "" {
		return
	}

	extSignal := getExtendedSignalForFormat(signalExt, adFormat)
	if extSignal == nil {
		return
	}

	imp := &request.Imp[0]

	switch adFormat {
	case apsAdFormatBanner:
		imp.Video = nil
	case apsAdFormatRewarded:
		imp.Banner = nil
		fallthrough
	case apsAdFormatMrec, apsAdFormatInterstitial:
		applyVideoFieldsFromExtendedSignal(imp.Video, extSignal)
	default:
		return
	}

	setUserExtFromExtendedSignal(request, extSignal)
	if adFormat != apsAdFormatBanner {
		setImpExtFromExtendedSignal(request, extSignal, request.Device)
	}
}

func setUserExtFromExtendedSignal(request *openrtb2.BidRequest, extSignal []byte) {
	if request.User == nil {
		request.User = &openrtb2.User{}
	}

	request.User.Ext = sdkutils.SetIfKeysExists(extSignal, request.User.Ext, "impdepth", "lastadomain")
}

func setImpExtFromExtendedSignal(request *openrtb2.BidRequest, extSignal []byte, device *openrtb2.Device) {
	if len(request.Imp) == 0 {
		return
	}

	impExt := request.Imp[0].Ext
	impExt = setImpExtFieldFromExtSignal(impExt, extSignal, "ctaoverlay", "owsdk", "ctaoverlay")
	if sdkutils.IsIOSDevice(device) {
		impExt = setImpExtFieldFromExtSignal(impExt, extSignal, "skoverlay", "skadn", "skoverlay")
	}
	request.Imp[0].Ext = impExt
}

func setImpExtFieldFromExtSignal(impExt, extSignal []byte, signalKey string, targetPath ...string) []byte {
	value, dataType, _, err := jsonparser.Get(extSignal, signalKey)
	if value == nil || err != nil {
		return impExt
	}

	if impExt == nil {
		impExt = []byte(`{}`)
	}

	switch dataType {
	case jsonparser.String:
		if len(value) == 0 {
			return impExt
		}
		value = []byte(`"` + string(value) + `"`)
	case jsonparser.Array, jsonparser.Object:
		if len(value) <= 2 {
			return impExt
		}
	}

	result, err := jsonparser.Set(impExt, value, targetPath...)
	if err != nil {
		return impExt
	}

	return result
}

func applyVideoFieldsFromExtendedSignal(video *openrtb2.Video, extSignal []byte) {
	if video == nil {
		return
	}

	if placement, err := jsonparser.GetInt(extSignal, "videoplacement"); err == nil {
		video.Placement = adcom1.VideoPlacementSubtype(placement)
	}

	if plcmt, err := jsonparser.GetInt(extSignal, "videoplcmt"); err == nil {
		video.Plcmt = adcom1.VideoPlcmtSubtype(plcmt)
	}

	companion := ensureCompanionAd(video)

	if apiData, _, _, err := jsonparser.Get(extSignal, "companionapi"); err == nil {
		if apis := parseAPIFrameworks(apiData); len(apis) > 0 {
			companion.API = apis
		}
	}
}

func restoreApsVideoFields(video *openrtb2.Video, apsVideo *apsVideoFields, adFormat string) {
	if video == nil || apsVideo == nil {
		return
	}

	video.W = ptrutil.Clone(apsVideo.W)
	video.H = ptrutil.Clone(apsVideo.H)

	//don't set pos for mrec
	if adFormat != apsAdFormatMrec {
		video.Pos = apsVideo.Pos
	}

	if len(apsVideo.BAttr) > 0 {
		video.BAttr = slices.Clone(apsVideo.BAttr)
	}

	if len(apsVideo.CompanionAd) == 0 {
		return
	}

	if len(video.CompanionAd) == 0 {
		video.CompanionAd = make([]openrtb2.Banner, len(apsVideo.CompanionAd))
	}

	for i := range apsVideo.CompanionAd {
		if i >= len(video.CompanionAd) {
			video.CompanionAd = append(video.CompanionAd, openrtb2.Banner{})
		}

		if len(apsVideo.CompanionAd[i].Format) > 0 {
			video.CompanionAd[i].Format = slices.Clone(apsVideo.CompanionAd[i].Format)
		}

		//don't set pos for mrec
		if adFormat != apsAdFormatMrec {
			video.CompanionAd[i].Pos = apsVideo.CompanionAd[i].Pos
		}
	}
}

func ensureCompanionAd(video *openrtb2.Video) *openrtb2.Banner {
	if video == nil {
		return nil
	}

	if len(video.CompanionAd) == 0 {
		video.CompanionAd = []openrtb2.Banner{{}}
	}

	return &video.CompanionAd[0]
}

func parseAPIFrameworks(data []byte) []adcom1.APIFramework {
	var apis []adcom1.APIFramework

	_, _ = jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, _ int, err error) {
		if err != nil || dataType != jsonparser.Number {
			return
		}

		if api, err := jsonparser.ParseInt(value); err == nil {
			apis = append(apis, adcom1.APIFramework(api))
		}
	})

	return apis
}

func updateImpExtension(requestImpExt, signalImpExt []byte) []byte {
	if signalImpExt == nil {
		return requestImpExt
	}

	if len(requestImpExt) == 0 {
		requestImpExt = []byte(`{}`)
	}

	requestImpExt, _ = sdkutils.CopyPath(signalImpExt, requestImpExt, "skadn", "versions")
	requestImpExt, _ = sdkutils.CopyPath(signalImpExt, requestImpExt, "skadn", "version")
	requestImpExt, _ = sdkutils.CopyPath(signalImpExt, requestImpExt, "skadn", "productpage")
	requestImpExt, _ = sdkutils.CopyPath(signalImpExt, requestImpExt, "owsdk")

	return requestImpExt
}

func updateRegs(request *openrtb2.BidRequest, signalRegs *openrtb2.Regs) {
	if signalRegs == nil {
		return
	}

	if request.Regs == nil {
		request.Regs = &openrtb2.Regs{}
	}

	if signalRegs.COPPA > 0 {
		request.Regs.COPPA = signalRegs.COPPA
	}

	request.Regs.Ext, _ = sdkutils.CopyPath(signalRegs.Ext, request.Regs.Ext, "gpp")
	request.Regs.Ext, _ = sdkutils.CopyPath(signalRegs.Ext, request.Regs.Ext, "gpp_sid")
	request.Regs.Ext, _ = sdkutils.CopyPath(signalRegs.Ext, request.Regs.Ext, "gdpr")
	request.Regs.Ext, _ = sdkutils.CopyPath(signalRegs.Ext, request.Regs.Ext, "us_privacy")
	request.Regs.Ext, _ = sdkutils.CopyPath(signalRegs.Ext, request.Regs.Ext, "dsa", "dsarequired")
	request.Regs.Ext, _ = sdkutils.CopyPath(signalRegs.Ext, request.Regs.Ext, "dsa", "pubrender")
	request.Regs.Ext, _ = sdkutils.CopyPath(signalRegs.Ext, request.Regs.Ext, "dsa", "datatopub")
}

func updateApp(request *openrtb2.BidRequest, signalApp *openrtb2.App) {
	if signalApp == nil {
		return
	}

	if request.App == nil {
		request.App = &openrtb2.App{}
	}

	if len(request.App.Domain) == 0 && len(signalApp.Domain) > 0 {
		request.App.Domain = signalApp.Domain
	}

	if len(signalApp.Cat) > 0 {
		request.App.Cat = signalApp.Cat
	}

	if signalApp.Paid != nil {
		request.App.Paid = signalApp.Paid
	}

	if len(signalApp.Keywords) > 0 {
		request.App.Keywords = signalApp.Keywords
	}

	if signalApp.Name != "" {
		request.App.Name = signalApp.Name
	}

	if signalApp.Ver != "" {
		request.App.Ver = signalApp.Ver
	}

	if len(request.App.StoreURL) == 0 {
		request.App.StoreURL = signalApp.StoreURL
	}
}

func updateDevice(request *openrtb2.BidRequest, signalDevice *openrtb2.Device) {
	if signalDevice == nil {
		return
	}

	request.Device = sdkutils.MergeDevice(request.Device, signalDevice)

	request.Device.Ext, _ = sdkutils.CopyPath(signalDevice.Ext, request.Device.Ext, "atts")
	request.Device.Ext = sdkutils.CopyIFV(signalDevice.Ext, request.Device.Ext)
}

func updateUser(request *openrtb2.BidRequest, signalUser *openrtb2.User) {
	if signalUser == nil {
		return
	}

	if request.User == nil {
		request.User = &openrtb2.User{}
	}

	if signalUser.Data != nil {
		request.User.Data = signalUser.Data
	}

	if signalUser.Yob > 0 {
		request.User.Yob = signalUser.Yob
	}

	if signalUser.Gender != "" {
		request.User.Gender = signalUser.Gender
	}

	if signalUser.Keywords != "" {
		request.User.Keywords = signalUser.Keywords
	}

	request.User.Ext, _ = sdkutils.CopyPath(signalUser.Ext, request.User.Ext, "sessionduration")
	request.User.Ext, _ = sdkutils.CopyPath(signalUser.Ext, request.User.Ext, "consent")
	request.User.Ext, _ = sdkutils.CopyPath(signalUser.Ext, request.User.Ext, "eids")
}

func updateSource(request *openrtb2.BidRequest, signalSource *openrtb2.Source) {
	if signalSource == nil {
		return
	}

	if request.Source == nil {
		request.Source = &openrtb2.Source{}
	}

	request.Source.Ext, _ = sdkutils.CopyPath(signalSource.Ext, request.Source.Ext, "omidpn")
	request.Source.Ext, _ = sdkutils.CopyPath(signalSource.Ext, request.Source.Ext, "omidpv")
}
