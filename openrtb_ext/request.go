package openrtb_ext

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mxmCherry/openrtb/v14/openrtb2"
)

// FirstPartyDataExtKey defines a field name within request.ext and request.imp.ext reserved for first party data.
const FirstPartyDataExtKey = "data"

// FirstPartyDataContextExtKey defines a field name within request.ext and request.imp.ext reserved for first party data.
const FirstPartyDataContextExtKey = "context"

// SKAdNExtKey defines the field name within request.ext reserved for Apple's SKAdNetwork.
const SKAdNExtKey = "skadn"

const MaxDecimalFigures int = 15

// ExtRequest defines the contract for bidrequest.ext
type ExtRequest struct {
	Prebid ExtRequestPrebid `json:"prebid"`
}

// ExtRequestPrebid defines the contract for bidrequest.ext.prebid
type ExtRequestPrebid struct {
	Aliases              map[string]string         `json:"aliases,omitempty"`
	BidAdjustmentFactors map[string]float64        `json:"bidadjustmentfactors,omitempty"`
	Cache                *ExtRequestPrebidCache    `json:"cache,omitempty"`
	Data                 *ExtRequestPrebidData     `json:"data,omitempty"`
	Debug                bool                      `json:"debug,omitempty"`
	Events               json.RawMessage           `json:"events,omitempty"`
	SChains              []*ExtRequestPrebidSChain `json:"schains,omitempty"`
	StoredRequest        *ExtStoredRequest         `json:"storedrequest,omitempty"`
	SupportDeals         bool                      `json:"supportdeals,omitempty"`
	Targeting            *ExtRequestTargeting      `json:"targeting,omitempty"`

	// NoSale specifies bidders with whom the publisher has a legal relationship where the
	// passing of personally identifiable information doesn't constitute a sale per CCPA law.
	// The array may contain a single sstar ('*') entry to represent all bidders.
	NoSale []string `json:"nosale,omitempty"`
}

// ExtRequestPrebid defines the contract for bidrequest.ext.prebid.schains
type ExtRequestPrebidSChain struct {
	Bidders []string                     `json:"bidders,omitempty"`
	SChain  ExtRequestPrebidSChainSChain `json:"schain"`
}

// ExtRequestPrebidSChainSChain defines the contract for bidrequest.ext.prebid.schains[i].schain
type ExtRequestPrebidSChainSChain struct {
	Complete int                                 `json:"complete"`
	Nodes    []*ExtRequestPrebidSChainSChainNode `json:"nodes"`
	Ver      string                              `json:"ver"`
	Ext      json.RawMessage                     `json:"ext,omitempty"`
}

// ExtRequestPrebidSChainSChainNode defines the contract for bidrequest.ext.prebid.schains[i].schain[i].nodes
type ExtRequestPrebidSChainSChainNode struct {
	ASI    string          `json:"asi"`
	SID    string          `json:"sid"`
	RID    string          `json:"rid,omitempty"`
	Name   string          `json:"name,omitempty"`
	Domain string          `json:"domain,omitempty"`
	HP     int             `json:"hp"`
	Ext    json.RawMessage `json:"ext,omitempty"`
}

// SourceExt defines the contract for bidrequest.source.ext
type SourceExt struct {
	SChain ExtRequestPrebidSChainSChain `json:"schain"`
}

// ExtRequestPrebidCache defines the contract for bidrequest.ext.prebid.cache
type ExtRequestPrebidCache struct {
	Bids    *ExtRequestPrebidCacheBids `json:"bids"`
	VastXML *ExtRequestPrebidCacheVAST `json:"vastxml"`
}

// UnmarshalJSON prevents nil bids arguments.
func (ert *ExtRequestPrebidCache) UnmarshalJSON(b []byte) error {
	type typesAlias ExtRequestPrebidCache // Prevents infinite UnmarshalJSON loops
	var proxy typesAlias
	if err := json.Unmarshal(b, &proxy); err != nil {
		return err
	}

	if proxy.Bids == nil && proxy.VastXML == nil {
		return errors.New(`request.ext.prebid.cache requires one of the "bids" or "vastxml" properties`)
	}

	*ert = ExtRequestPrebidCache(proxy)
	return nil
}

