package openrtb_ext

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mxmCherry/openrtb/v14/openrtb2"
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
	rw.RegExt = &RegExt{}
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
	rw.SiteExt = &SiteExt{}
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
	Ext            map[string]json.RawMessage
	Consent        *string
	ConsentDirty   bool
	Prebid         *ExtUserPrebid
	PrebidDirty    bool
	DigiTrust      *ExtUserDigiTrust
	DigiTrustDirty bool
	Eids           *[]ExtUserEid
	EidsDirty      bool
}

func (ue *UserExt) Unmarshal(extJson json.RawMessage) error {
	ue.Ext = make(map[string]json.RawMessage)
	ue.Eids = &[]ExtUserEid{}
	if len(extJson) == 0 || len(ue.Ext) != 0 {
		return nil
	}
	err := json.Unmarshal(extJson, &ue.Ext)
	if err != nil {
		return err
	}

	consentJson, hasConsent := ue.Ext["consent"]
	if hasConsent {
		err = json.Unmarshal(consentJson, &ue.Consent)
		if err != nil {
			return err
		}
	}

	prebidJson, hasPrebid := ue.Ext["prebid"]
	if hasPrebid {
		ue.Prebid = &ExtUserPrebid{}
		err = json.Unmarshal(prebidJson, ue.Prebid)
		if err != nil {
			return err
		}
	}

	digiTrustJson, hasDigiTrust := ue.Ext["digitrust"]
	if hasDigiTrust {
		ue.DigiTrust = &ExtUserDigiTrust{}
		err = json.Unmarshal(digiTrustJson, ue.DigiTrust)
		if err != nil {
			return err
		}
	}

	eidsJson, hasEids := ue.Ext["eids"]
	ue.Eids = &[]ExtUserEid{}
	if hasEids {
		err = json.Unmarshal(eidsJson, ue.Eids)
		if err != nil {
			return err
		}
	}

	return err
}

func (ue *UserExt) Marshal() (json.RawMessage, error) {
	if ue.ConsentDirty {
		consentJson, err := json.Marshal(ue.Consent)
		if err != nil {
			return nil, err
		}
		if len(consentJson) > 0 {
			ue.Ext["consent"] = json.RawMessage(consentJson)
		} else {
			delete(ue.Ext, "consent")
		}
		ue.ConsentDirty = false
	}

	if ue.PrebidDirty {
		prebidJson, err := json.Marshal(ue.Prebid)
		if err != nil {
			return nil, err
		}
		if len(prebidJson) > 0 {
			ue.Ext["prebid"] = json.RawMessage(prebidJson)
		} else {
			delete(ue.Ext, "prebid")
		}
		ue.PrebidDirty = false
	}

	if ue.DigiTrustDirty {
		digiTrustJson, err := json.Marshal(ue.DigiTrust)
		if err != nil {
			return nil, err
		}
		if len(digiTrustJson) > 0 {
			ue.Ext["digitrust"] = json.RawMessage(digiTrustJson)
		} else {
			delete(ue.Ext, "digitrust")
		}
		ue.DigiTrustDirty = false
	}

	if ue.EidsDirty {
		if len(*ue.Eids) > 0 {
			eidsJson, err := json.Marshal(ue.Eids)
			if err != nil {
				return nil, err
			}
			ue.Ext["eids"] = json.RawMessage(eidsJson)
		} else {
			delete(ue.Ext, "eids")
		}
		ue.EidsDirty = false
	}

	return json.Marshal(ue.Ext)

}

func (ue *UserExt) Dirty() bool {
	return ue.DigiTrustDirty || ue.EidsDirty || ue.PrebidDirty || ue.ConsentDirty
}

// ---------------------------------------------------------------
// RequestExt provides an interface for request.ext
// ---------------------------------------------------------------

type RequestExt struct {
	Ext         map[string]json.RawMessage
	Prebid      *ExtRequestPrebid
	PrebidDirty bool
}

func (re *RequestExt) Unmarshal(extJson json.RawMessage) error {
	re.Ext = make(map[string]json.RawMessage)
	if len(extJson) == 0 || len(re.Ext) != 0 {
		return nil
	}
	err := json.Unmarshal(extJson, &re.Ext)
	if err != nil {
		return err
	}
	prebidJson, hasPrebid := re.Ext["prebid"]
	if hasPrebid {
		re.Prebid = &ExtRequestPrebid{}
		err = json.Unmarshal(prebidJson, re.Prebid)
	}

	return err
}

func (re *RequestExt) Marshal() (json.RawMessage, error) {
	if re.PrebidDirty {
		prebidJson, err := json.Marshal(re.Prebid)
		if err != nil {
			return nil, err
		}
		if len(prebidJson) > 2 {
			re.Ext["prebid"] = json.RawMessage(prebidJson)
		} else {
			delete(re.Ext, "prebid")
		}
		re.PrebidDirty = false
	}

	return json.Marshal(re.Ext)
}

