package openrtb_ext

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
)

// RequestWrapper wraps the OpenRTB request to provide a storage location for unmarshalled ext fields, so they
// will not need to be unmarshalled multiple times.
type RequestWrapper struct {
	// json json.RawMessage
	Request *openrtb2.BidRequest
	// Dirty bool // Probably don't care
	UserExt    *UserExt
	DeviceExt  *DeviceExt
	RequestExt *RequestExt
	AppExt     *AppExt
	RegExt     *RegExt
	SiteExt    *SiteExt
}

func (rw *RequestWrapper) ExtractUserExt() error {
	if rw.UserExt != nil {
		return nil
	}
	rw.UserExt = &UserExt{}
	if rw.Request == nil || rw.Request.User == nil || rw.Request.User.Ext == nil {
		return rw.UserExt.Unmarshal(json.RawMessage{})
	}

	return rw.UserExt.Unmarshal(rw.Request.User.Ext)
}

func (rw *RequestWrapper) ExtractDeviceExt() error {
	if rw.DeviceExt != nil {
		return nil
	}
	rw.DeviceExt = &DeviceExt{}
	if rw.Request == nil || rw.Request.Device == nil || rw.Request.Device.Ext == nil {
		return rw.DeviceExt.Unmarshal(json.RawMessage{})
	}
	return rw.DeviceExt.Unmarshal(rw.Request.Device.Ext)
}

func (rw *RequestWrapper) ExtractRequestExt() error {
	if rw.RequestExt != nil {
		return nil
	}
	rw.RequestExt = &RequestExt{}
	if rw.Request == nil || rw.Request.Ext == nil {
		return rw.RequestExt.Unmarshal(json.RawMessage{})
	}
	return rw.RequestExt.Unmarshal(rw.Request.Ext)
}

func (rw *RequestWrapper) ExtractAppExt() error {
	if rw.AppExt != nil {
		return nil
	}
	rw.AppExt = &AppExt{}
	if rw.Request == nil || rw.Request.App == nil || rw.Request.App.Ext == nil {
		return rw.AppExt.Unmarshal(json.RawMessage{})
	}
	return rw.AppExt.Unmarshal(rw.Request.App.Ext)
}

func (rw *RequestWrapper) ExtractRegExt() error {
	if rw.RegExt != nil {
		return nil
	}
	rw.RegExt = &RegExt{}
	if rw.Request == nil || rw.Request.Regs == nil || rw.Request.Regs.Ext == nil {
		return rw.RegExt.Unmarshal(json.RawMessage{})
	}
	return rw.RegExt.Unmarshal(rw.Request.Regs.Ext)
}

func (rw *RequestWrapper) ExtractSiteExt() error {
	if rw.SiteExt != nil {
		return nil
	}
	rw.SiteExt = &SiteExt{}
	if rw.Request == nil || rw.Request.Site == nil || rw.Request.Site.Ext == nil {
		return rw.SiteExt.Unmarshal(json.RawMessage{})
	}
	return rw.SiteExt.Unmarshal(rw.Request.Site.Ext)
}

func (rw *RequestWrapper) Sync() error {
	if rw.Request == nil {
		return fmt.Errorf("Requestwrapper Sync called on a nil Request")
	}

	if err := rw.syncUserExt(); err != nil {
		return err
	}

	if err := rw.syncDeviceExt(); err != nil {
		return err
	}

	if err := rw.syncRequestExt(); err != nil {
		return err
	}
	if err := rw.syncAppExt(); err != nil {
		return err
	}
	if err := rw.syncRegExt(); err != nil {
		return err
	}
	if err := rw.syncSiteExt(); err != nil {
		return err
	}

	return nil
}

func (rw *RequestWrapper) syncUserExt() error {
	if rw.Request.User == nil && rw.UserExt != nil && rw.UserExt.Dirty() {
		rw.Request.User = &openrtb2.User{}
	}
	if rw.UserExt != nil && rw.UserExt.Dirty() {
		userJson, err := rw.UserExt.Marshal()
		if err != nil {
			return err
		}
		rw.Request.User.Ext = userJson
	}
	return nil
}

func (rw *RequestWrapper) syncDeviceExt() error {
	if rw.Request.Device == nil && rw.DeviceExt != nil && rw.DeviceExt.Dirty() {
		rw.Request.Device = &openrtb2.Device{}
	}
	if rw.DeviceExt != nil && rw.DeviceExt.Dirty() {
		deviceJson, err := rw.DeviceExt.Marshal()
		if err != nil {
			return err
		}
		rw.Request.Device.Ext = deviceJson
	}
	return nil
}

func (rw *RequestWrapper) syncRequestExt() error {
	if rw.RequestExt != nil && rw.RequestExt.Dirty() {
		requestJson, err := rw.RequestExt.Marshal()
		if err != nil {
			return err
		}
		rw.Request.Ext = requestJson
	}
	return nil
}

