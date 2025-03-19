package openrtb_ext

import (
	"encoding/json"
	"errors"
	"maps"
	"slices"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

// RequestWrapper wraps the OpenRTB request to provide a storage location for unmarshalled ext fields, so they
// will not need to be unmarshalled multiple times.
//
// To start with, the wrapper can be created for a request 'req' via:
// reqWrapper := openrtb_ext.RequestWrapper{BidRequest: req}
//
// In order to access an object's ext field, fetch it via:
// userExt, err := reqWrapper.GetUserExt()
// or other Get method as appropriate.
//
// To read or write values, use the Ext objects Get and Set methods. If you need to write to a field that has its own Set
// method, use that to set the value rather than using SetExt() with that change done in the map; when rewritting the
// ext JSON the code will overwrite the the values in the map with the values stored in the seperate fields.
//
// userPrebid := userExt.GetPrebid()
// userExt.SetConsent(consentString)
//
// The GetExt() and SetExt() should only be used to access fields that have not already been resolved in the object.
// Using SetExt() at all is a strong hint that the ext object should be extended to support the new fields being set
// in the map.
//
// NOTE: The RequestWrapper methods (particularly the ones calling (un)Marshal are not thread safe)
type RequestWrapper struct {
	*openrtb2.BidRequest
	impWrappers         []*ImpWrapper
	impWrappersAccessed bool
	userExt             *UserExt
	deviceExt           *DeviceExt
	requestExt          *RequestExt
	appExt              *AppExt
	regExt              *RegExt
	siteExt             *SiteExt
	doohExt             *DOOHExt
	sourceExt           *SourceExt
}

const (
	jsonEmptyObjectLength               = 2
	consentedProvidersSettingsStringKey = "ConsentedProvidersSettings"
	consentedProvidersSettingsListKey   = "consented_providers_settings"
	consentKey                          = "consent"
	ampKey                              = "amp"
	dsaKey                              = "dsa"
	eidsKey                             = "eids"
	gdprKey                             = "gdpr"
	prebidKey                           = "prebid"
	dataKey                             = "data"
	schainKey                           = "schain"
	us_privacyKey                       = "us_privacy"
	cdepKey                             = "cdep"
	gpcKey                              = "gpc"
)

// LenImp returns the number of impressions without causing the creation of ImpWrapper objects.
func (rw *RequestWrapper) LenImp() int {
	if rw.impWrappersAccessed {
		return len(rw.impWrappers)
	}

	return len(rw.Imp)
}

func (rw *RequestWrapper) GetImp() []*ImpWrapper {
	if rw.impWrappersAccessed {
		return rw.impWrappers
	}

	// There is minimal difference between nil and empty arrays in Go, but it matters
	// for json encoding. In practice there will always be at least one impression,
	// so this is an optimization for tests with (appropriately) incomplete requests.
	if rw.Imp != nil {
		rw.impWrappers = make([]*ImpWrapper, len(rw.Imp))
		for i := range rw.Imp {
			rw.impWrappers[i] = &ImpWrapper{Imp: &rw.Imp[i]}
		}
	}

	rw.impWrappersAccessed = true

	return rw.impWrappers
}

func (rw *RequestWrapper) SetImp(imps []*ImpWrapper) {
	rw.impWrappers = imps
	imparr := make([]openrtb2.Imp, len(imps))
	for i, iw := range imps {
		imparr[i] = *iw.Imp
		iw.Imp = &imparr[i]
	}
	rw.Imp = imparr
	rw.impWrappersAccessed = true
}

func (rw *RequestWrapper) GetUserExt() (*UserExt, error) {
	if rw.userExt != nil {
		return rw.userExt, nil
	}
	rw.userExt = &UserExt{}
	if rw.BidRequest == nil || rw.User == nil || rw.User.Ext == nil {
		return rw.userExt, rw.userExt.unmarshal(json.RawMessage{})
	}

	return rw.userExt, rw.userExt.unmarshal(rw.User.Ext)
}

func (rw *RequestWrapper) GetDeviceExt() (*DeviceExt, error) {
	if rw.deviceExt != nil {
		return rw.deviceExt, nil
	}
	rw.deviceExt = &DeviceExt{}
	if rw.BidRequest == nil || rw.Device == nil || rw.Device.Ext == nil {
		return rw.deviceExt, rw.deviceExt.unmarshal(json.RawMessage{})
	}
	return rw.deviceExt, rw.deviceExt.unmarshal(rw.Device.Ext)
}

func (rw *RequestWrapper) GetRequestExt() (*RequestExt, error) {
	if rw.requestExt != nil {
		return rw.requestExt, nil
	}
	rw.requestExt = &RequestExt{}
	if rw.BidRequest == nil || rw.Ext == nil {
		return rw.requestExt, rw.requestExt.unmarshal(json.RawMessage{})
	}
	return rw.requestExt, rw.requestExt.unmarshal(rw.Ext)
}

func (rw *RequestWrapper) GetAppExt() (*AppExt, error) {
	if rw.appExt != nil {
		return rw.appExt, nil
	}
	rw.appExt = &AppExt{}
	if rw.BidRequest == nil || rw.App == nil || rw.App.Ext == nil {
		return rw.appExt, rw.appExt.unmarshal(json.RawMessage{})
	}
	return rw.appExt, rw.appExt.unmarshal(rw.App.Ext)
}

func (rw *RequestWrapper) GetRegExt() (*RegExt, error) {
	if rw.regExt != nil {
		return rw.regExt, nil
	}
	rw.regExt = &RegExt{}
	if rw.BidRequest == nil || rw.Regs == nil || rw.Regs.Ext == nil {
		return rw.regExt, rw.regExt.unmarshal(json.RawMessage{})
	}
	return rw.regExt, rw.regExt.unmarshal(rw.Regs.Ext)
}

func (rw *RequestWrapper) GetSiteExt() (*SiteExt, error) {
	if rw.siteExt != nil {
		return rw.siteExt, nil
	}
	rw.siteExt = &SiteExt{}
	if rw.BidRequest == nil || rw.Site == nil || rw.Site.Ext == nil {
		return rw.siteExt, rw.siteExt.unmarshal(json.RawMessage{})
	}
	return rw.siteExt, rw.siteExt.unmarshal(rw.Site.Ext)
}

func (rw *RequestWrapper) GetDOOHExt() (*DOOHExt, error) {
	if rw.doohExt != nil {
		return rw.doohExt, nil
	}
	rw.doohExt = &DOOHExt{}
	if rw.BidRequest == nil || rw.DOOH == nil || rw.DOOH.Ext == nil {
		return rw.doohExt, rw.doohExt.unmarshal(json.RawMessage{})
	}
	return rw.doohExt, rw.doohExt.unmarshal(rw.DOOH.Ext)
}

func (rw *RequestWrapper) GetSourceExt() (*SourceExt, error) {
	if rw.sourceExt != nil {
		return rw.sourceExt, nil
	}
	rw.sourceExt = &SourceExt{}
	if rw.BidRequest == nil || rw.Source == nil || rw.Source.Ext == nil {
		return rw.sourceExt, rw.sourceExt.unmarshal(json.RawMessage{})
	}
	return rw.sourceExt, rw.sourceExt.unmarshal(rw.Source.Ext)
}

func (rw *RequestWrapper) RebuildRequest() error {
	if rw.BidRequest == nil {
		return errors.New("Requestwrapper RebuildRequest called on a nil BidRequest")
	}

	if err := rw.rebuildImp(); err != nil {
		return err
	}
	if err := rw.rebuildUserExt(); err != nil {
		return err
	}
	if err := rw.rebuildDeviceExt(); err != nil {
		return err
	}
	if err := rw.rebuildRequestExt(); err != nil {
		return err
	}
	if err := rw.rebuildAppExt(); err != nil {
		return err
	}
	if err := rw.rebuildRegExt(); err != nil {
		return err
	}
	if err := rw.rebuildSiteExt(); err != nil {
		return err
	}
	if err := rw.rebuildDOOHExt(); err != nil {
		return err
	}
	if err := rw.rebuildSourceExt(); err != nil {
		return err
	}

	return nil
}

func (rw *RequestWrapper) rebuildImp() error {
	if !rw.impWrappersAccessed {
		return nil
	}

	if rw.impWrappers == nil {
		rw.Imp = nil
		return nil
	}

	rw.Imp = make([]openrtb2.Imp, len(rw.impWrappers))
	for i := range rw.impWrappers {
		if err := rw.impWrappers[i].RebuildImp(); err != nil {
			return err
		}
		rw.Imp[i] = *rw.impWrappers[i].Imp
		rw.impWrappers[i].Imp = &rw.Imp[i]
	}

	return nil
}

func (rw *RequestWrapper) rebuildUserExt() error {
	if rw.userExt == nil || !rw.userExt.Dirty() {
		return nil
	}

	userJson, err := rw.userExt.marshal()
	if err != nil {
		return err
	}

	if userJson != nil && rw.User == nil {
		rw.User = &openrtb2.User{Ext: userJson}
	} else if rw.User != nil {
		rw.User.Ext = userJson
	}

	return nil
}

func (rw *RequestWrapper) rebuildDeviceExt() error {
	if rw.deviceExt == nil || !rw.deviceExt.Dirty() {
		return nil
	}

	deviceJson, err := rw.deviceExt.marshal()
	if err != nil {
		return err
	}

	if deviceJson != nil && rw.Device == nil {
		rw.Device = &openrtb2.Device{Ext: deviceJson}
	} else if rw.Device != nil {
		rw.Device.Ext = deviceJson
	}

	return nil
}

func (rw *RequestWrapper) rebuildRequestExt() error {
	if rw.requestExt == nil || !rw.requestExt.Dirty() {
		return nil
	}

	requestJson, err := rw.requestExt.marshal()
	if err != nil {
		return err
	}

	rw.Ext = requestJson

	return nil
}

func (rw *RequestWrapper) rebuildAppExt() error {
	if rw.appExt == nil || !rw.appExt.Dirty() {
		return nil
	}

	appJson, err := rw.appExt.marshal()
	if err != nil {
		return err
	}

	if appJson != nil && rw.App == nil {
		rw.App = &openrtb2.App{Ext: appJson}
	} else if rw.App != nil {
		rw.App.Ext = appJson
	}

	return nil
}

func (rw *RequestWrapper) rebuildDOOHExt() error {
	if rw.doohExt == nil || !rw.doohExt.Dirty() {
		return nil
	}

	doohJson, err := rw.doohExt.marshal()
	if err != nil {
		return err
	}

	if doohJson != nil && rw.DOOH == nil {
		rw.DOOH = &openrtb2.DOOH{Ext: doohJson}
	} else if rw.DOOH != nil {
		rw.DOOH.Ext = doohJson
	}

	return nil
}

func (rw *RequestWrapper) rebuildRegExt() error {
	if rw.regExt == nil || !rw.regExt.Dirty() {
		return nil
	}

	regsJson, err := rw.regExt.marshal()
	if err != nil {
		return err
	}

	if regsJson != nil && rw.Regs == nil {
		rw.Regs = &openrtb2.Regs{Ext: regsJson}
	} else if rw.Regs != nil {
		rw.Regs.Ext = regsJson
	}

	return nil
}

func (rw *RequestWrapper) rebuildSiteExt() error {
	if rw.siteExt == nil || !rw.siteExt.Dirty() {
		return nil
	}

	siteJson, err := rw.siteExt.marshal()
	if err != nil {
		return err
	}

	if siteJson != nil && rw.Site == nil {
		rw.Site = &openrtb2.Site{Ext: siteJson}
	} else if rw.Site != nil {
		rw.Site.Ext = siteJson
	}

	return nil
}

func (rw *RequestWrapper) rebuildSourceExt() error {
	if rw.sourceExt == nil || !rw.sourceExt.Dirty() {
		return nil
	}

	sourceJson, err := rw.sourceExt.marshal()
	if err != nil {
		return err
	}

	if sourceJson != nil && rw.Source == nil {
		rw.Source = &openrtb2.Source{Ext: sourceJson}
	} else if rw.Source != nil {
		rw.Source.Ext = sourceJson
	}

	return nil
}

// Clone clones the request wrapper exts and the imp wrappers
// the cloned imp wrappers are pointing to the bid request imps
func (rw *RequestWrapper) Clone() *RequestWrapper {
	if rw == nil {
		return nil
	}
	clone := *rw
	newImpWrappers := make([]*ImpWrapper, len(rw.impWrappers))
	for i, iw := range rw.impWrappers {
		newImpWrappers[i] = iw.Clone()
	}
	clone.impWrappers = newImpWrappers
	clone.userExt = rw.userExt.Clone()
	clone.deviceExt = rw.deviceExt.Clone()
	clone.requestExt = rw.requestExt.Clone()
	clone.appExt = rw.appExt.Clone()
	clone.regExt = rw.regExt.Clone()
	clone.siteExt = rw.siteExt.Clone()
	clone.doohExt = rw.doohExt.Clone()
	clone.sourceExt = rw.sourceExt.Clone()

	return &clone
}

func (rw *RequestWrapper) CloneAndClearImpWrappers() *RequestWrapper {
	if rw == nil {
		return nil
	}
	rw.impWrappersAccessed = false

	clone := *rw
	clone.impWrappers = nil
	clone.userExt = rw.userExt.Clone()
	clone.deviceExt = rw.deviceExt.Clone()
	clone.requestExt = rw.requestExt.Clone()
	clone.appExt = rw.appExt.Clone()
	clone.regExt = rw.regExt.Clone()
	clone.siteExt = rw.siteExt.Clone()
	clone.doohExt = rw.doohExt.Clone()
	clone.sourceExt = rw.sourceExt.Clone()

	return &clone
}

// ---------------------------------------------------------------
// UserExt provides an interface for request.user.ext
// ---------------------------------------------------------------

type UserExt struct {
	ext                                map[string]json.RawMessage
	extDirty                           bool
	consent                            *string
	consentDirty                       bool
	prebid                             *ExtUserPrebid
	prebidDirty                        bool
	eids                               *[]openrtb2.EID
	eidsDirty                          bool
	consentedProvidersSettingsIn       *ConsentedProvidersSettingsIn
	consentedProvidersSettingsInDirty  bool
	consentedProvidersSettingsOut      *ConsentedProvidersSettingsOut
	consentedProvidersSettingsOutDirty bool
}

func (ue *UserExt) unmarshal(extJson json.RawMessage) error {
	if len(ue.ext) != 0 || ue.Dirty() {
		return nil
	}

	ue.ext = make(map[string]json.RawMessage)

	if len(extJson) == 0 {
		return nil
	}

	if err := jsonutil.Unmarshal(extJson, &ue.ext); err != nil {
		return err
	}

	consentJson, hasConsent := ue.ext[consentKey]
	if hasConsent && consentJson != nil {
		if err := jsonutil.Unmarshal(consentJson, &ue.consent); err != nil {
			return err
		}
	}

	prebidJson, hasPrebid := ue.ext[prebidKey]
	if hasPrebid {
		ue.prebid = &ExtUserPrebid{}
	}
	if prebidJson != nil {
		if err := jsonutil.Unmarshal(prebidJson, ue.prebid); err != nil {
			return err
		}
	}

	eidsJson, hasEids := ue.ext[eidsKey]
	if hasEids {
		ue.eids = &[]openrtb2.EID{}
	}
	if eidsJson != nil {
		if err := jsonutil.Unmarshal(eidsJson, ue.eids); err != nil {
			return err
		}
	}

	consentedProviderSettingsInJson, hasCPSettingsIn := ue.ext[consentedProvidersSettingsStringKey]
	if hasCPSettingsIn {
		ue.consentedProvidersSettingsIn = &ConsentedProvidersSettingsIn{}
	}
	if consentedProviderSettingsInJson != nil {
		if err := jsonutil.Unmarshal(consentedProviderSettingsInJson, ue.consentedProvidersSettingsIn); err != nil {
			return err
		}
	}

	consentedProviderSettingsOutJson, hasCPSettingsOut := ue.ext[consentedProvidersSettingsListKey]
	if hasCPSettingsOut {
		ue.consentedProvidersSettingsOut = &ConsentedProvidersSettingsOut{}
	}
	if consentedProviderSettingsOutJson != nil {
		if err := jsonutil.Unmarshal(consentedProviderSettingsOutJson, ue.consentedProvidersSettingsOut); err != nil {
			return err
		}
	}

	return nil
}

func (ue *UserExt) marshal() (json.RawMessage, error) {
	if ue.consentDirty {
		if ue.consent != nil && len(*ue.consent) > 0 {
			consentJson, err := jsonutil.Marshal(ue.consent)
			if err != nil {
				return nil, err
			}
			ue.ext[consentKey] = json.RawMessage(consentJson)
		} else {
			delete(ue.ext, consentKey)
		}
		ue.consentDirty = false
	}

	if ue.prebidDirty {
		if ue.prebid != nil {
			prebidJson, err := jsonutil.Marshal(ue.prebid)
			if err != nil {
				return nil, err
			}
			if len(prebidJson) > jsonEmptyObjectLength {
				ue.ext[prebidKey] = json.RawMessage(prebidJson)
			} else {
				delete(ue.ext, prebidKey)
			}
		} else {
			delete(ue.ext, prebidKey)
		}
		ue.prebidDirty = false
	}

	if ue.consentedProvidersSettingsInDirty {
		if ue.consentedProvidersSettingsIn != nil {
			cpSettingsJson, err := jsonutil.Marshal(ue.consentedProvidersSettingsIn)
			if err != nil {
				return nil, err
			}
			if len(cpSettingsJson) > jsonEmptyObjectLength {
				ue.ext[consentedProvidersSettingsStringKey] = json.RawMessage(cpSettingsJson)
			} else {
				delete(ue.ext, consentedProvidersSettingsStringKey)
			}
		} else {
			delete(ue.ext, consentedProvidersSettingsStringKey)
		}
		ue.consentedProvidersSettingsInDirty = false
	}

	if ue.consentedProvidersSettingsOutDirty {
		if ue.consentedProvidersSettingsOut != nil {
			cpSettingsJson, err := jsonutil.Marshal(ue.consentedProvidersSettingsOut)
			if err != nil {
				return nil, err
			}
			if len(cpSettingsJson) > jsonEmptyObjectLength {
				ue.ext[consentedProvidersSettingsListKey] = json.RawMessage(cpSettingsJson)
			} else {
				delete(ue.ext, consentedProvidersSettingsListKey)
			}
		} else {
			delete(ue.ext, consentedProvidersSettingsListKey)
		}
		ue.consentedProvidersSettingsOutDirty = false
	}

	if ue.eidsDirty {
		if ue.eids != nil && len(*ue.eids) > 0 {
			eidsJson, err := jsonutil.Marshal(ue.eids)
			if err != nil {
				return nil, err
			}
			ue.ext[eidsKey] = json.RawMessage(eidsJson)
		} else {
			delete(ue.ext, eidsKey)
		}
		ue.eidsDirty = false
	}

	ue.extDirty = false
	if len(ue.ext) == 0 {
		return nil, nil
	}
	return jsonutil.Marshal(ue.ext)
}

func (ue *UserExt) Dirty() bool {
	return ue.extDirty || ue.eidsDirty || ue.prebidDirty || ue.consentDirty || ue.consentedProvidersSettingsInDirty || ue.consentedProvidersSettingsOutDirty
}

func (ue *UserExt) GetExt() map[string]json.RawMessage {
	ext := make(map[string]json.RawMessage)
	for k, v := range ue.ext {
		ext[k] = v
	}
	return ext
}

func (ue *UserExt) SetExt(ext map[string]json.RawMessage) {
	ue.ext = ext
	ue.extDirty = true
}

func (ue *UserExt) GetConsent() *string {
	if ue.consent == nil {
		return nil
	}
	consent := *ue.consent
	return &consent
}

func (ue *UserExt) SetConsent(consent *string) {
	ue.consent = consent
	ue.consentDirty = true
}

// GetConsentedProvidersSettingsIn() returns a reference to a copy of ConsentedProvidersSettingsIn, a struct that
// has a string field formatted as a Google's Additional Consent string
func (ue *UserExt) GetConsentedProvidersSettingsIn() *ConsentedProvidersSettingsIn {
	if ue.consentedProvidersSettingsIn == nil {
		return nil
	}
	consentedProvidersSettingsIn := *ue.consentedProvidersSettingsIn
	return &consentedProvidersSettingsIn
}

// SetConsentedProvidersSettingsIn() sets ConsentedProvidersSettingsIn, a struct that
// has a string field formatted as a Google's Additional Consent string
func (ue *UserExt) SetConsentedProvidersSettingsIn(cpSettings *ConsentedProvidersSettingsIn) {
	ue.consentedProvidersSettingsIn = cpSettings
	ue.consentedProvidersSettingsInDirty = true
}

// GetConsentedProvidersSettingsOut() returns a reference to a copy of ConsentedProvidersSettingsOut, a struct that
// has an int array field listing Google's Additional Consent string elements
func (ue *UserExt) GetConsentedProvidersSettingsOut() *ConsentedProvidersSettingsOut {
	if ue.consentedProvidersSettingsOut == nil {
		return nil
	}
	consentedProvidersSettingsOut := *ue.consentedProvidersSettingsOut
	return &consentedProvidersSettingsOut
}

// SetConsentedProvidersSettingsIn() sets ConsentedProvidersSettingsOut, a struct that
// has an int array field listing Google's Additional Consent string elements. This
// function overrides an existing ConsentedProvidersSettingsOut object, if any
func (ue *UserExt) SetConsentedProvidersSettingsOut(cpSettings *ConsentedProvidersSettingsOut) {
	if cpSettings == nil {
		return
	}

	ue.consentedProvidersSettingsOut = cpSettings
	ue.consentedProvidersSettingsOutDirty = true
}

func (ue *UserExt) GetPrebid() *ExtUserPrebid {
	if ue.prebid == nil {
		return nil
	}
	prebid := *ue.prebid
	return &prebid
}

func (ue *UserExt) SetPrebid(prebid *ExtUserPrebid) {
	ue.prebid = prebid
	ue.prebidDirty = true
}

func (ue *UserExt) GetEid() *[]openrtb2.EID {
	if ue.eids == nil {
		return nil
	}
	eids := *ue.eids
	return &eids
}

func (ue *UserExt) SetEid(eid *[]openrtb2.EID) {
	ue.eids = eid
	ue.eidsDirty = true
}

func (ue *UserExt) Clone() *UserExt {
	if ue == nil {
		return nil
	}
	clone := *ue
	clone.ext = maps.Clone(ue.ext)

	if ue.consent != nil {
		clonedConsent := *ue.consent
		clone.consent = &clonedConsent
	}

	if ue.prebid != nil {
		clone.prebid = &ExtUserPrebid{}
		clone.prebid.BuyerUIDs = maps.Clone(ue.prebid.BuyerUIDs)
	}

	if ue.eids != nil {
		clonedEids := make([]openrtb2.EID, len(*ue.eids))
		for i, eid := range *ue.eids {
			newEid := eid
			newEid.UIDs = slices.Clone(eid.UIDs)
			clonedEids[i] = newEid
		}
		clone.eids = &clonedEids
	}

	if ue.consentedProvidersSettingsIn != nil {
		clone.consentedProvidersSettingsIn = &ConsentedProvidersSettingsIn{ConsentedProvidersString: ue.consentedProvidersSettingsIn.ConsentedProvidersString}
	}
	if ue.consentedProvidersSettingsOut != nil {
		clone.consentedProvidersSettingsOut = &ConsentedProvidersSettingsOut{ConsentedProvidersList: slices.Clone(ue.consentedProvidersSettingsOut.ConsentedProvidersList)}
	}

	return &clone
}

// ---------------------------------------------------------------
// RequestExt provides an interface for request.ext
// ---------------------------------------------------------------

type RequestExt struct {
	ext         map[string]json.RawMessage
	extDirty    bool
	prebid      *ExtRequestPrebid
	prebidDirty bool
	schain      *openrtb2.SupplyChain // ORTB 2.4 location
	schainDirty bool
}

func (re *RequestExt) unmarshal(extJson json.RawMessage) error {
	if len(re.ext) != 0 || re.Dirty() {
		return nil
	}

	re.ext = make(map[string]json.RawMessage)

	if len(extJson) == 0 {
		return nil
	}

	if err := jsonutil.Unmarshal(extJson, &re.ext); err != nil {
		return err
	}

	prebidJson, hasPrebid := re.ext[prebidKey]
	if hasPrebid {
		re.prebid = &ExtRequestPrebid{}
	}
	if prebidJson != nil {
		if err := jsonutil.Unmarshal(prebidJson, re.prebid); err != nil {
			return err
		}
	}

	schainJson, hasSChain := re.ext[schainKey]
	if hasSChain {
		re.schain = &openrtb2.SupplyChain{}
	}
	if schainJson != nil {
		if err := jsonutil.Unmarshal(schainJson, re.schain); err != nil {
			return err
		}
	}

	return nil
}

func (re *RequestExt) marshal() (json.RawMessage, error) {
	if re.prebidDirty {
		if re.prebid != nil {
			prebidJson, err := jsonutil.Marshal(re.prebid)
			if err != nil {
				return nil, err
			}
			if len(prebidJson) > jsonEmptyObjectLength {
				re.ext[prebidKey] = json.RawMessage(prebidJson)
			} else {
				delete(re.ext, prebidKey)
			}
		} else {
			delete(re.ext, prebidKey)
		}
		re.prebidDirty = false
	}

	if re.schainDirty {
		if re.schain != nil {
			schainJson, err := jsonutil.Marshal(re.schain)
			if err != nil {
				return nil, err
			}
			if len(schainJson) > jsonEmptyObjectLength {
				re.ext[schainKey] = json.RawMessage(schainJson)
			} else {
				delete(re.ext, schainKey)
			}
		} else {
			delete(re.ext, schainKey)
		}
		re.schainDirty = false
	}

	re.extDirty = false
	if len(re.ext) == 0 {
		return nil, nil
	}
	return jsonutil.Marshal(re.ext)
}

func (re *RequestExt) Dirty() bool {
	return re.extDirty || re.prebidDirty || re.schainDirty
}

func (re *RequestExt) GetExt() map[string]json.RawMessage {
	ext := make(map[string]json.RawMessage)
	for k, v := range re.ext {
		ext[k] = v
	}
	return ext
}

func (re *RequestExt) SetExt(ext map[string]json.RawMessage) {
	re.ext = ext
	re.extDirty = true
}

func (re *RequestExt) GetPrebid() *ExtRequestPrebid {
	if re == nil || re.prebid == nil {
		return nil
	}
	prebid := *re.prebid
	return &prebid
}

func (re *RequestExt) SetPrebid(prebid *ExtRequestPrebid) {
	re.prebid = prebid
	re.prebidDirty = true
}

// These schain methods on the request.ext are only for ORTB 2.4 backwards compatibility and
// should not be used for any other purposes. To access ORTB 2.5 schains, see source.ext.schain
// or request.ext.prebid.schains.
func (re *RequestExt) GetSChain() *openrtb2.SupplyChain {
	if re.schain == nil {
		return nil
	}
	schain := *re.schain
	return &schain
}

func (re *RequestExt) SetSChain(schain *openrtb2.SupplyChain) {
	re.schain = schain
	re.schainDirty = true
}

func (re *RequestExt) Clone() *RequestExt {
	if re == nil {
		return nil
	}

	clone := *re
	clone.ext = maps.Clone(re.ext)

	if re.prebid != nil {
		clone.prebid = re.prebid.Clone()
	}

	clone.schain = cloneSupplyChain(re.schain)

	return &clone
}

// ---------------------------------------------------------------
// DeviceExt provides an interface for request.device.ext
// ---------------------------------------------------------------
// NOTE: openrtb_ext/device.go:ParseDeviceExtATTS() uses ext.atts, as read only, via jsonparser, only for IOS.
// Doesn't seem like we will see any performance savings by parsing atts at this point, and as it is read only,
// we don't need to worry about write conflicts. Note here in case additional uses of atts evolve as things progress.
// ---------------------------------------------------------------

type DeviceExt struct {
	ext         map[string]json.RawMessage
	extDirty    bool
	prebid      *ExtDevicePrebid
	prebidDirty bool
	cdep        string
	cdepDirty   bool
}

func (de *DeviceExt) unmarshal(extJson json.RawMessage) error {
	if len(de.ext) != 0 || de.Dirty() {
		return nil
	}

	de.ext = make(map[string]json.RawMessage)

	if len(extJson) == 0 {
		return nil
	}

	if err := jsonutil.Unmarshal(extJson, &de.ext); err != nil {
		return err
	}

	prebidJson, hasPrebid := de.ext[prebidKey]
	if hasPrebid {
		de.prebid = &ExtDevicePrebid{}
	}
	if prebidJson != nil {
		if err := jsonutil.Unmarshal(prebidJson, de.prebid); err != nil {
			return err
		}
	}

	cdepJson, hasCDep := de.ext[cdepKey]
	if hasCDep && cdepJson != nil {
		if err := jsonutil.Unmarshal(cdepJson, &de.cdep); err != nil {
			return err
		}
	}

	return nil
}

func (de *DeviceExt) marshal() (json.RawMessage, error) {
	if de.prebidDirty {
		if de.prebid != nil {
			prebidJson, err := jsonutil.Marshal(de.prebid)
			if err != nil {
				return nil, err
			}
			if len(prebidJson) > jsonEmptyObjectLength {
				de.ext[prebidKey] = json.RawMessage(prebidJson)
			} else {
				delete(de.ext, prebidKey)
			}
		} else {
			delete(de.ext, prebidKey)
		}
		de.prebidDirty = false
	}

	if de.cdepDirty {
		if len(de.cdep) > 0 {
			rawjson, err := jsonutil.Marshal(de.cdep)
			if err != nil {
				return nil, err
			}
			de.ext[cdepKey] = rawjson
		} else {
			delete(de.ext, cdepKey)
		}
		de.cdepDirty = false
	}

	de.extDirty = false
	if len(de.ext) == 0 {
		return nil, nil
	}
	return jsonutil.Marshal(de.ext)
}

func (de *DeviceExt) Dirty() bool {
	return de.extDirty || de.prebidDirty || de.cdepDirty
}

func (de *DeviceExt) GetExt() map[string]json.RawMessage {
	ext := make(map[string]json.RawMessage)
	for k, v := range de.ext {
		ext[k] = v
	}
	return ext
}

func (de *DeviceExt) SetExt(ext map[string]json.RawMessage) {
	de.ext = ext
	de.extDirty = true
}

func (de *DeviceExt) GetPrebid() *ExtDevicePrebid {
	if de.prebid == nil {
		return nil
	}
	prebid := *de.prebid
	return &prebid
}

func (de *DeviceExt) SetPrebid(prebid *ExtDevicePrebid) {
	de.prebid = prebid
	de.prebidDirty = true
}

func (de *DeviceExt) GetCDep() string {
	return de.cdep
}

func (de *DeviceExt) SetCDep(cdep string) {
	de.cdep = cdep
	de.cdepDirty = true
}

func (de *DeviceExt) Clone() *DeviceExt {
	if de == nil {
		return nil
	}

	clone := *de
	clone.ext = maps.Clone(de.ext)

	if de.prebid != nil {
		clonedPrebid := *de.prebid
		if clonedPrebid.Interstitial != nil {
			clonedInterstitial := *de.prebid.Interstitial
			clonedPrebid.Interstitial = &clonedInterstitial
		}
		clone.prebid = &clonedPrebid
	}

	return &clone
}

// ---------------------------------------------------------------
// AppExt provides an interface for request.app.ext
// ---------------------------------------------------------------

type AppExt struct {
	ext         map[string]json.RawMessage
	extDirty    bool
	prebid      *ExtAppPrebid
	prebidDirty bool
}

func (ae *AppExt) unmarshal(extJson json.RawMessage) error {
	if len(ae.ext) != 0 || ae.Dirty() {
		return nil
	}

	ae.ext = make(map[string]json.RawMessage)

	if len(extJson) == 0 {
		return nil
	}

	if err := jsonutil.Unmarshal(extJson, &ae.ext); err != nil {
		return err
	}

	prebidJson, hasPrebid := ae.ext[prebidKey]
	if hasPrebid {
		ae.prebid = &ExtAppPrebid{}
	}
	if prebidJson != nil {
		if err := jsonutil.Unmarshal(prebidJson, ae.prebid); err != nil {
			return err
		}
	}

	return nil
}

func (ae *AppExt) marshal() (json.RawMessage, error) {
	if ae.prebidDirty {
		if ae.prebid != nil {
			prebidJson, err := jsonutil.Marshal(ae.prebid)
			if err != nil {
				return nil, err
			}
			if len(prebidJson) > jsonEmptyObjectLength {
				ae.ext[prebidKey] = json.RawMessage(prebidJson)
			} else {
				delete(ae.ext, prebidKey)
			}
		} else {
			delete(ae.ext, prebidKey)
		}
		ae.prebidDirty = false
	}

	ae.extDirty = false
	if len(ae.ext) == 0 {
		return nil, nil
	}
	return jsonutil.Marshal(ae.ext)
}

func (ae *AppExt) Dirty() bool {
	return ae.extDirty || ae.prebidDirty
}

func (ae *AppExt) GetExt() map[string]json.RawMessage {
	ext := make(map[string]json.RawMessage)
	for k, v := range ae.ext {
		ext[k] = v
	}
	return ext
}

func (ae *AppExt) SetExt(ext map[string]json.RawMessage) {
	ae.ext = ext
	ae.extDirty = true
}

func (ae *AppExt) GetPrebid() *ExtAppPrebid {
	if ae.prebid == nil {
		return nil
	}
	prebid := *ae.prebid
	return &prebid
}

func (ae *AppExt) SetPrebid(prebid *ExtAppPrebid) {
	ae.prebid = prebid
	ae.prebidDirty = true
}

func (ae *AppExt) Clone() *AppExt {
	if ae == nil {
		return nil
	}

	clone := *ae
	clone.ext = maps.Clone(ae.ext)

	clone.prebid = ptrutil.Clone(ae.prebid)

	return &clone
}

// ---------------------------------------------------------------
// DOOHExt provides an interface for request.dooh.ext
// This is currently a placeholder for consistency with others - no useful attributes and getters/setters exist yet
// ---------------------------------------------------------------

type DOOHExt struct {
	ext      map[string]json.RawMessage
	extDirty bool
}

func (de *DOOHExt) unmarshal(extJson json.RawMessage) error {
	if len(de.ext) != 0 || de.Dirty() {
		return nil
	}

	de.ext = make(map[string]json.RawMessage)

	if len(extJson) == 0 {
		return nil
	}

	if err := jsonutil.Unmarshal(extJson, &de.ext); err != nil {
		return err
	}

	return nil
}

func (de *DOOHExt) marshal() (json.RawMessage, error) {
	de.extDirty = false
	if len(de.ext) == 0 {
		return nil, nil
	}
	return jsonutil.Marshal(de.ext)
}

func (de *DOOHExt) Dirty() bool {
	return de.extDirty
}

func (de *DOOHExt) GetExt() map[string]json.RawMessage {
	ext := make(map[string]json.RawMessage)
	for k, v := range de.ext {
		ext[k] = v
	}
	return ext
}

func (de *DOOHExt) SetExt(ext map[string]json.RawMessage) {
	de.ext = ext
	de.extDirty = true
}

func (de *DOOHExt) Clone() *DOOHExt {
	if de == nil {
		return nil
	}

	clone := *de
	clone.ext = maps.Clone(de.ext)

	return &clone
}

// ---------------------------------------------------------------
// RegExt provides an interface for request.regs.ext
// ---------------------------------------------------------------

type RegExt struct {
	ext            map[string]json.RawMessage
	extDirty       bool
	dsa            *ExtRegsDSA
	dsaDirty       bool
	gdpr           *int8
	gdprDirty      bool
	gpc            *string
	gpcDirty       bool
	usPrivacy      string
	usPrivacyDirty bool
}

func (re *RegExt) unmarshal(extJson json.RawMessage) error {
	if len(re.ext) != 0 || re.Dirty() {
		return nil
	}

	re.ext = make(map[string]json.RawMessage)

	if len(extJson) == 0 {
		return nil
	}

	if err := jsonutil.Unmarshal(extJson, &re.ext); err != nil {
		return err
	}

	dsaJson, hasDSA := re.ext[dsaKey]
	if hasDSA {
		re.dsa = &ExtRegsDSA{}
	}
	if dsaJson != nil {
		if err := jsonutil.Unmarshal(dsaJson, re.dsa); err != nil {
			return err
		}
	}

	gdprJson, hasGDPR := re.ext[gdprKey]
	if hasGDPR && gdprJson != nil {
		if err := jsonutil.Unmarshal(gdprJson, &re.gdpr); err != nil {
			return errors.New("gdpr must be an integer")
		}
	}

	uspJson, hasUsp := re.ext[us_privacyKey]
	if hasUsp && uspJson != nil {
		if err := jsonutil.Unmarshal(uspJson, &re.usPrivacy); err != nil {
			return err
		}
	}

	gpcJson, hasGPC := re.ext[gpcKey]
	if hasGPC && gpcJson != nil {
		return jsonutil.ParseIntoString(gpcJson, &re.gpc)
	}

	return nil
}

func (re *RegExt) marshal() (json.RawMessage, error) {
	if re.dsaDirty {
		if re.dsa != nil {
			rawjson, err := jsonutil.Marshal(re.dsa)
			if err != nil {
				return nil, err
			}
			re.ext[dsaKey] = rawjson
		} else {
			delete(re.ext, dsaKey)
		}
		re.dsaDirty = false
	}

	if re.gdprDirty {
		if re.gdpr != nil {
			rawjson, err := jsonutil.Marshal(re.gdpr)
			if err != nil {
				return nil, err
			}
			re.ext[gdprKey] = rawjson
		} else {
			delete(re.ext, gdprKey)
		}
		re.gdprDirty = false
	}

	if re.usPrivacyDirty {
		if len(re.usPrivacy) > 0 {
			rawjson, err := jsonutil.Marshal(re.usPrivacy)
			if err != nil {
				return nil, err
			}
			re.ext[us_privacyKey] = rawjson
		} else {
			delete(re.ext, us_privacyKey)
		}
		re.usPrivacyDirty = false
	}

	if re.gpcDirty {
		if re.gpc != nil {
			rawjson, err := jsonutil.Marshal(re.gpc)
			if err != nil {
				return nil, err
			}
			re.ext[gpcKey] = rawjson
		} else {
			delete(re.ext, gpcKey)
		}
		re.gpcDirty = false
	}

	re.extDirty = false
	if len(re.ext) == 0 {
		return nil, nil
	}
	return jsonutil.Marshal(re.ext)
}

func (re *RegExt) Dirty() bool {
	return re.extDirty || re.dsaDirty || re.gdprDirty || re.usPrivacyDirty || re.gpcDirty
}

func (re *RegExt) GetExt() map[string]json.RawMessage {
	ext := make(map[string]json.RawMessage)
	for k, v := range re.ext {
		ext[k] = v
	}
	return ext
}

func (re *RegExt) SetExt(ext map[string]json.RawMessage) {
	re.ext = ext
	re.extDirty = true
}

func (re *RegExt) GetDSA() *ExtRegsDSA {
	if re.dsa == nil {
		return nil
	}
	dsa := *re.dsa
	return &dsa
}

func (re *RegExt) SetDSA(dsa *ExtRegsDSA) {
	re.dsa = dsa
	re.dsaDirty = true
}

func (re *RegExt) GetGDPR() *int8 {
	if re.gdpr == nil {
		return nil
	}
	gdpr := *re.gdpr
	return &gdpr
}

func (re *RegExt) SetGDPR(gdpr *int8) {
	re.gdpr = gdpr
	re.gdprDirty = true
}

func (re *RegExt) GetGPC() *string {
	if re.gpc == nil {
		return nil
	}
	gpc := *re.gpc
	return &gpc
}

func (re *RegExt) SetGPC(gpc *string) {
	re.gpc = gpc
	re.gpcDirty = true
}

func (re *RegExt) GetUSPrivacy() string {
	uSPrivacy := re.usPrivacy
	return uSPrivacy
}

func (re *RegExt) SetUSPrivacy(usPrivacy string) {
	re.usPrivacy = usPrivacy
	re.usPrivacyDirty = true
}

func (re *RegExt) Clone() *RegExt {
	if re == nil {
		return nil
	}

	clone := *re
	clone.ext = maps.Clone(re.ext)

	clone.gdpr = ptrutil.Clone(re.gdpr)

	return &clone
}

// ---------------------------------------------------------------
// SiteExt provides an interface for request.site.ext
// ---------------------------------------------------------------

type SiteExt struct {
	ext      map[string]json.RawMessage
	extDirty bool
	amp      *int8
	ampDirty bool
}

func (se *SiteExt) unmarshal(extJson json.RawMessage) error {
	if len(se.ext) != 0 || se.Dirty() {
		return nil
	}

	se.ext = make(map[string]json.RawMessage)

	if len(extJson) == 0 {
		return nil
	}

	if err := jsonutil.Unmarshal(extJson, &se.ext); err != nil {
		return err
	}

	ampJson, hasAmp := se.ext[ampKey]
	if hasAmp && ampJson != nil {
		if err := jsonutil.Unmarshal(ampJson, &se.amp); err != nil {
			return errors.New(`request.site.ext.amp must be either 1, 0, or undefined`)
		}
	}

	return nil
}

func (se *SiteExt) marshal() (json.RawMessage, error) {
	if se.ampDirty {
		if se.amp != nil {
			ampJson, err := jsonutil.Marshal(se.amp)
			if err != nil {
				return nil, err
			}
			se.ext[ampKey] = json.RawMessage(ampJson)
		} else {
			delete(se.ext, ampKey)
		}
		se.ampDirty = false
	}

	se.extDirty = false
	if len(se.ext) == 0 {
		return nil, nil
	}
	return jsonutil.Marshal(se.ext)
}

func (se *SiteExt) Dirty() bool {
	return se.extDirty || se.ampDirty
}

func (se *SiteExt) GetExt() map[string]json.RawMessage {
	ext := make(map[string]json.RawMessage)
	for k, v := range se.ext {
		ext[k] = v
	}
	return ext
}

func (se *SiteExt) SetExt(ext map[string]json.RawMessage) {
	se.ext = ext
	se.extDirty = true
}

func (se *SiteExt) GetAmp() *int8 {
	return se.amp
}

func (se *SiteExt) SetAmp(amp *int8) {
	se.amp = amp
	se.ampDirty = true
}

func (se *SiteExt) Clone() *SiteExt {
	if se == nil {
		return nil
	}

	clone := *se
	clone.ext = maps.Clone(se.ext)
	clone.amp = ptrutil.Clone(se.amp)

	return &clone
}

// ---------------------------------------------------------------
// SourceExt provides an interface for request.source.ext
// ---------------------------------------------------------------

type SourceExt struct {
	ext         map[string]json.RawMessage
	extDirty    bool
	schain      *openrtb2.SupplyChain
	schainDirty bool
}

func (se *SourceExt) unmarshal(extJson json.RawMessage) error {
	if len(se.ext) != 0 || se.Dirty() {
		return nil
	}

	se.ext = make(map[string]json.RawMessage)

	if len(extJson) == 0 {
		return nil
	}

	if err := jsonutil.Unmarshal(extJson, &se.ext); err != nil {
		return err
	}

	schainJson, hasSChain := se.ext[schainKey]
	if hasSChain && schainJson != nil {
		if err := jsonutil.Unmarshal(schainJson, &se.schain); err != nil {
			return err
		}
	}

	return nil
}

func (se *SourceExt) marshal() (json.RawMessage, error) {
	if se.schainDirty {
		if se.schain != nil {
			schainJson, err := jsonutil.Marshal(se.schain)
			if err != nil {
				return nil, err
			}
			if len(schainJson) > jsonEmptyObjectLength {
				se.ext[schainKey] = json.RawMessage(schainJson)
			} else {
				delete(se.ext, schainKey)
			}
		} else {
			delete(se.ext, schainKey)
		}
		se.schainDirty = false
	}

	se.extDirty = false
	if len(se.ext) == 0 {
		return nil, nil
	}
	return jsonutil.Marshal(se.ext)
}

func (se *SourceExt) Dirty() bool {
	return se.extDirty || se.schainDirty
}

func (se *SourceExt) GetExt() map[string]json.RawMessage {
	ext := make(map[string]json.RawMessage)
	for k, v := range se.ext {
		ext[k] = v
	}
	return ext
}

func (se *SourceExt) SetExt(ext map[string]json.RawMessage) {
	se.ext = ext
	se.extDirty = true
}

func (se *SourceExt) GetSChain() *openrtb2.SupplyChain {
	if se.schain == nil {
		return nil
	}
	schain := *se.schain
	return &schain
}

func (se *SourceExt) SetSChain(schain *openrtb2.SupplyChain) {
	se.schain = schain
	se.schainDirty = true
}

func (se *SourceExt) Clone() *SourceExt {
	if se == nil {
		return nil
	}

	clone := *se
	clone.ext = maps.Clone(se.ext)

	clone.schain = cloneSupplyChain(se.schain)

	return &clone
}

// ImpWrapper wraps an OpenRTB impression object to provide storage for unmarshalled ext fields, so they
// will not need to be unmarshalled multiple times. It is intended to use the ImpWrapper via the RequestWrapper
// and follow the same usage conventions.
type ImpWrapper struct {
	*openrtb2.Imp
	impExt *ImpExt
}

func (w *ImpWrapper) GetImpExt() (*ImpExt, error) {
	if w.impExt != nil {
		return w.impExt, nil
	}
	w.impExt = &ImpExt{}
	if w.Imp == nil || w.Ext == nil {
		return w.impExt, w.impExt.unmarshal(json.RawMessage{})
	}
	return w.impExt, w.impExt.unmarshal(w.Ext)
}

func (w *ImpWrapper) RebuildImp() error {
	if w.Imp == nil {
		return errors.New("ImpWrapper RebuildImp called on a nil Imp")
	}

	if err := w.rebuildImpExt(); err != nil {
		return err
	}

	return nil
}

func (w *ImpWrapper) rebuildImpExt() error {
	if w.impExt == nil || !w.impExt.Dirty() {
		return nil
	}

	impJson, err := w.impExt.marshal()
	if err != nil {
		return err
	}

	w.Ext = impJson

	return nil
}

func (w *ImpWrapper) Clone() *ImpWrapper {
	if w == nil {
		return nil
	}

	clone := *w
	clone.impExt = w.impExt.Clone()

	return &clone
}

// ---------------------------------------------------------------
// ImpExt provides an interface for imp.ext
// ---------------------------------------------------------------

type ImpExt struct {
	ext         map[string]json.RawMessage
	extDirty    bool
	prebid      *ExtImpPrebid
	data        *ExtImpData
	prebidDirty bool
	tid         string
	gpId        string
	tidDirty    bool
}

func (e *ImpExt) unmarshal(extJson json.RawMessage) error {
	if len(e.ext) != 0 || e.Dirty() {
		return nil
	}

	e.ext = make(map[string]json.RawMessage)

	if len(extJson) == 0 {
		return nil
	}

	if err := jsonutil.Unmarshal(extJson, &e.ext); err != nil {
		return err
	}

	prebidJson, hasPrebid := e.ext[prebidKey]
	if hasPrebid {
		e.prebid = &ExtImpPrebid{}
	}
	if prebidJson != nil {
		if err := jsonutil.Unmarshal(prebidJson, e.prebid); err != nil {
			return err
		}
	}

	dataJson, hasData := e.ext[dataKey]
	if hasData {
		e.data = &ExtImpData{}
	}
	if dataJson != nil {
		if err := jsonutil.Unmarshal(dataJson, e.data); err != nil {
			return err
		}
	}

	tidJson, hasTid := e.ext["tid"]
	if hasTid && tidJson != nil {
		if err := jsonutil.Unmarshal(tidJson, &e.tid); err != nil {
			return err
		}
	}

	gpIdJson, hasGpId := e.ext["gpid"]
	if hasGpId && gpIdJson != nil {
		if err := jsonutil.Unmarshal(gpIdJson, &e.gpId); err != nil {
			return err
		}
	}

	return nil
}

func (e *ImpExt) marshal() (json.RawMessage, error) {
	if e.prebidDirty {
		if e.prebid != nil {
			prebidJson, err := jsonutil.Marshal(e.prebid)
			if err != nil {
				return nil, err
			}
			if len(prebidJson) > jsonEmptyObjectLength {
				e.ext[prebidKey] = json.RawMessage(prebidJson)
			} else {
				delete(e.ext, prebidKey)
			}
		} else {
			delete(e.ext, prebidKey)
		}
		e.prebidDirty = false
	}

	if e.tidDirty {
		if len(e.tid) > 0 {
			tidJson, err := jsonutil.Marshal(e.tid)
			if err != nil {
				return nil, err
			}
			e.ext["tid"] = tidJson
		} else {
			delete(e.ext, "tid")
		}
		e.tidDirty = false
	}

	e.extDirty = false
	if len(e.ext) == 0 {
		return nil, nil
	}
	return jsonutil.Marshal(e.ext)
}

func (e *ImpExt) Dirty() bool {
	return e.extDirty || e.prebidDirty || e.tidDirty
}

func (e *ImpExt) GetExt() map[string]json.RawMessage {
	ext := make(map[string]json.RawMessage)
	for k, v := range e.ext {
		ext[k] = v
	}
	return ext
}

func (e *ImpExt) SetExt(ext map[string]json.RawMessage) {
	e.ext = ext
	e.extDirty = true
}

func (e *ImpExt) GetPrebid() *ExtImpPrebid {
	if e.prebid == nil {
		return nil
	}
	prebid := *e.prebid
	return &prebid
}

func (e *ImpExt) GetOrCreatePrebid() *ExtImpPrebid {
	if e.prebid == nil {
		e.prebid = &ExtImpPrebid{}
	}
	return e.GetPrebid()
}

func (e *ImpExt) SetPrebid(prebid *ExtImpPrebid) {
	e.prebid = prebid
	e.prebidDirty = true
}

func (e *ImpExt) GetData() *ExtImpData {
	if e.data == nil {
		return nil
	}
	data := *e.data
	return &data
}

func (e *ImpExt) GetTid() string {
	tid := e.tid
	return tid
}

func (e *ImpExt) SetTid(tid string) {
	e.tid = tid
	e.tidDirty = true
}

func (e *ImpExt) GetGpId() string {
	gpId := e.gpId
	return gpId
}

func CreateImpExtForTesting(ext map[string]json.RawMessage, prebid *ExtImpPrebid) ImpExt {
	return ImpExt{ext: ext, prebid: prebid}
}

func (e *ImpExt) Clone() *ImpExt {
	if e == nil {
		return nil
	}

	clone := *e
	clone.ext = maps.Clone(e.ext)

	if e.prebid != nil {
		clonedPrebid := *e.prebid
		clonedPrebid.StoredRequest = ptrutil.Clone(e.prebid.StoredRequest)
		clonedPrebid.StoredAuctionResponse = ptrutil.Clone(e.prebid.StoredAuctionResponse)
		if e.prebid.StoredBidResponse != nil {
			clonedPrebid.StoredBidResponse = make([]ExtStoredBidResponse, len(e.prebid.StoredBidResponse))
			for i, sbr := range e.prebid.StoredBidResponse {
				clonedPrebid.StoredBidResponse[i] = sbr
				clonedPrebid.StoredBidResponse[i].ReplaceImpId = ptrutil.Clone(sbr.ReplaceImpId)
			}
		}
		clonedPrebid.IsRewardedInventory = ptrutil.Clone(e.prebid.IsRewardedInventory)
		clonedPrebid.Bidder = maps.Clone(e.prebid.Bidder)
		clonedPrebid.Options = ptrutil.Clone(e.prebid.Options)
		clonedPrebid.Floors = ptrutil.Clone(e.prebid.Floors)
		clone.prebid = &clonedPrebid
	}

	return &clone
}
