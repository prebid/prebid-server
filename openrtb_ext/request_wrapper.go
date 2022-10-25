package openrtb_ext

import (
	"encoding/json"
	"errors"

	"github.com/prebid/openrtb/v17/openrtb2"
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
	sourceExt           *SourceExt
}

const (
	jsonEmptyObjectLength               = 2
	ConsentedProvidersSettingsStringKey = "ConsentedProvidersSettings"
	ConsentedProvidersSettingsListKey   = "consented_providers_settings"
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

	if err := json.Unmarshal(extJson, &ue.ext); err != nil {
		return err
	}

	consentJson, hasConsent := ue.ext["consent"]
	if hasConsent {
		if err := json.Unmarshal(consentJson, &ue.consent); err != nil {
			return err
		}
	}

	prebidJson, hasPrebid := ue.ext["prebid"]
	if hasPrebid {
		ue.prebid = &ExtUserPrebid{}
		if err := json.Unmarshal(prebidJson, ue.prebid); err != nil {
			return err
		}
	}

	eidsJson, hasEids := ue.ext["eids"]
	if hasEids {
		ue.eids = &[]openrtb2.EID{}
		if err := json.Unmarshal(eidsJson, ue.eids); err != nil {
			return err
		}
	}

	if consentedProviderSettingsJson, hasCPSettings := ue.ext[ConsentedProvidersSettingsStringKey]; hasCPSettings {
		ue.consentedProvidersSettingsIn = &ConsentedProvidersSettingsIn{}
		if err := json.Unmarshal(consentedProviderSettingsJson, ue.consentedProvidersSettingsIn); err != nil {
			return err
		}
	}

	if consentedProviderSettingsJson, hasCPSettings := ue.ext[ConsentedProvidersSettingsListKey]; hasCPSettings {
		ue.consentedProvidersSettingsOut = &ConsentedProvidersSettingsOut{}
		if err := json.Unmarshal(consentedProviderSettingsJson, ue.consentedProvidersSettingsOut); err != nil {
			return err
		}
	}

	return nil
}

func (ue *UserExt) marshal() (json.RawMessage, error) {
	if ue.consentDirty {
		if ue.consent != nil && len(*ue.consent) > 0 {
			consentJson, err := json.Marshal(ue.consent)
			if err != nil {
				return nil, err
			}
			ue.ext["consent"] = json.RawMessage(consentJson)
		} else {
			delete(ue.ext, "consent")
		}
		ue.consentDirty = false
	}

	if ue.prebidDirty {
		if ue.prebid != nil {
			prebidJson, err := json.Marshal(ue.prebid)
			if err != nil {
				return nil, err
			}
			if len(prebidJson) > jsonEmptyObjectLength {
				ue.ext["prebid"] = json.RawMessage(prebidJson)
			} else {
				delete(ue.ext, "prebid")
			}
		} else {
			delete(ue.ext, "prebid")
		}
		ue.prebidDirty = false
	}

	if ue.consentedProvidersSettingsInDirty {
		if ue.consentedProvidersSettingsIn != nil {
			cpSettingsJson, err := json.Marshal(ue.consentedProvidersSettingsIn)
			if err != nil {
				return nil, err
			}
			if len(cpSettingsJson) > jsonEmptyObjectLength {
				ue.ext[ConsentedProvidersSettingsStringKey] = json.RawMessage(cpSettingsJson)
			} else {
				delete(ue.ext, ConsentedProvidersSettingsStringKey)
			}
		} else {
			delete(ue.ext, ConsentedProvidersSettingsStringKey)
		}
		ue.consentedProvidersSettingsInDirty = false
	}

	if ue.consentedProvidersSettingsOutDirty {
		if ue.consentedProvidersSettingsOut != nil {
			cpSettingsJson, err := json.Marshal(ue.consentedProvidersSettingsOut)
			if err != nil {
				return nil, err
			}
			if len(cpSettingsJson) > jsonEmptyObjectLength {
				ue.ext[ConsentedProvidersSettingsListKey] = json.RawMessage(cpSettingsJson)
			} else {
				delete(ue.ext, ConsentedProvidersSettingsListKey)
			}
		} else {
			delete(ue.ext, ConsentedProvidersSettingsListKey)
		}
		ue.consentedProvidersSettingsOutDirty = false
	}

	if ue.eidsDirty {
		if ue.eids != nil && len(*ue.eids) > 0 {
			eidsJson, err := json.Marshal(ue.eids)
			if err != nil {
				return nil, err
			}
			ue.ext["eids"] = json.RawMessage(eidsJson)
		} else {
			delete(ue.ext, "eids")
		}
		ue.eidsDirty = false
	}

	ue.extDirty = false
	if len(ue.ext) == 0 {
		return nil, nil
	}
	return json.Marshal(ue.ext)
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
	return
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

	if err := json.Unmarshal(extJson, &re.ext); err != nil {
		return err
	}

	prebidJson, hasPrebid := re.ext["prebid"]
	if hasPrebid {
		re.prebid = &ExtRequestPrebid{}
		if err := json.Unmarshal(prebidJson, re.prebid); err != nil {
			return err
		}
	}

	schainJson, hasSChain := re.ext["schain"]
	if hasSChain {
		re.schain = &openrtb2.SupplyChain{}
		if err := json.Unmarshal(schainJson, re.schain); err != nil {
			return err
		}
	}

	return nil
}

