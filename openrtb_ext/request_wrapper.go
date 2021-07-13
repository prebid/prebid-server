package openrtb_ext

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
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

type RequestWrapper struct {
	// json json.RawMessage
	*openrtb2.BidRequest
	// Dirty bool // Probably don't care
	userExt    *UserExt
	deviceExt  *DeviceExt
	requestExt *RequestExt
	appExt     *AppExt
	regExt     *RegExt
	siteExt    *SiteExt
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

func (rw *RequestWrapper) RebuildRequest() error {
	if rw.BidRequest == nil {
		return fmt.Errorf("Requestwrapper Sync called on a nil Request")
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

	return nil
}

func (rw *RequestWrapper) rebuildUserExt() error {
	if rw.BidRequest.User == nil && rw.userExt != nil && rw.userExt.Dirty() {
		rw.User = &openrtb2.User{}
	}
	if rw.userExt != nil && rw.userExt.Dirty() {
		userJson, err := rw.userExt.marshal()
		if err != nil {
			return err
		}
		rw.User.Ext = userJson
	}
	return nil
}

func (rw *RequestWrapper) rebuildDeviceExt() error {
	if rw.Device == nil && rw.deviceExt != nil && rw.deviceExt.Dirty() {
		rw.Device = &openrtb2.Device{}
	}
	if rw.deviceExt != nil && rw.deviceExt.Dirty() {
		deviceJson, err := rw.deviceExt.marshal()
		if err != nil {
			return err
		}
		rw.Device.Ext = deviceJson
	}
	return nil
}

func (rw *RequestWrapper) rebuildRequestExt() error {
	if rw.requestExt != nil && rw.requestExt.Dirty() {
		requestJson, err := rw.requestExt.marshal()
		if err != nil {
			return err
		}
		rw.Ext = requestJson
	}
	return nil
}

func (rw *RequestWrapper) rebuildAppExt() error {
	if rw.App == nil && rw.appExt != nil && rw.appExt.Dirty() {
		rw.App = &openrtb2.App{}
	}
	if rw.appExt != nil && rw.appExt.Dirty() {
		appJson, err := rw.appExt.marshal()
		if err != nil {
			return err
		}
		rw.App.Ext = appJson
	}
	return nil
}

func (rw *RequestWrapper) rebuildRegExt() error {
	if rw.Regs == nil && rw.regExt != nil && rw.regExt.Dirty() {
		rw.Regs = &openrtb2.Regs{}
	}
	if rw.regExt != nil && rw.regExt.Dirty() {
		regsJson, err := rw.regExt.marshal()
		if err != nil {
			return err
		}
		rw.Regs.Ext = regsJson
	}
	return nil
}

func (rw *RequestWrapper) rebuildSiteExt() error {
	if rw.Site == nil && rw.siteExt != nil && rw.siteExt.Dirty() {
		rw.Site = &openrtb2.Site{}
	}
	if rw.siteExt != nil && rw.siteExt.Dirty() {
		siteJson, err := rw.siteExt.marshal()
		if err != nil {
			return err
		}
		rw.Regs.Ext = siteJson
	}
	return nil
}

// ---------------------------------------------------------------
// UserExt provides an interface for request.user.ext
// ---------------------------------------------------------------

type UserExt struct {
	ext            map[string]json.RawMessage
	extDirty       bool
	consent        *string
	consentDirty   bool
	prebid         *ExtUserPrebid
	prebidDirty    bool
	digiTrust      *ExtUserDigiTrust
	digiTrustDirty bool
	eids           *[]ExtUserEid
	eidsDirty      bool
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

	digiTrustJson, hasDigiTrust := ue.ext["digitrust"]
	if hasDigiTrust {
		ue.digiTrust = &ExtUserDigiTrust{}
		if err := json.Unmarshal(digiTrustJson, ue.digiTrust); err != nil {
			return err
		}
	}

	eidsJson, hasEids := ue.ext["eids"]
	if hasEids {
		ue.eids = &[]ExtUserEid{}
		if err := json.Unmarshal(eidsJson, ue.eids); err != nil {
			return err
		}
	}

	return nil
}

func (ue *UserExt) marshal() (json.RawMessage, error) {
	if ue.consentDirty {
		consentJson, err := json.Marshal(ue.consent)
		if err != nil {
			return nil, err
		}
		if len(consentJson) > 0 {
			ue.ext["consent"] = json.RawMessage(consentJson)
		} else {
			delete(ue.ext, "consent")
		}
		ue.consentDirty = false
	}

	if ue.prebidDirty {
		prebidJson, err := json.Marshal(ue.prebid)
		if err != nil {
			return nil, err
		}
		if len(prebidJson) > 0 {
			ue.ext["prebid"] = json.RawMessage(prebidJson)
		} else {
			delete(ue.ext, "prebid")
		}
		ue.prebidDirty = false
	}

	if ue.digiTrustDirty {
		digiTrustJson, err := json.Marshal(ue.digiTrust)
		if err != nil {
			return nil, err
		}
		if len(digiTrustJson) > 0 {
			ue.ext["digitrust"] = json.RawMessage(digiTrustJson)
		} else {
			delete(ue.ext, "digitrust")
		}
		ue.digiTrustDirty = false
	}

	if ue.eidsDirty {
		if len(*ue.eids) > 0 {
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

	return json.Marshal(ue.ext)

}

func (ue *UserExt) Dirty() bool {
	return ue.extDirty || ue.digiTrustDirty || ue.eidsDirty || ue.prebidDirty || ue.consentDirty
}

func (ue *UserExt) GetExt() map[string]json.RawMessage {
	ext := ue.ext
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

func (ue *UserExt) GetDigiTrust() *ExtUserDigiTrust {
	if ue.digiTrust == nil {
		return nil
	}
	digiTrust := *ue.digiTrust
	return &digiTrust
}

func (ue *UserExt) SetDigiTrust(digiTrust *ExtUserDigiTrust) {
	ue.digiTrust = digiTrust
	ue.digiTrustDirty = true
}

func (ue *UserExt) GetEid() *[]ExtUserEid {
	if ue.eids == nil {
		return nil
	}
	eids := *ue.eids
	return &eids
}

func (ue *UserExt) SetEid(eid *[]ExtUserEid) {
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
}

func (re *RequestExt) unmarshal(extJson json.RawMessage) error {
	if len(re.ext) != 0 || re.Dirty() {
		return nil
	}
	re.ext = make(map[string]json.RawMessage)
	if len(extJson) == 0 {
		return nil
	}
	err := json.Unmarshal(extJson, &re.ext)
	if err != nil {
		return err
	}
	prebidJson, hasPrebid := re.ext["prebid"]
	if hasPrebid {
		re.prebid = &ExtRequestPrebid{}
		err = json.Unmarshal(prebidJson, re.prebid)
	}

	return err
}

func (re *RequestExt) marshal() (json.RawMessage, error) {
	if re.prebidDirty {
		prebidJson, err := json.Marshal(re.prebid)
		if err != nil {
			return nil, err
		}
		if len(prebidJson) > 2 {
			re.ext["prebid"] = json.RawMessage(prebidJson)
		} else {
			delete(re.ext, "prebid")
		}
		re.extDirty = false
		re.prebidDirty = false
	}

	return json.Marshal(re.ext)
}

func (re *RequestExt) Dirty() bool {
	return re.extDirty || re.prebidDirty
}

func (re *RequestExt) GetExt() map[string]json.RawMessage {
	ext := re.ext
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
	err := json.Unmarshal(extJson, &de.ext)
	if err != nil {
		return err
	}
	prebidJson, hasPrebid := de.ext["prebid"]
	if hasPrebid {
		de.prebid = &ExtDevicePrebid{}
		err = json.Unmarshal(prebidJson, de.prebid)
	}

	return err
}

func (de *DeviceExt) marshal() (json.RawMessage, error) {
	if de.prebidDirty {
		prebidJson, err := json.Marshal(de.prebid)
		if err != nil {
			return nil, err
		}
		if len(prebidJson) > 0 {
			de.ext["prebid"] = json.RawMessage(prebidJson)
		} else {
			delete(de.ext, "prebid")
		}
		de.extDirty = false
		de.prebidDirty = false
	}

	rawJson, err := json.Marshal(de.ext)
	if err == nil {
		de.prebidDirty = false
	}

	return rawJson, err
}

func (de *DeviceExt) Dirty() bool {
	return de.extDirty || de.prebidDirty
}

func (de *DeviceExt) GetExt() map[string]json.RawMessage {
	ext := de.ext
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
	err := json.Unmarshal(extJson, &ae.ext)
	if err != nil {
		return err
	}
	prebidJson, hasPrebid := ae.ext["prebid"]
	if hasPrebid {
		ae.prebid = &ExtAppPrebid{}
		err = json.Unmarshal(prebidJson, ae.prebid)
	}

	return err
}

func (ae *AppExt) marshal() (json.RawMessage, error) {
	if ae.prebidDirty {
		prebidJson, err := json.Marshal(ae.prebid)
		if err != nil {
			return nil, err
		}
		if len(prebidJson) > 0 {
			ae.ext["prebid"] = json.RawMessage(prebidJson)
		} else {
			delete(ae.ext, "prebid")
		}
	}

	rawJson, err := json.Marshal(ae.ext)
	if err == nil {
		ae.prebidDirty = false
	}
	ae.extDirty = false
	return rawJson, err
}

func (ae *AppExt) Dirty() bool {
	return ae.extDirty || ae.prebidDirty
}

func (ae *AppExt) GetExt() map[string]json.RawMessage {
	ext := ae.ext
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
	uSPrivacy      string
	uSPrivacyDirty bool
}

func (re *RegExt) unmarshal(extJson json.RawMessage) error {
	if len(re.ext) != 0 || re.Dirty() {
		return nil
	}
	re.ext = make(map[string]json.RawMessage)
	if len(extJson) == 0 {
		return nil
	}
	err := json.Unmarshal(extJson, &re.ext)
	if err != nil {
		return err
	}
	uspJson, hasUsp := re.ext["us_privacy"]
	if hasUsp {
		err = json.Unmarshal(uspJson, &re.uSPrivacy)
	}

	return err
}

func (re *RegExt) marshal() (json.RawMessage, error) {
	if re.uSPrivacyDirty {
		if len(re.uSPrivacy) > 0 {
			rawjson, err := json.Marshal(re.uSPrivacy)
			if err != nil {
				return nil, err
			}
			re.ext["us_privacy"] = rawjson
		} else {
			delete(re.ext, "us_privacy")
		}
	}
	if len(re.ext) == 0 {
		re.uSPrivacyDirty = false
		return nil, nil
	}

	rawJson, err := json.Marshal(re.ext)
	if err == nil {
		re.uSPrivacyDirty = false
	}
	return rawJson, err
}

func (re *RegExt) Dirty() bool {
	return re.extDirty || re.uSPrivacyDirty
}

func (re *RegExt) GetExt() map[string]json.RawMessage {
	ext := re.ext
	return ext
}

func (re *RegExt) SetExt(ext map[string]json.RawMessage) {
	re.ext = ext
	re.extDirty = true
}

func (re *RegExt) GetUSPrivacy() string {
	uSPrivacy := re.uSPrivacy
	return uSPrivacy
}

func (re *RegExt) SetUSPrivacy(uSPrivacy string) {
	re.uSPrivacy = uSPrivacy
	re.uSPrivacyDirty = true
}

// ---------------------------------------------------------------
// SiteExt provides an interface for request.site.ext
// ---------------------------------------------------------------

type SiteExt struct {
	ext      map[string]json.RawMessage
	extDirty bool
	amp      int8
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
	err := json.Unmarshal(extJson, &se.ext)
	if err != nil {
		return err
	}
	AmpJson, hasAmp := se.ext["amp"]
	if hasAmp {
		err = json.Unmarshal(AmpJson, &se.amp)
		// Replace with a more specific error message
		if err != nil {
			err = errors.New(`request.site.ext.amp must be either 1, 0, or undefined`)
		}
	}

	return err
}

func (se *SiteExt) marshal() (json.RawMessage, error) {
	if se.ampDirty {
		ampJson, err := json.Marshal(se.amp)
		if err != nil {
			return nil, err
		}
		if len(ampJson) > 0 {
			se.ext["amp"] = json.RawMessage(ampJson)
		} else {
			delete(se.ext, "amp")
		}
		se.ampDirty = false
	}

	rawJson, err := json.Marshal(se.ext)
	if err == nil {
		se.ampDirty = false
	}
	return rawJson, err
}

func (se *SiteExt) Dirty() bool {
	return se.extDirty || se.ampDirty
}

func (se *SiteExt) GetExt() map[string]json.RawMessage {
	ext := se.ext
	return ext
}

func (se *SiteExt) SetExt(ext map[string]json.RawMessage) {
	se.ext = ext
	se.extDirty = true
}

func (se *SiteExt) GetAmp() int8 {
	return se.amp
}

func (se *SiteExt) SetUSPrivacy(amp int8) {
	se.amp = amp
	se.ampDirty = true
}
