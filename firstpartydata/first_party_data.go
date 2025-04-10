package firstpartydata

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	jsonpatch "gopkg.in/evanphx/json-patch.v5"

	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

var (
	ErrBadRequest = errors.New("invalid request ext")
	ErrBadFPD     = errors.New("invalid first party data ext")
)

const (
	siteKey = "site"
	appKey  = "app"
	userKey = "user"
	dataKey = "data"

	userDataKey        = "userData"
	appContentDataKey  = "appContentData"
	siteContentDataKey = "siteContentData"
)

type ResolvedFirstPartyData struct {
	Site   *openrtb2.Site
	App    *openrtb2.App
	User   *openrtb2.User
	Device *openrtb2.Device
}

// ExtractGlobalFPD extracts request level FPD from the request and removes req.{site,app,user}.ext.data if exists
func ExtractGlobalFPD(req *openrtb_ext.RequestWrapper) (map[string][]byte, error) {
	fpdReqData := make(map[string][]byte, 3)

	siteExt, err := req.GetSiteExt()
	if err != nil {
		return nil, err
	}
	refreshExt := false

	if len(siteExt.GetExt()[dataKey]) > 0 {
		newSiteExt := siteExt.GetExt()
		fpdReqData[siteKey] = newSiteExt[dataKey]
		delete(newSiteExt, dataKey)
		siteExt.SetExt(newSiteExt)
		refreshExt = true
	}

	appExt, err := req.GetAppExt()
	if err != nil {
		return nil, err
	}
	if len(appExt.GetExt()[dataKey]) > 0 {
		newAppExt := appExt.GetExt()
		fpdReqData[appKey] = newAppExt[dataKey]
		delete(newAppExt, dataKey)
		appExt.SetExt(newAppExt)
		refreshExt = true
	}

	userExt, err := req.GetUserExt()
	if err != nil {
		return nil, err
	}
	if len(userExt.GetExt()[dataKey]) > 0 {
		newUserExt := userExt.GetExt()
		fpdReqData[userKey] = newUserExt[dataKey]
		delete(newUserExt, dataKey)
		userExt.SetExt(newUserExt)
		refreshExt = true
	}
	if refreshExt {
		// need to keep site/app/user ext clean in case bidder is not in global fpd bidder list
		// rebuild/resync the request in the request wrapper.
		if err := req.RebuildRequest(); err != nil {
			return nil, err
		}
	}

	return fpdReqData, nil
}

// ExtractOpenRtbGlobalFPD extracts and deletes user.data and {app/site}.content.data from request
func ExtractOpenRtbGlobalFPD(bidRequest *openrtb2.BidRequest) map[string][]openrtb2.Data {
	openRtbGlobalFPD := make(map[string][]openrtb2.Data, 3)
	if bidRequest.User != nil && len(bidRequest.User.Data) > 0 {
		openRtbGlobalFPD[userDataKey] = bidRequest.User.Data
		bidRequest.User.Data = nil
	}

	if bidRequest.Site != nil && bidRequest.Site.Content != nil && len(bidRequest.Site.Content.Data) > 0 {
		openRtbGlobalFPD[siteContentDataKey] = bidRequest.Site.Content.Data
		bidRequest.Site.Content.Data = nil
	}

	if bidRequest.App != nil && bidRequest.App.Content != nil && len(bidRequest.App.Content.Data) > 0 {
		openRtbGlobalFPD[appContentDataKey] = bidRequest.App.Content.Data
		bidRequest.App.Content.Data = nil
	}

	return openRtbGlobalFPD
}