func (re *RequestExt) Dirty() bool {
	return re.PrebidDirty
}

// ---------------------------------------------------------------
// DeviceExt provides an interface for request.device.ext
// ---------------------------------------------------------------
// NOTE: openrtb_ext/device.go:ParseDeviceExtATTS() uses ext.atts, as read only, via jsonparser, only for IOS.
// Doesn't seem like we will see any performance savings by parsing atts at this point, and as it is read only,
// we don't need to worry about write conflicts. Note here in case additional uses of atts evolve as things progress.
// ---------------------------------------------------------------

type DeviceExt struct {
	Ext         map[string]json.RawMessage
	Prebid      *ExtDevicePrebid
	PrebidDirty bool
}

func (de *DeviceExt) Unmarshal(extJson json.RawMessage) error {
	de.Ext = make(map[string]json.RawMessage)
	if len(extJson) == 0 || len(de.Ext) != 0 {
		return nil
	}
	err := json.Unmarshal(extJson, &de.Ext)
	if err != nil {
		return err
	}
	prebidJson, hasPrebid := de.Ext["prebid"]
	if hasPrebid {
		de.Prebid = &ExtDevicePrebid{}
		err = json.Unmarshal(prebidJson, de.Prebid)
	}

	return err
}

func (de *DeviceExt) Marshal() (json.RawMessage, error) {
	if de.PrebidDirty {
		prebidJson, err := json.Marshal(de.Prebid)
		if err != nil {
			return nil, err
		}
		if len(prebidJson) > 0 {
			de.Ext["prebid"] = json.RawMessage(prebidJson)
		} else {
			delete(de.Ext, "prebid")
		}
		de.PrebidDirty = false
	}

	rawJson, err := json.Marshal(de.Ext)
	if err == nil {
		de.PrebidDirty = false
	}
	return rawJson, err
}

func (de *DeviceExt) Dirty() bool {
	return de.PrebidDirty
}

// ---------------------------------------------------------------
// AppExt provides an interface for request.app.ext
// ---------------------------------------------------------------

type AppExt struct {
	Ext         map[string]json.RawMessage
	Prebid      *ExtAppPrebid
	PrebidDirty bool
}

func (ae *AppExt) Unmarshal(extJson json.RawMessage) error {
	ae.Ext = make(map[string]json.RawMessage)
	if len(extJson) == 0 || len(ae.Ext) != 0 {
		return nil
	}
	err := json.Unmarshal(extJson, &ae.Ext)
	if err != nil {
		return err
	}
	prebidJson, hasPrebid := ae.Ext["prebid"]
	if hasPrebid {
		ae.Prebid = &ExtAppPrebid{}
		err = json.Unmarshal(prebidJson, ae.Prebid)
	}

	return err
}

func (ae *AppExt) Marshal() (json.RawMessage, error) {
	if ae.PrebidDirty {
		prebidJson, err := json.Marshal(ae.Prebid)
		if err != nil {
			return nil, err
		}
		if len(prebidJson) > 0 {
			ae.Ext["prebid"] = json.RawMessage(prebidJson)
		} else {
			delete(ae.Ext, "prebid")
		}
	}

	rawJson, err := json.Marshal(ae.Ext)
	if err == nil {
		ae.PrebidDirty = false
	}
	return rawJson, err
}

func (ae *AppExt) Dirty() bool {
	return ae.PrebidDirty
}

// ---------------------------------------------------------------
// RegExt provides an interface for request.regs.ext
// ---------------------------------------------------------------

type RegExt struct {
	Ext            map[string]json.RawMessage
	USPrivacy      string
	USPrivacyDirty bool
}

func (re *RegExt) Unmarshal(extJson json.RawMessage) error {
	re.Ext = make(map[string]json.RawMessage)
	if len(extJson) == 0 || len(re.Ext) != 0 {
		return nil
	}
	err := json.Unmarshal(extJson, &re.Ext)
	if err != nil {
		return err
	}
	uspJson, hasUsp := re.Ext["us_privacy"]
	if hasUsp {
		err = json.Unmarshal(uspJson, &re.USPrivacy)
	}

	return err
}

func (re *RegExt) Marshal() (json.RawMessage, error) {
	if re.USPrivacyDirty {
		if len(re.USPrivacy) > 0 {
			rawjson, err := json.Marshal(re.USPrivacy)
			if err != nil {
				return nil, err
			}
			re.Ext["us_privacy"] = rawjson
		} else {
			delete(re.Ext, "us_privacy")
		}
	}
	if len(re.Ext) == 0 {
		re.USPrivacyDirty = false
		return nil, nil
	}

	rawJson, err := json.Marshal(re.Ext)
	if err == nil {
		re.USPrivacyDirty = false
	}
	return rawJson, err
}

func (re *RegExt) Dirty() bool {
	return re.USPrivacyDirty
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
	se.Ext = make(map[string]json.RawMessage)
	if len(extJson) == 0 || len(se.Ext) != 0 {
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