func (rw *RequestWrapper) syncAppExt() error {
	if rw.Request.App == nil && rw.AppExt != nil && rw.AppExt.Dirty() {
		rw.Request.App = &openrtb2.App{}
	}
	if rw.AppExt != nil && rw.AppExt.Dirty() {
		appJson, err := rw.AppExt.Marshal()
		if err != nil {
			return err
		}
		rw.Request.App.Ext = appJson
	}
	return nil
}

func (rw *RequestWrapper) syncRegExt() error {
	if rw.Request.Regs == nil && rw.RegExt != nil && rw.RegExt.Dirty() {
		rw.Request.Regs = &openrtb2.Regs{}
	}
	if rw.RegExt != nil && rw.RegExt.Dirty() {
		regsJson, err := rw.RegExt.Marshal()
		if err != nil {
			return err
		}
		rw.Request.Regs.Ext = regsJson
	}
	return nil
}

func (rw *RequestWrapper) syncSiteExt() error {
	if rw.Request.Site == nil && rw.SiteExt != nil && rw.SiteExt.Dirty() {
		rw.Request.Site = &openrtb2.Site{}
	}
	if rw.SiteExt != nil && rw.SiteExt.Dirty() {
		siteJson, err := rw.SiteExt.Marshal()
		if err != nil {
			return err
		}
		rw.Request.Regs.Ext = siteJson
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

func (ue *UserExt) Unmarshal(extJson json.RawMessage) error {
	if len(ue.ext) != 0 || ue.Dirty() {
		return nil
	}
	ue.ext = make(map[string]json.RawMessage)
	ue.eids = &[]ExtUserEid{}
	if len(extJson) == 0 {
		return nil
	}
	err := json.Unmarshal(extJson, &ue.ext)
	if err != nil {
		return err
	}

	consentJson, hasConsent := ue.ext["consent"]
	if hasConsent {
		err = json.Unmarshal(consentJson, &ue.consent)
		if err != nil {
			return err
		}
	}

	prebidJson, hasPrebid := ue.ext["prebid"]
	if hasPrebid {
		ue.prebid = &ExtUserPrebid{}
		err = json.Unmarshal(prebidJson, ue.prebid)
		if err != nil {
			return err
		}
	}

	digiTrustJson, hasDigiTrust := ue.ext["digitrust"]
	if hasDigiTrust {
		ue.digiTrust = &ExtUserDigiTrust{}
		err = json.Unmarshal(digiTrustJson, ue.digiTrust)
		if err != nil {
			return err
		}
	}

	eidsJson, hasEids := ue.ext["eids"]
	if hasEids {
		err = json.Unmarshal(eidsJson, ue.eids)
		if err != nil {
			return err
		}
	}

	return err
}

func (ue *UserExt) Marshal() (json.RawMessage, error) {
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

func (re *RequestExt) Unmarshal(extJson json.RawMessage) error {
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

func (re *RequestExt) Marshal() (json.RawMessage, error) {
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

func (de *DeviceExt) Unmarshal(extJson json.RawMessage) error {
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

func (de *DeviceExt) Marshal() (json.RawMessage, error) {
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

func (ae *AppExt) Unmarshal(extJson json.RawMessage) error {
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

func (ae *AppExt) Marshal() (json.RawMessage, error) {
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

func (re *RegExt) Unmarshal(extJson json.RawMessage) error {
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

func (re *RegExt) Marshal() (json.RawMessage, error) {
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
	return re.uSPrivacyDirty
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
	Ext      map[string]json.RawMessage
	Amp      int8
	AmpDirty bool
}

func (se *SiteExt) Unmarshal(extJson json.RawMessage) error {
	if len(se.Ext) != 0 || se.Dirty() {
		return nil
	}
	se.Ext = make(map[string]json.RawMessage)
	if len(extJson) == 0 {
		return nil
	}
	err := json.Unmarshal(extJson, &se.Ext)
	if err != nil {
		return err
	}
	AmpJson, hasAmp := se.Ext["amp"]
	if hasAmp {
		err = json.Unmarshal(AmpJson, &se.Amp)
		// Replace with a more specific error message
		if err != nil {
			err = errors.New(`request.site.ext.amp must be either 1, 0, or undefined`)
		}
	}

	return err
}

func (se *SiteExt) Marshal() (json.RawMessage, error) {
	if se.AmpDirty {
		ampJson, err := json.Marshal(se.Amp)
		if err != nil {
			return nil, err
		}
		if len(ampJson) > 0 {
			se.Ext["amp"] = json.RawMessage(ampJson)
		} else {
			delete(se.Ext, "amp")
		}
		se.AmpDirty = false
	}

	rawJson, err := json.Marshal(se.Ext)
	if err == nil {
		se.AmpDirty = false
	}
	return rawJson, err
}

func (se *SiteExt) Dirty() bool {
	return se.AmpDirty
}