// ExtRequestPrebidCacheBids defines the contract for bidrequest.ext.prebid.cache.bids
type ExtRequestPrebidCacheBids struct {
	ReturnCreative *bool `json:"returnCreative"`
}

// ExtRequestPrebidCacheVAST defines the contract for bidrequest.ext.prebid.cache.vastxml
type ExtRequestPrebidCacheVAST struct {
	ReturnCreative *bool `json:"returnCreative"`
}

// ExtRequestTargeting defines the contract for bidrequest.ext.prebid.targeting
type ExtRequestTargeting struct {
	PriceGranularity     PriceGranularity         `json:"pricegranularity"`
	IncludeWinners       bool                     `json:"includewinners"`
	IncludeBidderKeys    bool                     `json:"includebidderkeys"`
	IncludeBrandCategory *ExtIncludeBrandCategory `json:"includebrandcategory"`
	IncludeFormat        bool                     `json:"includeformat"`
	DurationRangeSec     []int                    `json:"durationrangesec"`
	PreferDeals          bool                     `json:"preferdeals"`
	AppendBidderNames    bool                     `json:"appendbiddernames,omitempty"`
}

type ExtIncludeBrandCategory struct {
	PrimaryAdServer     int    `json:"primaryadserver"`
	Publisher           string `json:"publisher"`
	WithCategory        bool   `json:"withcategory"`
	TranslateCategories *bool  `json:"translatecategories,omitempty"`
}

// Make an unmarshaller that will set a default PriceGranularity
func (ert *ExtRequestTargeting) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}

	// define separate type to prevent infinite recursive calls to UnmarshalJSON
	type extRequestTargetingDefaults ExtRequestTargeting
	defaults := &extRequestTargetingDefaults{
		PriceGranularity:  priceGranularityMed,
		IncludeWinners:    true,
		IncludeBidderKeys: true,
	}

	err := json.Unmarshal(b, defaults)
	if err == nil {
		if !defaults.IncludeWinners && !defaults.IncludeBidderKeys {
			return errors.New("ext.prebid.targeting: At least one of includewinners or includebidderkeys must be enabled to enable targeting support")
		}
		*ert = ExtRequestTargeting(*defaults)
	}

	return err
}

// PriceGranularity defines the allowed values for bidrequest.ext.prebid.targeting.pricegranularity
type PriceGranularity struct {
	Precision int                `json:"precision,omitempty"`
	Ranges    []GranularityRange `json:"ranges,omitempty"`
}

type PriceGranularityRaw PriceGranularity

// GranularityRange struct defines a range of prices used by PriceGranularity
type GranularityRange struct {
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
	Increment float64 `json:"increment"`
}

// UnmarshalJSON : custom unmarshaller to handle legacy string granularites.
func (pg *PriceGranularity) UnmarshalJSON(b []byte) error {
	// We default to medium
	if len(b) == 0 {
		*pg = priceGranularityMed
		return nil
	}
	// First check for legacy strings
	var pgString string
	err := json.Unmarshal(b, &pgString)
	if err == nil {
		*pg = PriceGranularityFromString(pgString)
		if len(pg.Ranges) > 0 {
			// Only exit if we matched something, else we try processing as custom granularity
			// This way we error as expecting the new custom granularity standard.
			return nil
		}
	}
	// Not legacy, so we do a normal Unmarshal
	pgraw := PriceGranularityRaw{}
	pgraw.Precision = 2
	err = json.Unmarshal(b, &pgraw)
	if err != nil {
		return err
	}
	if pgraw.Precision < 0 {
		return errors.New("Price granularity error: precision must be non-negative")
	}
	if pgraw.Precision > MaxDecimalFigures {
		return errors.New("Price granularity error: precision of more than 15 significant figures is not supported")
	}
	if len(pgraw.Ranges) > 0 {
		var prevMax float64 = 0
		for i, gr := range pgraw.Ranges {
			if gr.Max <= prevMax {
				return errors.New("Price granularity error: range list must be ordered with increasing \"max\"")
			}
			if gr.Increment <= 0.0 {
				return errors.New("Price granularity error: increment must be a nonzero positive number")
			}
			// Enforce that we don't read "min" from the request
			pgraw.Ranges[i].Min = prevMax
			prevMax = gr.Max
		}
		*pg = PriceGranularity(pgraw)
		return nil
	}
	// Default to medium if no ranges are specified
	*pg = priceGranularityMed
	return nil
}