func (re *RequestExt) marshal() (json.RawMessage, error) {
	if re.prebidDirty {
		if re.prebid != nil {
			prebidJson, err := json.Marshal(re.prebid)
			if err != nil {
				return nil, err
			}
			if len(prebidJson) > jsonEmptyObjectLength {
				re.ext["prebid"] = json.RawMessage(prebidJson)
			} else {
				delete(re.ext, "prebid")
			}
		} else {
			delete(re.ext, "prebid")
		}
		re.prebidDirty = false
	}

	if re.schainDirty {
		if re.schain != nil {
			schainJson, err := json.Marshal(re.schain)
			if err != nil {
				return nil, err
			}
			if len(schainJson) > jsonEmptyObjectLength {
				re.ext["schain"] = json.RawMessage(schainJson)
			} else {
				delete(re.ext, "schain")
			}
		} else {
			delete(re.ext, "schain")
		}
		re.schainDirty = false
	}

	re.extDirty = false
	if len(re.ext) == 0 {
		return nil, nil
	}
	return json.Marshal(re.ext)
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
	if re.prebid == nil {
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
}

func (de *DeviceExt) unmarshal(extJson json.RawMessage) error {
	if len(de.ext) != 0 || de.Dirty() {
		return nil
	}

	de.ext = make(map[string]json.RawMessage)

	if len(extJson) == 0 {
		return nil
	}

	if err := json.Unmarshal(extJson, &de.ext); err != nil {
		return err
	}

	prebidJson, hasPrebid := de.ext["prebid"]
	if hasPrebid {
		de.prebid = &ExtDevicePrebid{}
		if err := json.Unmarshal(prebidJson, de.prebid); err != nil {
			return err
		}
	}

	return nil
}

func (de *DeviceExt) marshal() (json.RawMessage, error) {
	if de.prebidDirty {
		if de.prebid != nil {
			prebidJson, err := json.Marshal(de.prebid)
			if err != nil {
				return nil, err
			}
			if len(prebidJson) > jsonEmptyObjectLength {
				de.ext["prebid"] = json.RawMessage(prebidJson)
			} else {
				delete(de.ext, "prebid")
			}
		} else {
			delete(de.ext, "prebid")
		}
		de.prebidDirty = false
	}

	de.extDirty = false
	if len(de.ext) == 0 {
		return nil, nil
	}
	return json.Marshal(de.ext)
}

func (de *DeviceExt) Dirty() bool {
	return de.extDirty || de.prebidDirty
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

	if err := json.Unmarshal(extJson, &ae.ext); err != nil {
		return err
	}

	prebidJson, hasPrebid := ae.ext["prebid"]
	if hasPrebid {
		ae.prebid = &ExtAppPrebid{}
		if err := json.Unmarshal(prebidJson, ae.prebid); err != nil {
			return err
		}
	}

	return nil
}

func (ae *AppExt) marshal() (json.RawMessage, error) {
	if ae.prebidDirty {
		if ae.prebid != nil {
			prebidJson, err := json.Marshal(ae.prebid)
			if err != nil {
				return nil, err
			}
			if len(prebidJson) > jsonEmptyObjectLength {
				ae.ext["prebid"] = json.RawMessage(prebidJson)
			} else {
				delete(ae.ext, "prebid")
			}
		} else {
			delete(ae.ext, "prebid")
		}
		ae.prebidDirty = false
	}

	ae.extDirty = false
	if len(ae.ext) == 0 {
		return nil, nil
	}
	return json.Marshal(ae.ext)
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

// ---------------------------------------------------------------
// RegExt provides an interface for request.regs.ext
// ---------------------------------------------------------------

type RegExt struct {
	ext            map[string]json.RawMessage
	extDirty       bool
	gdpr           *int8
	gdprDirty      bool
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

	if err := json.Unmarshal(extJson, &re.ext); err != nil {
		return err
	}

	gdprJson, hasGDPR := re.ext["gdpr"]
	if hasGDPR {
		if err := json.Unmarshal(gdprJson, &re.gdpr); err != nil {
			return errors.New("gdpr must be an integer")
		}
	}

	uspJson, hasUsp := re.ext["us_privacy"]
	if hasUsp {
		if err := json.Unmarshal(uspJson, &re.usPrivacy); err != nil {
			return err
		}
	}

	return nil
}

func (re *RegExt) marshal() (json.RawMessage, error) {
	if re.gdprDirty {
		if re.gdpr != nil {
			rawjson, err := json.Marshal(re.gdpr)
			if err != nil {
				return nil, err
			}
			re.ext["gdpr"] = rawjson
		} else {
			delete(re.ext, "gdpr")
		}
		re.gdprDirty = false
	}

	if re.usPrivacyDirty {
		if len(re.usPrivacy) > 0 {
			rawjson, err := json.Marshal(re.usPrivacy)
			if err != nil {
				return nil, err
			}
			re.ext["us_privacy"] = rawjson
		} else {
			delete(re.ext, "us_privacy")
		}
		re.usPrivacyDirty = false
	}

	re.extDirty = false
	if len(re.ext) == 0 {
		return nil, nil
	}
	return json.Marshal(re.ext)
}

func (re *RegExt) Dirty() bool {
	return re.extDirty || re.gdprDirty || re.usPrivacyDirty
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

func (re *RegExt) GetGDPR() *int8 {
	gdpr := re.gdpr
	return gdpr
}

func (re *RegExt) SetGDPR(gdpr *int8) {
	re.gdpr = gdpr
	re.gdprDirty = true
}

func (re *RegExt) GetUSPrivacy() string {
	uSPrivacy := re.usPrivacy
	return uSPrivacy
}

func (re *RegExt) SetUSPrivacy(usPrivacy string) {
	re.usPrivacy = usPrivacy
	re.usPrivacyDirty = true
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

	if err := json.Unmarshal(extJson, &se.ext); err != nil {
		return err
	}

	ampJson, hasAmp := se.ext["amp"]
	if hasAmp {
		if err := json.Unmarshal(ampJson, &se.amp); err != nil {
			return errors.New(`request.site.ext.amp must be either 1, 0, or undefined`)
		}
	}

	return nil
}

func (se *SiteExt) marshal() (json.RawMessage, error) {
	if se.ampDirty {
		if se.amp != nil {
			ampJson, err := json.Marshal(se.amp)
			if err != nil {
				return nil, err
			}
			se.ext["amp"] = json.RawMessage(ampJson)
		} else {
			delete(se.ext, "amp")
		}
		se.ampDirty = false
	}

	se.extDirty = false
	if len(se.ext) == 0 {
		return nil, nil
	}
	return json.Marshal(se.ext)
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

	if err := json.Unmarshal(extJson, &se.ext); err != nil {
		return err
	}

	schainJson, hasSChain := se.ext["schain"]
	if hasSChain {
		if err := json.Unmarshal(schainJson, &se.schain); err != nil {
			return err
		}
	}

	return nil
}

func (se *SourceExt) marshal() (json.RawMessage, error) {
	if se.schainDirty {
		if se.schain != nil {
			schainJson, err := json.Marshal(se.schain)
			if err != nil {
				return nil, err
			}
			if len(schainJson) > jsonEmptyObjectLength {
				se.ext["schain"] = json.RawMessage(schainJson)
			} else {
				delete(se.ext, "schain")
			}
		} else {
			delete(se.ext, "schain")
		}
		se.schainDirty = false
	}

	se.extDirty = false
	if len(se.ext) == 0 {
		return nil, nil
	}
	return json.Marshal(se.ext)
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

// ---------------------------------------------------------------
// ImpExt provides an interface for imp.ext
// ---------------------------------------------------------------

type ImpExt struct {
	ext         map[string]json.RawMessage
	extDirty    bool
	prebid      *ExtImpPrebid
	prebidDirty bool
}

func (e *ImpExt) unmarshal(extJson json.RawMessage) error {
	if len(e.ext) != 0 || e.Dirty() {
		return nil
	}

	e.ext = make(map[string]json.RawMessage)

	if len(extJson) == 0 {
		return nil
	}

	if err := json.Unmarshal(extJson, &e.ext); err != nil {
		return err
	}

	prebidJson, hasPrebid := e.ext["prebid"]
	if hasPrebid {
		e.prebid = &ExtImpPrebid{}
		if err := json.Unmarshal(prebidJson, e.prebid); err != nil {
			return err
		}
	}

	return nil
}

func (e *ImpExt) marshal() (json.RawMessage, error) {
	if e.prebidDirty {
		if e.prebid != nil {
			prebidJson, err := json.Marshal(e.prebid)
			if err != nil {
				return nil, err
			}
			if len(prebidJson) > jsonEmptyObjectLength {
				e.ext["prebid"] = json.RawMessage(prebidJson)
			} else {
				delete(e.ext, "prebid")
			}
		} else {
			delete(e.ext, "prebid")
		}
		e.prebidDirty = false
	}

	e.extDirty = false
	if len(e.ext) == 0 {
		return nil, nil
	}
	return json.Marshal(e.ext)
}

func (e *ImpExt) Dirty() bool {
	return e.extDirty || e.prebidDirty
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

func CreateImpExtForTesting(ext map[string]json.RawMessage, prebid *ExtImpPrebid) ImpExt {
	return ImpExt{ext: ext, prebid: prebid}
}