// ResolveFPD consolidates First Party Data from different sources and returns valid FPD that will be applied to bidders later or returns errors
func ResolveFPD(bidRequest *openrtb2.BidRequest, fpdBidderConfigData map[openrtb_ext.BidderName]*openrtb_ext.ORTB2, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data, biddersWithGlobalFPD []string) (map[openrtb_ext.BidderName]*ResolvedFirstPartyData, []error) {
	var errL []error

	resolvedFpd := make(map[openrtb_ext.BidderName]*ResolvedFirstPartyData)

	allBiddersTable := make(map[string]struct{})

	if biddersWithGlobalFPD == nil {
		// add all bidders in bidder configs to receive global data and bidder specific data
		for bidderName := range fpdBidderConfigData {
			if _, present := allBiddersTable[string(bidderName)]; !present {
				allBiddersTable[string(bidderName)] = struct{}{}
			}
		}
	} else {
		// only bidders in global bidder list will receive global data and bidder specific data
		for _, bidder := range biddersWithGlobalFPD {
			bidderName := openrtb_ext.NormalizeBidderNameOrUnchanged(bidder)

			if _, present := allBiddersTable[string(bidderName)]; !present {
				allBiddersTable[string(bidderName)] = struct{}{}
			}
		}
	}

	for bidderName := range allBiddersTable {
		fpdConfig := fpdBidderConfigData[openrtb_ext.BidderName(bidderName)]

		resolvedFpdConfig := &ResolvedFirstPartyData{}

		newUser, err := resolveUser(fpdConfig, bidRequest.User, globalFPD, openRtbGlobalFPD, bidderName)
		if err != nil {
			errL = append(errL, err)
		}
		resolvedFpdConfig.User = newUser

		newApp, err := resolveApp(fpdConfig, bidRequest.App, globalFPD, openRtbGlobalFPD, bidderName)
		if err != nil {
			errL = append(errL, err)
		}
		resolvedFpdConfig.App = newApp

		newSite, err := resolveSite(fpdConfig, bidRequest.Site, globalFPD, openRtbGlobalFPD, bidderName)
		if err != nil {
			errL = append(errL, err)
		}
		resolvedFpdConfig.Site = newSite

		newDevice, err := resolveDevice(fpdConfig, bidRequest.Device)
		if err != nil {
			errL = append(errL, err)
		}
		resolvedFpdConfig.Device = newDevice

		if len(errL) == 0 {
			resolvedFpd[openrtb_ext.BidderName(bidderName)] = resolvedFpdConfig
		}
	}
	return resolvedFpd, errL
}

// resolveDevice merges the device information from the FPD (First Party Data) configuration
// with the device information provided in the bid request. It returns a new Device object
// that contains the merged data.
func resolveDevice(fpdConfig *openrtb_ext.ORTB2, bidRequestDevice *openrtb2.Device) (*openrtb2.Device, error) {
	var fpdConfigDevice json.RawMessage

	if fpdConfig != nil && fpdConfig.Device != nil {
		fpdConfigDevice = fpdConfig.Device
	}

	if bidRequestDevice == nil && fpdConfigDevice == nil {
		return nil, nil
	}

	var newDevice *openrtb2.Device
	if bidRequestDevice != nil {
		newDevice = ptrutil.Clone(bidRequestDevice)
	} else {
		newDevice = &openrtb2.Device{}
	}

	if fpdConfigDevice != nil {
		if err := jsonutil.MergeClone(newDevice, fpdConfigDevice); err != nil {
			return nil, formatMergeCloneError(err)
		}
	}

	err := validateDevice(newDevice)
	if err != nil {
		return nil, err
	}
	return newDevice, nil
}

func validateDevice(device *openrtb2.Device) error {
	if device == nil {
		return nil
	}

	// The following fields were previously uints in the OpenRTB library we use, but have
	// since been changed to ints. We decided to maintain the non-negative check.
	if device.W < 0 {
		return errors.New("request.device.w must be a positive number")
	}
	if device.H < 0 {
		return errors.New("request.device.h must be a positive number")
	}
	if device.PPI < 0 {
		return errors.New("request.device.ppi must be a positive number")
	}
	if device.Geo != nil && device.Geo.Accuracy < 0 {
		return errors.New("request.device.geo.accuracy must be a positive number")
	}
	return nil
}

func resolveUser(fpdConfig *openrtb_ext.ORTB2, bidRequestUser *openrtb2.User, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data, bidderName string) (*openrtb2.User, error) {
	var fpdConfigUser json.RawMessage

	if fpdConfig != nil && fpdConfig.User != nil {
		fpdConfigUser = fpdConfig.User
	}

	if bidRequestUser == nil && fpdConfigUser == nil {
		return nil, nil
	}

	var newUser *openrtb2.User
	if bidRequestUser != nil {
		newUser = ptrutil.Clone(bidRequestUser)
	} else {
		newUser = &openrtb2.User{}
	}

	//apply global fpd
	if len(globalFPD[userKey]) > 0 {
		extData := buildExtData(globalFPD[userKey])
		if len(newUser.Ext) > 0 {
			var err error
			newUser.Ext, err = jsonpatch.MergePatch(newUser.Ext, extData)
			if err != nil {
				return nil, formatMergePatchError(err)
			}
		} else {
			newUser.Ext = extData
		}
	}
	if openRtbGlobalFPD != nil && len(openRtbGlobalFPD[userDataKey]) > 0 {
		newUser.Data = openRtbGlobalFPD[userDataKey]
	}
	if fpdConfigUser != nil {
		if err := jsonutil.MergeClone(newUser, fpdConfigUser); err != nil {
			return nil, formatMergeCloneError(err)
		}
	}

	return newUser, nil
}