// PriceGranularityFromString converts a legacy string into the new PriceGranularity
func PriceGranularityFromString(gran string) PriceGranularity {
	switch gran {
	case "low":
		return priceGranularityLow
	case "med", "medium":
		// Seems that PBS was written with medium = "med", so hacking that in
		return priceGranularityMed
	case "high":
		return priceGranularityHigh
	case "auto":
		return priceGranularityAuto
	case "dense":
		return priceGranularityDense
	}
	// Return empty if not matched
	return PriceGranularity{}
}

var priceGranularityLow = PriceGranularity{
	Precision: 2,
	Ranges: []GranularityRange{{
		Min:       0,
		Max:       5,
		Increment: 0.5}},
}

var priceGranularityMed = PriceGranularity{
	Precision: 2,
	Ranges: []GranularityRange{{
		Min:       0,
		Max:       20,
		Increment: 0.1}},
}

var priceGranularityHigh = PriceGranularity{
	Precision: 2,
	Ranges: []GranularityRange{{
		Min:       0,
		Max:       20,
		Increment: 0.01}},
}

var priceGranularityDense = PriceGranularity{
	Precision: 2,
	Ranges: []GranularityRange{
		{
			Min:       0,
			Max:       3,
			Increment: 0.01,
		},
		{
			Min:       3,
			Max:       8,
			Increment: 0.05,
		},
		{
			Min:       8,
			Max:       20,
			Increment: 0.5,
		},
	},
}

var priceGranularityAuto = PriceGranularity{
	Precision: 2,
	Ranges: []GranularityRange{
		{
			Min:       0,
			Max:       5,
			Increment: 0.05,
		},
		{
			Min:       5,
			Max:       10,
			Increment: 0.1,
		},
		{
			Min:       10,
			Max:       20,
			Increment: 0.5,
		},
	},
}

// ExtRequestPrebidData defines Prebid's First Party Data (FPD) and related bid request options.
type ExtRequestPrebidData struct {
	EidPermissions []ExtRequestPrebidDataEidPermission `json:"eidpermissions"`
}

// ExtRequestPrebidDataEidPermission defines a filter rule for filter user.ext.eids
type ExtRequestPrebidDataEidPermission struct {
	Source  string   `json:"source"`
	Bidders []string `json:"bidders"`
}

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

func (ue *UserExt) Extract(extJson json.RawMessage) (*UserExt, error) {
	newUE := &UserExt{}
	newUE.Ext = make(map[string]json.RawMessage)
	newUE.Eids = &[]ExtUserEid{}
	err := newUE.Unmarshal(extJson)
	return newUE, err
}

