package googlesdk

import (
	"encoding/base64"
	"errors"
	"strconv"

	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	jsoniter "github.com/json-iterator/go"
	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/feature"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/sdk/sdkutils"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

const (
	androidAppId                      = "com.google.ads.mediation.pubmatic.PubMaticMediationAdapter"
	iOSAppId                          = "GADMediationAdapterPubMatic"
	consentedProvidersSettingsListKey = "consented_providers_settings"
	consentedProvidersKey             = "consented_providers"
)

var jsoniterator = jsoniter.ConfigCompatibleWithStandardLibrary

type wrapperData struct {
	PublisherId string
	ProfileId   string
	TagId       string
}

func (wd *wrapperData) setProfileID(request *openrtb2.BidRequest) {
	if len(wd.ProfileId) == 0 {
		return
	}

	if request.Ext == nil {
		request.Ext = []byte(`{}`)
	}

	request.Ext, _ = jsonparser.Set(request.Ext, []byte(wd.ProfileId), "prebid", "bidderparams", "pubmatic", "wrapper", "profileid")
}

func (wd *wrapperData) setPublisherId(request *openrtb2.BidRequest) {
	if len(wd.PublisherId) == 0 {
		return
	}

	if request.App == nil {
		request.App = &openrtb2.App{}
	}

	if request.App.Publisher == nil {
		request.App.Publisher = &openrtb2.Publisher{}
	}

	request.App.Publisher.ID = wd.PublisherId
}

func (wd *wrapperData) setTagId(request *openrtb2.BidRequest) {
	if len(wd.TagId) == 0 || len(request.Imp) == 0 {
		return
	}

	request.Imp[0].TagID = wd.TagId
}

func getSignalData(body []byte, rctx models.RequestCtx, wrapperData *wrapperData) *openrtb2.BidRequest {
	var found bool
	defer func() {
		if !found {
			rctx.MetricsEngine.RecordSignalDataStatus(wrapperData.PublisherId, wrapperData.ProfileId, models.MissingSignal)
		}
	}()

	if len(body) == 0 {
		return nil
	}

	data, dataType, _, err := jsonparser.Get(body, "imp", "[0]", "ext", "buyer_generated_request_data")
	if err != nil || dataType != jsonparser.Array {
		return nil
	}

	var signalData *openrtb2.BidRequest
	_, err = jsonparser.ArrayEach(data, func(sdkData []byte, dataType jsonparser.ValueType, offset int, err error) {
		if err != nil || dataType != jsonparser.Object {
			return
		}

		id, err := jsonparser.GetString(sdkData, "source_app", "id")
		if err != nil || (id != androidAppId && id != iOSAppId) {
			return
		}

		signal, err := jsonparser.GetString(sdkData, "data")
		if err != nil || len(signal) == 0 {
			return
		}

		// decode base64 signal
		decodedSignal, err := base64.StdEncoding.DecodeString(signal)
		if err != nil {
			return
		}

		// Signal data found
		found = true

		signalData = &openrtb2.BidRequest{}
		if err := jsoniterator.Unmarshal(decodedSignal, signalData); err != nil {
			rctx.MetricsEngine.RecordSignalDataStatus(wrapperData.PublisherId, wrapperData.ProfileId, models.InvalidSignal)
			signalData = nil
			return
		}

	})
	if err != nil {
		return nil
	}

	return signalData
}

func getWrapperData(body []byte) (*wrapperData, error) {
	if len(body) == 0 {
		return nil, errors.New("empty request body")
	}

	adunitMappingByte, datatype, _, err := jsonparser.Get(body, "imp", "[0]", "ext", "ad_unit_mapping")
	if adunitMappingByte == nil || err != nil || datatype != jsonparser.Array {
		glog.Errorf("[GoogleSDK] [Error]: failed to get ad unit mapping %v", err)
		return nil, errors.New("failed to get ad unit mapping")
	}

	var adunitMapping []map[string]interface{}
	if err := jsoniterator.Unmarshal(adunitMappingByte, &adunitMapping); err != nil {
		glog.Errorf("[GoogleSDK] [Error]: failed to unmarshal ad unit mapping %v", err)
		return nil, errors.New("failed to unmarshal ad unit mapping")
	}

	wprData := &wrapperData{}
	for _, mapping := range adunitMapping {
		keyvals, ok := mapping["keyvals"].([]interface{})
		if !ok {
			continue
		}

		for _, kv := range keyvals {
			kvMap, ok := kv.(map[string]any)
			if !ok {
				continue
			}

			key, ok := kvMap["key"].(string)
			if !ok {
				continue
			}
			value, ok := kvMap["value"].(string)
			if !ok {
				continue
			}

			switch key {
			case "publisher_id":
				wprData.PublisherId = value
			case "profile_id":
				wprData.ProfileId = value
			case "ad_unit_id":
				wprData.TagId = value
			}
		}

		// Check if all values are found
		if len(wprData.PublisherId) > 0 && len(wprData.ProfileId) > 0 && len(wprData.TagId) > 0 {
			break
		}
	}

	if len(wprData.PublisherId) == 0 && len(wprData.ProfileId) == 0 && len(wprData.TagId) == 0 {
		glog.Errorf("[GoogleSDK] [Error]: wrapper data not found in ad unit mapping")
		return nil, errors.New("wrapper data not found in ad unit mapping")
	}

	return wprData, nil
}

func ModifyRequestWithGoogleSDKParams(requestBody []byte, rctx models.RequestCtx, features feature.Features) []byte {
	if len(requestBody) == 0 {
		return requestBody
	}

	sdkRequest := &openrtb2.BidRequest{}
	if err := jsoniterator.Unmarshal(requestBody, sdkRequest); err != nil {
		return requestBody
	}

	// Get wrapper data
	wrapperData, err := getWrapperData(requestBody)
	if err != nil {
		return requestBody
	}

	// Modify request with static data
	modifyRequestWithStaticData(sdkRequest)

	//Fetch Signal data and modify request
	signalData := getSignalData(requestBody, rctx, wrapperData)
	modifyRequestWithSignalData(sdkRequest, signalData)

	// Set Publisher Id
	wrapperData.setPublisherId(sdkRequest)

	// Set profile Id at ext.prebid.bidderparams.pubmatic.wrapper.profileid
	wrapperData.setProfileID(sdkRequest)

	// Set Tag Id
	wrapperData.setTagId(sdkRequest)

	// Google SDK specific modifications
	modifyRequestWithGoogleFeature(sdkRequest, features)

	modifiedRequest, err := jsoniterator.Marshal(sdkRequest)
	if err != nil {
		return requestBody
	}

	return modifiedRequest
}

func modifyRequestWithGoogleFeature(request *openrtb2.BidRequest, features feature.Features) {
	if request == nil || len(request.Imp) == 0 || features == nil {
		return
	}

	for i := range request.Imp {
		bannerSizes := GetFlexSlotSizes(request.Imp[i].Banner, features)
		SetFlexSlotSizes(request.Imp[i].Banner, bannerSizes)
	}
}

func modifyRequestWithStaticData(request *openrtb2.BidRequest) {
	if request == nil {
		return
	}

	if len(request.Imp) > 0 {
		// Always set secure to 1
		request.Imp[0].Secure = ptrutil.ToPtr(int8(1))

		impExt := request.Imp[0].Ext
		gpid, err := jsonparser.GetString(impExt, "gpid")
		if err != nil || gpid == "" {
			// gpid not set, prefer imp.ext.dfp_ad_unit_code or imp.tagid
			if dfpAdUnitCode, err := jsonparser.GetString(impExt, "dfp_ad_unit_code"); err == nil && dfpAdUnitCode != "" {
				gpid = dfpAdUnitCode
			} else if len(request.Imp[0].TagID) > 0 {
				gpid = request.Imp[0].TagID
			}

			if gpid != "" {
				// Quote the string value
				quotedGpID := []byte(`"` + string(gpid) + `"`)
				request.Imp[0].Ext, err = jsonparser.Set(impExt, quotedGpID, "gpid")
				if err != nil {
					glog.Errorf("[GoogleSDK] [Error]: failed to set gpid %v", err)
				}
			}
		}

		// Remove banner if impression is rewarded and banner and video both are present
		if request.Imp[0].Rwdd == 1 && request.Imp[0].Banner != nil && request.Imp[0].Video != nil {
			request.Imp[0].Banner = nil
		}

		// Remove unsupported fields from banner
		if request.Imp[0].Banner != nil {
			request.Imp[0].Banner.WMin = 0
			request.Imp[0].Banner.HMin = 0
			request.Imp[0].Banner.WMax = 0
			request.Imp[0].Banner.HMax = 0
		}

		// Remove metric
		request.Imp[0].Metric = nil

		// Remove native from request
		request.Imp[0].Native = nil

		// Remove video from request
		request.Imp[0].Video = nil
	}

	// change data type of user.ext.consented_providers_settings.consented_providers from []string to []int
	if request.User != nil && request.User.Ext != nil {
		consentedProvidedBytes, dataType, _, err := jsonparser.Get(request.User.Ext, consentedProvidersSettingsListKey, consentedProvidersKey)
		if err != nil || dataType != jsonparser.Array {
			return
		}

		var consentedProviders []int
		_, err = jsonparser.ArrayEach(consentedProvidedBytes, func(provider []byte, dataType jsonparser.ValueType, offset int, err error) {
			if err != nil || dataType != jsonparser.String {
				return
			}

			providerInt, err := strconv.Atoi(string(provider))
			if err == nil {
				consentedProviders = append(consentedProviders, providerInt)
			}
		})
		if err != nil {
			// Delete consented_providers_settings.consented_providers in case of errors to avoid bad request
			request.User.Ext = jsonparser.Delete(request.User.Ext, consentedProvidersSettingsListKey, consentedProvidersKey)
			return
		}

		providersBytes, err := jsoniterator.Marshal(consentedProviders)
		if err != nil {
			// Delete consented_providers_settings.consented_providers in case of errors to avoid bad request
			request.User.Ext = jsonparser.Delete(request.User.Ext, consentedProvidersSettingsListKey, consentedProvidersKey)
			return
		}

		request.User.Ext, _ = jsonparser.Set(request.User.Ext, providersBytes, consentedProvidersSettingsListKey, consentedProvidersKey)
	}
}

func modifyRequestWithSignalData(request *openrtb2.BidRequest, signalData *openrtb2.BidRequest) {
	if request == nil || signalData == nil {
		return
	}

	modifyImpression(request, signalData.Imp)
	modifyApp(request, signalData.App)
	modifyDevice(request, signalData.Device)
	modifyRegs(request, signalData.Regs)
	modifySource(request, signalData.Source)
	modifyUser(request, signalData.User)

	// Request Ext
	request.Ext, _ = sdkutils.CopyPath(signalData.Ext, request.Ext, "wrapper", "clientconfig")
}

func modifyUser(request *openrtb2.BidRequest, signalUser *openrtb2.User) {
	if signalUser == nil {
		return
	}

	if request.User == nil {
		request.User = &openrtb2.User{}
	}

	if request.User.Ext == nil {
		request.User.Ext = []byte(`{}`)
	}

	request.User.Ext, _ = sdkutils.CopyPath(signalUser.Ext, request.User.Ext, "sessionduration")
	request.User.Ext, _ = sdkutils.CopyPath(signalUser.Ext, request.User.Ext, "impdepth")
	request.User.Ext, _ = sdkutils.CopyPath(signalUser.Ext, request.User.Ext, "consent")
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

	request.Regs.Ext, _ = sdkutils.CopyPath(signalRegs.Ext, request.Regs.Ext, "dsa", "dsarequired")
	request.Regs.Ext, _ = sdkutils.CopyPath(signalRegs.Ext, request.Regs.Ext, "dsa", "pubrender")
	request.Regs.Ext, _ = sdkutils.CopyPath(signalRegs.Ext, request.Regs.Ext, "dsa", "datatopub")
	request.Regs.Ext, _ = sdkutils.CopyPath(signalRegs.Ext, request.Regs.Ext, "gpp")
	request.Regs.Ext, _ = sdkutils.CopyPath(signalRegs.Ext, request.Regs.Ext, "gpp_sid")
	request.Regs.Ext, _ = sdkutils.CopyPath(signalRegs.Ext, request.Regs.Ext, "gdpr")
	request.Regs.Ext, _ = sdkutils.CopyPath(signalRegs.Ext, request.Regs.Ext, "us_privacy")
}

func modifyDevice(request *openrtb2.BidRequest, signalDevice *openrtb2.Device) {
	if signalDevice == nil {
		return
	}

	request.Device = sdkutils.MergeDevice(request.Device, signalDevice)

	request.Device.Ext = sdkutils.CopyIFV(signalDevice.Ext, request.Device.Ext)
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

	if signalApp.Paid != nil {
		request.App.Paid = signalApp.Paid
	}

	if len(signalApp.Keywords) > 0 {
		request.App.Keywords = signalApp.Keywords
	}

	if len(request.App.StoreURL) == 0 {
		request.App.StoreURL = signalApp.StoreURL
	}
}

func modifyBanner(requestBanner *openrtb2.Banner, signalBanner *openrtb2.Banner) {
	if requestBanner == nil || signalBanner == nil {
		return
	}

	if len(signalBanner.MIMEs) > 0 {
		requestBanner.MIMEs = signalBanner.MIMEs
	}

	if len(signalBanner.API) > 0 {
		requestBanner.API = signalBanner.API
	}
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
	requestImpExt, _ = sdkutils.CopyPath(signalImpExt, requestImpExt, "owsdk")
	return requestImpExt
}

func modifyImpression(request *openrtb2.BidRequest, signalImps []openrtb2.Imp) {
	if len(request.Imp) == 0 || len(signalImps) == 0 {
		return
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

	// Update clickbrowser
	// TODO: This is shallow copy, check if we need deep copy
	if signalImps[0].ClickBrowser != nil {
		request.Imp[0].ClickBrowser = signalImps[0].ClickBrowser
	}

	// Update banner
	modifyBanner(request.Imp[0].Banner, signalImps[0].Banner)

	// Update video (replace entire video object from signal except battr)
	var battrVideo []adcom1.CreativeAttribute
	if request.Imp[0].Video != nil && len(request.Imp[0].Video.BAttr) > 0 {
		battrVideo = make([]adcom1.CreativeAttribute, len(request.Imp[0].Video.BAttr))
		copy(battrVideo, request.Imp[0].Video.BAttr)
	}

	if signalImps[0].Video != nil {
		request.Imp[0].Video = signalImps[0].Video
		if len(battrVideo) > 0 {
			request.Imp[0].Video.BAttr = battrVideo
		}
	}

	// Update native
	request.Imp[0].Native = signalImps[0].Native
	if request.Imp[0].Native != nil {
		request.Imp[0].Native.Request = string(jsonparser.Delete([]byte(request.Imp[0].Native.Request), "privacy"))
	}

	// Update imp extension
	request.Imp[0].Ext = modifyImpExtension(request.Imp[0].Ext, signalImps[0].Ext)
}