func resolveSite(fpdConfig *openrtb_ext.ORTB2, bidRequestSite *openrtb2.Site, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data, bidderName string) (*openrtb2.Site, error) {
	var fpdConfigSite json.RawMessage

	if fpdConfig != nil && fpdConfig.Site != nil {
		fpdConfigSite = fpdConfig.Site
	}

	if bidRequestSite == nil && fpdConfigSite == nil {
		return nil, nil
	}
	if bidRequestSite == nil && fpdConfigSite != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("incorrect First Party Data for bidder %s: Site object is not defined in request, but defined in FPD config", bidderName),
		}
	}

	var newSite *openrtb2.Site
	if bidRequestSite != nil {
		newSite = ptrutil.Clone(bidRequestSite)
	} else {
		newSite = &openrtb2.Site{}
	}

	//apply global fpd
	if len(globalFPD[siteKey]) > 0 {
		extData := buildExtData(globalFPD[siteKey])
		if len(newSite.Ext) > 0 {
			var err error
			newSite.Ext, err = jsonpatch.MergePatch(newSite.Ext, extData)
			if err != nil {
				return nil, formatMergePatchError(err)
			}
		} else {
			newSite.Ext = extData
		}
	}
	// apply global openRTB fpd if exists
	if len(openRtbGlobalFPD) > 0 && len(openRtbGlobalFPD[siteContentDataKey]) > 0 {
		if newSite.Content == nil {
			newSite.Content = &openrtb2.Content{}
		} else {
			contentCopy := *newSite.Content
			newSite.Content = &contentCopy
		}
		newSite.Content.Data = openRtbGlobalFPD[siteContentDataKey]
	}
	if fpdConfigSite != nil {
		if err := jsonutil.MergeClone(newSite, fpdConfigSite); err != nil {
			return nil, formatMergeCloneError(err)
		}

		// Re-Validate Site
		if newSite.ID == "" && newSite.Page == "" {
			return nil, &errortypes.BadInput{
				Message: fmt.Sprintf("incorrect First Party Data for bidder %s: Site object cannot set empty page if req.site.id is empty", bidderName),
			}
		}
	}
	return newSite, nil
}

func formatMergePatchError(err error) error {
	if errors.Is(err, jsonpatch.ErrBadJSONDoc) {
		return ErrBadRequest
	}

	if errors.Is(err, jsonpatch.ErrBadJSONPatch) {
		return ErrBadFPD
	}

	return err
}

func formatMergeCloneError(err error) error {
	if strings.Contains(err.Error(), "invalid json on existing object") {
		return ErrBadRequest
	}
	return ErrBadFPD
}

func resolveApp(fpdConfig *openrtb_ext.ORTB2, bidRequestApp *openrtb2.App, globalFPD map[string][]byte, openRtbGlobalFPD map[string][]openrtb2.Data, bidderName string) (*openrtb2.App, error) {
	var fpdConfigApp json.RawMessage

	if fpdConfig != nil {
		fpdConfigApp = fpdConfig.App
	}

	if bidRequestApp == nil && fpdConfigApp == nil {
		return nil, nil
	}

	if bidRequestApp == nil && fpdConfigApp != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("incorrect First Party Data for bidder %s: App object is not defined in request, but defined in FPD config", bidderName),
		}
	}

	var newApp *openrtb2.App
	if bidRequestApp != nil {
		newApp = ptrutil.Clone(bidRequestApp)
	} else {
		newApp = &openrtb2.App{}
	}

	//apply global fpd if exists
	if len(globalFPD[appKey]) > 0 {
		extData := buildExtData(globalFPD[appKey])
		if len(newApp.Ext) > 0 {
			var err error
			newApp.Ext, err = jsonpatch.MergePatch(newApp.Ext, extData)
			if err != nil {
				return nil, formatMergePatchError(err)
			}
		} else {
			newApp.Ext = extData
		}
	}

	// apply global openRTB fpd if exists
	if len(openRtbGlobalFPD) > 0 && len(openRtbGlobalFPD[appContentDataKey]) > 0 {
		if newApp.Content == nil {
			newApp.Content = &openrtb2.Content{}
		} else {
			contentCopy := *newApp.Content
			newApp.Content = &contentCopy
		}
		newApp.Content.Data = openRtbGlobalFPD[appContentDataKey]
	}

	if fpdConfigApp != nil {
		if err := jsonutil.MergeClone(newApp, fpdConfigApp); err != nil {
			return nil, formatMergeCloneError(err)
		}
	}

	return newApp, nil
}