func (ue *UserExt) Unmarshal(extJson json.RawMessage) error {
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

type RequestExt struct {
	Ext         map[string]json.RawMessage
	Prebid      *ExtRequestPrebid
	PrebidDirty bool
}

func (re *RequestExt) Extract(extJson json.RawMessage) (*RequestExt, error) {
	newRE := &RequestExt{}
	newRE.Ext = make(map[string]json.RawMessage)
	err := newRE.Unmarshal(extJson)
	return newRE, err
}

func (re *RequestExt) Unmarshal(extJson json.RawMessage) error {
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

// NOTE: openrtb_ext/device.go:ParseDeviceExtATTS() uses ext.atts, as read only, via jsonparser, only for IOS.
// Doesn't seem like we will see any performance savings by parsing atts at this point, and as it is read only,
// we don't need to worry about write conflicts. Note here in case additional uses of atts evolve as things progress.
type DeviceExt struct {
	Ext         map[string]json.RawMessage
	Prebid      *ExtDevicePrebid
	PrebidDirty bool
}

func (de *DeviceExt) Extract(extJson json.RawMessage) (*DeviceExt, error) {
	newDE := &DeviceExt{}
	newDE.Ext = make(map[string]json.RawMessage)
	err := newDE.Unmarshal(extJson)
	return newDE, err
}

func (de *DeviceExt) Unmarshal(extJson json.RawMessage) error {
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

type AppExt struct {
	Ext         map[string]json.RawMessage
	Prebid      *ExtAppPrebid
	PrebidDirty bool
}

func (ae *AppExt) Extract(extJson json.RawMessage) (*AppExt, error) {
	newAE := &AppExt{}
	newAE.Ext = make(map[string]json.RawMessage)
	err := newAE.Unmarshal(extJson)
	return newAE, err
}

func (ae *AppExt) Unmarshal(extJson json.RawMessage) error {
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

type RegExt struct {
	Ext            map[string]json.RawMessage
	USPrivacy      string
	USPrivacyDirty bool
}

func (re *RegExt) Extract(extJson json.RawMessage) (*RegExt, error) {
	newRE := &RegExt{}
	newRE.Ext = make(map[string]json.RawMessage)
	err := newRE.Unmarshal(extJson)
	return newRE, err
}

func (re *RegExt) Unmarshal(extJson json.RawMessage) error {
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

type SiteExt struct {
	Ext      map[string]json.RawMessage
	Amp      int8
	AmpDirty bool
}

func (se *SiteExt) Extract(extJson json.RawMessage) (*SiteExt, error) {
	newSE := &SiteExt{}
	newSE.Ext = make(map[string]json.RawMessage)
	err := newSE.Unmarshal(extJson)
	return newSE, err
}

func (se *SiteExt) Unmarshal(extJson json.RawMessage) error {
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

func (rw *RequestWrapper) ExtractUserExt() error {
	if rw.UserExt != nil {
		return nil
	}
	if rw.Request == nil || rw.Request.User == nil || rw.Request.User.Ext == nil {
		rw.UserExt = &UserExt{}
		rw.UserExt.Ext = make(map[string]json.RawMessage)
		rw.UserExt.Eids = &[]ExtUserEid{}
		return nil
	}

	var err error
	rw.UserExt, err = rw.UserExt.Extract(rw.Request.User.Ext)
	return err
}

func (rw *RequestWrapper) ExtractDeviceExt() error {
	if rw.DeviceExt != nil {
		return nil
	}
	if rw.Request == nil || rw.Request.Device == nil || rw.Request.Device.Ext == nil {
		rw.DeviceExt = &DeviceExt{}
		rw.DeviceExt.Ext = make(map[string]json.RawMessage)
		return nil
	}
	var err error
	rw.DeviceExt, err = rw.DeviceExt.Extract(rw.Request.Device.Ext)
	return err
}

func (rw *RequestWrapper) ExtractRequestExt() error {
	if rw.RequestExt != nil {
		return nil
	}
	if rw.Request == nil || rw.Request.Ext == nil {
		rw.RequestExt = &RequestExt{}
		rw.RequestExt.Ext = make(map[string]json.RawMessage)
		return nil
	}
	var err error
	rw.RequestExt, err = rw.RequestExt.Extract(rw.Request.Ext)
	return err
}

func (rw *RequestWrapper) ExtractAppExt() error {
	if rw.AppExt != nil {
		return nil
	}
	if rw.Request == nil || rw.Request.App == nil || rw.Request.App.Ext == nil {
		rw.AppExt = &AppExt{}
		rw.AppExt.Ext = make(map[string]json.RawMessage)
		return nil
	}
	var err error
	rw.AppExt, err = rw.AppExt.Extract(rw.Request.App.Ext)
	return err
}

func (rw *RequestWrapper) ExtractRegExt() error {
	if rw.RegExt != nil {
		return nil
	}
	if rw.Request == nil || rw.Request.Regs == nil || rw.Request.Regs.Ext == nil {
		rw.RegExt = &RegExt{}
		rw.RegExt.Ext = make(map[string]json.RawMessage)
		return nil
	}
	var err error
	rw.RegExt, err = rw.RegExt.Extract(rw.Request.Regs.Ext)
	return err
}

func (rw *RequestWrapper) ExtractSiteExt() error {
	if rw.SiteExt != nil {
		return nil
	}
	if rw.Request == nil || rw.Request.Site == nil || rw.Request.Site.Ext == nil {
		rw.SiteExt = &SiteExt{}
		rw.SiteExt.Ext = make(map[string]json.RawMessage)
		return nil
	}
	var err error
	rw.SiteExt, err = rw.SiteExt.Extract(rw.Request.Site.Ext)
	return err
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
