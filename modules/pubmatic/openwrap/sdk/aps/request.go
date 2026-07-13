package aps

import (
	"github.com/buger/jsonparser"
	jsoniter "github.com/json-iterator/go"
	adcom1 "github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/metrics"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/sdk/sdkutils"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

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
func (a *Aps) ModifyRequestWithAPSParams(requestBody []byte, rctx models.RequestCtx) []byte {
	if len(requestBody) == 0 {
		return requestBody
	}
	request := &openrtb2.BidRequest{}
	if err := jsoniterator.Unmarshal(requestBody, request); err != nil {
		return requestBody
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
	a.modifyRequestWithSignalData(request)
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

func (a *Aps) modifyRequestWithSignalData(request *openrtb2.BidRequest) {
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

	updateImpression(request, signalRequest.Imp)
	updateRegs(request, signalRequest.Regs)
	updateApp(request, signalRequest.App)
	updateDevice(request, signalRequest.Device)
	updateUser(request, signalRequest.User)
	updateSource(request, signalRequest.Source)

	// Request Ext
	request.Ext, _ = sdkutils.CopyPath(signalRequest.Ext, request.Ext, "wrapper", "clientconfig")

	// Embedded signal lives in buyeruid; merged device/app/imp/user/source/regs stay—do not forward raw JSON to partners.
	if request.User != nil {
		request.User.BuyerUID = ""
	}
}

func modifyBanner(requestBanner *openrtb2.Banner, signalBanner *openrtb2.Banner) {
	if requestBanner == nil || signalBanner == nil {
		return
	}

	if signalBanner.API != nil {
		requestBanner.API = signalBanner.API
	}

}

func updateImpression(request *openrtb2.BidRequest, signalImps []openrtb2.Imp) {
	if len(request.Imp) == 0 || len(signalImps) == 0 {
		return
	}

	request.Imp[0].Instl = signalImps[0].Instl

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

	// modify banner
	modifyBanner(request.Imp[0].Banner, signalImps[0].Banner)

	// modify video
	// Update video (replace entire video object from signal except battr)
	var battrVideo []adcom1.CreativeAttribute
	if request.Imp[0].Video != nil && len(request.Imp[0].Video.BAttr) > 0 {
		battrVideo = make([]adcom1.CreativeAttribute, len(request.Imp[0].Video.BAttr))
		copy(battrVideo, request.Imp[0].Video.BAttr)
	}

	if signalImps[0].Video != nil {
		//setting complete video object from signal, except video.battr
		request.Imp[0].Video = signalImps[0].Video
		if len(battrVideo) > 0 {
			request.Imp[0].Video.BAttr = battrVideo
		}
	}

	// modify ext
	request.Imp[0].Ext = updateImpExtension(request.Imp[0].Ext, signalImps[0].Ext)
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
	requestImpExt, _ = sdkutils.CopyPath(signalImpExt, requestImpExt, "skadn", "skoverlay")
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
	request.User.Ext, _ = sdkutils.CopyPath(signalUser.Ext, request.User.Ext, "impdepth")
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