func buildExtData(data []byte) []byte {
	res := make([]byte, 0, len(data)+len(`"{"data":}"`))
	res = append(res, []byte(`{"data":`)...)
	res = append(res, data...)
	res = append(res, []byte(`}`)...)
	return res
}

// ExtractBidderConfigFPD extracts bidder specific configs from req.ext.prebid.bidderconfig
func ExtractBidderConfigFPD(reqExt *openrtb_ext.RequestExt) (map[openrtb_ext.BidderName]*openrtb_ext.ORTB2, error) {
	fpd := make(map[openrtb_ext.BidderName]*openrtb_ext.ORTB2)

	reqExtPrebid := reqExt.GetPrebid()
	if reqExtPrebid != nil {
		for _, bidderConfig := range reqExtPrebid.BidderConfigs {
			for _, bidder := range bidderConfig.Bidders {
				bidderName := openrtb_ext.NormalizeBidderNameOrUnchanged(bidder)

				if _, duplicate := fpd[bidderName]; duplicate {
					return nil, &errortypes.BadInput{
						Message: fmt.Sprintf("multiple First Party Data bidder configs provided for bidder: %s", bidder),
					}
				}

				fpdBidderData := &openrtb_ext.ORTB2{}

				if bidderConfig.Config != nil && bidderConfig.Config.ORTB2 != nil {
					fpdBidderData.Site = bidderConfig.Config.ORTB2.Site
					fpdBidderData.App = bidderConfig.Config.ORTB2.App
					fpdBidderData.User = bidderConfig.Config.ORTB2.User
					fpdBidderData.Device = bidderConfig.Config.ORTB2.Device
				}

				fpd[bidderName] = fpdBidderData
			}
		}
		reqExtPrebid.BidderConfigs = nil
		reqExt.SetPrebid(reqExtPrebid)
	}
	return fpd, nil
}

// ExtractFPDForBidders extracts FPD data from request if specified
func ExtractFPDForBidders(req *openrtb_ext.RequestWrapper) (map[openrtb_ext.BidderName]*ResolvedFirstPartyData, []error) {
	reqExt, err := req.GetRequestExt()
	if err != nil {
		return nil, []error{err}
	}
	if reqExt == nil || reqExt.GetPrebid() == nil {
		return nil, nil
	}
	var biddersWithGlobalFPD []string

	extPrebid := reqExt.GetPrebid()
	if extPrebid.Data != nil {
		biddersWithGlobalFPD = extPrebid.Data.Bidders
		extPrebid.Data.Bidders = nil
		reqExt.SetPrebid(extPrebid)
	}

	fbdBidderConfigData, err := ExtractBidderConfigFPD(reqExt)
	if err != nil {
		return nil, []error{err}
	}

	var globalFpd map[string][]byte
	var openRtbGlobalFPD map[string][]openrtb2.Data

	if biddersWithGlobalFPD != nil {
		//global fpd data should not be extracted and removed from request if global bidder list is nil.
		//Bidders that don't have any fpd config should receive request data as is
		globalFpd, err = ExtractGlobalFPD(req)
		if err != nil {
			return nil, []error{err}
		}
		openRtbGlobalFPD = ExtractOpenRtbGlobalFPD(req.BidRequest)
	}

	return ResolveFPD(req.BidRequest, fbdBidderConfigData, globalFpd, openRtbGlobalFPD, biddersWithGlobalFPD)
}
