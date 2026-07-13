package unitylevelplay

import (
	"encoding/base64"

	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	jsoniter "github.com/json-iterator/go"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/metrics"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/sdk/sdkutils"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

var jsoniterator = jsoniter.ConfigCompatibleWithStandardLibrary

type LevelPlay struct {
	metricsEngine metrics.MetricsEngine
	publisherId   string
	profileId     string
}

func NewLevelPlay(metricsEngine metrics.MetricsEngine) *LevelPlay {
	return &LevelPlay{
		metricsEngine: metricsEngine,
	}
}

func (l *LevelPlay) ModifyRequestWithUnityLevelPlayParams(requestBody []byte) []byte {
	if len(requestBody) == 0 {
		return nil
	}

	request := &openrtb2.BidRequest{}
	if err := jsoniterator.Unmarshal(requestBody, request); err != nil {
		glog.Errorf("[UnityLevelPlay] [Error]: failed to unmarshal request body with error %v and request body %s", err, string(requestBody))
		return requestBody
	}

	// modify request with static data
	l.modifyRequestWithStaticData(request)

	// Set publisher id
	if request.App != nil && request.App.Publisher != nil {
		l.publisherId = request.App.Publisher.ID
	}

	// Set profile id
	if profileID, _, _, _ := jsonparser.Get(request.Ext, "prebid", "bidderparams", "pubmatic", "wrapper", "profileid"); profileID != nil {
		l.profileId = string(profileID)
	}

	// modify request with signal data
	l.modifyRequestWithSignalData(request)

	modifiedRequest, err := jsoniterator.Marshal(request)
	if err != nil {
		return requestBody
	}

	return modifiedRequest
}

func (l *LevelPlay) modifyRequestWithStaticData(request *openrtb2.BidRequest) {
	if request == nil {
		return
	}

	if len(request.Imp) > 0 {
		// Set imp.instl and imp.rwdd as 1 when video.ext.reward is 1
		if request.Imp[0].Video != nil && request.Imp[0].Video.Ext != nil {
			reward, err := jsonparser.GetInt(request.Imp[0].Video.Ext, "reward")
			if reward == 1 && err == nil {
				request.Imp[0].Instl = 1
				request.Imp[0].Rwdd = 1
				// remove banner
				request.Imp[0].Banner = nil
			}
		}

		// Set imp.secure as 1
		request.Imp[0].Secure = ptrutil.ToPtr(int8(1))

		// Remove native from request
		request.Imp[0].Native = nil

		// Remove video from request
		request.Imp[0].Video = nil
	}

	if request.App != nil {
		// delete app.ext.sessionDepth
		request.App.Ext = jsonparser.Delete(request.App.Ext, "sessionDepth")
	}
}

func (l *LevelPlay) modifyRequestWithSignalData(request *openrtb2.BidRequest) {
	if request == nil || request.App == nil || request.App.Ext == nil {
		return
	}

	token, err := jsonparser.GetString(request.App.Ext, "token")
	if token == "" || err != nil {
		l.metricsEngine.RecordSignalDataStatus(l.publisherId, l.profileId, models.MissingSignal)
		return
	}

	// decode token
	signalData, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		l.metricsEngine.RecordSignalDataStatus(l.publisherId, l.profileId, models.InvalidSignal)
		return
	}

	var signal *openrtb2.BidRequest
	if err := jsoniterator.Unmarshal(signalData, &signal); err != nil || signal == nil {
		l.metricsEngine.RecordSignalDataStatus(l.publisherId, l.profileId, models.InvalidSignal)
		return
	}

	modifyImpression(request, signal.Imp)
	modifyRegs(request, signal.Regs)
	modifyApp(request, signal.App)
	modifyDevice(request, signal.Device)
	modifyUser(request, signal.User)
	modifySource(request, signal.Source)

	// Request Ext
	request.Ext, _ = sdkutils.CopyPath(signal.Ext, request.Ext, "wrapper", "clientconfig")
}

func modifyBanner(requestBanner *openrtb2.Banner, signalBanner *openrtb2.Banner) {
	if requestBanner == nil || signalBanner == nil {
		return
	}

	if signalBanner.API != nil {
		requestBanner.API = signalBanner.API
	}

}

func modifyImpression(request *openrtb2.BidRequest, signalImps []openrtb2.Imp) {
	if len(request.Imp) == 0 || len(signalImps) == 0 {
		return
	}

	// read secure from signal
	if signalImps[0].Secure != nil {
		request.Imp[0].Secure = signalImps[0].Secure
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

	// modify banner
	modifyBanner(request.Imp[0].Banner, signalImps[0].Banner)

	// modify video
	if signalImps[0].Video != nil {
		request.Imp[0].Video = signalImps[0].Video
	}

	// modify native
	request.Imp[0].Native = nil
	if signalImps[0].Native != nil {
		request.Imp[0].Native = signalImps[0].Native
	}

	// modify ext
	request.Imp[0].Ext = modifyImpExtension(request.Imp[0].Ext, signalImps[0].Ext)
}

func modifyImpExtension(requestImpExt, signalImpExt []byte) []byte {
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
	requestImpExt, _ = sdkutils.CopyPath(signalImpExt, requestImpExt, "skadn", "skadnetids")
	requestImpExt, _ = sdkutils.CopyPath(signalImpExt, requestImpExt, "gpid")
	requestImpExt, _ = sdkutils.CopyPath(signalImpExt, requestImpExt, "owsdk")

	return requestImpExt
}

func modifyRegs(request *openrtb2.BidRequest, signalRegs *openrtb2.Regs) {
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

func modifyApp(request *openrtb2.BidRequest, signalApp *openrtb2.App) {
	if signalApp == nil {
		return
	}

	if request.App == nil {
		request.App = &openrtb2.App{}
	}

	if len(signalApp.Domain) > 0 {
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

func modifyDevice(request *openrtb2.BidRequest, signalDevice *openrtb2.Device) {
	if signalDevice == nil {
		return
	}

	request.Device = sdkutils.MergeDevice(request.Device, signalDevice)

	request.Device.Ext, _ = sdkutils.CopyPath(signalDevice.Ext, request.Device.Ext, "atts")
	request.Device.Ext = sdkutils.CopyIFV(signalDevice.Ext, request.Device.Ext)
}

func modifyUser(request *openrtb2.BidRequest, signalUser *openrtb2.User) {
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

func modifySource(request *openrtb2.BidRequest, signalSource *openrtb2.Source) {
	if signalSource == nil {
		return
	}

	if request.Source == nil {
		request.Source = &openrtb2.Source{}
	}

	request.Source.Ext, _ = sdkutils.CopyPath(signalSource.Ext, request.Source.Ext, "omidpn")
	request.Source.Ext, _ = sdkutils.CopyPath(signalSource.Ext, request.Source.Ext, "omidpv")
}
