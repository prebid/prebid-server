package openrtb_ext

import (
	"encoding/json"
	"errors"

	"github.com/mxmCherry/openrtb"
)

// FirstPartyDataContextExtKey defines the field name within bidrequest.ext reserved
// for first party data support.
const FirstPartyDataContextExtKey string = "context"
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
	Request *openrtb.BidRequest
	// Dirty bool // Probably don't care
	UserExt    *UserExt
	DeviceExt  *DeviceExt
	RequestExt *RequestExt
	SiteExt    *SiteExt
}

type UserExt struct {
	Ext         map[string]json.RawMessage
	Dirty       bool
	Prebid      *ExtUserPrebid
	PrebidDirty bool
	Sync        *ExtUserSync
	SyncDirty   bool
}

func (ue *UserExt) Extract(extJson json.RawMessage) (*UserExt, error) {
	newUE := &UserExt{}
	err := newUE.Unmarshal(extJson)
	return newUE, err
}

func (ue *UserExt) Unmarshal(extJson json.RawMessage) error {
	if len(extJson) == 0 || len(ue.Ext) != 0 {
		return nil
	}
	err := json.Unmarshal(extJson, ue.Ext)
	if err != nil {
		return err
	}
	prebidJson, hasPrebid := ue.Ext["prebid"]
	if hasPrebid {
		ue.Prebid = &ExtUserPrebid{}
		err = json.Unmarshal(prebidJson, ue.Prebid)
	}

	return nil
}

func (ue *UserExt) Marshal() (json.RawMessage, error) {
	if ue.PrebidDirty {
		prebidJson, err := json.Marshal(ue.Prebid)
		if err != nil {
			return nil, err
		}
		ue.Ext["prebid"] = json.RawMessage(prebidJson)
	}

	// Device

	return json.Marshal(ue.Ext)

}

type RequestExt struct {
	Ext         map[string]json.RawMessage
	Dirty       bool
	Prebid      *ExtRequestPrebid
	PrebidDirty bool
}

func (re *RequestExt) Extract(extJson json.RawMessage) (*RequestExt, error) {
	newRE := &RequestExt{}
	err := newRE.Unmarshal(extJson)
	return newRE, err
}

func (re *RequestExt) Unmarshal(extJson json.RawMessage) error {
	if len(extJson) == 0 || len(re.Ext) != 0 {
		return nil
	}
	err := json.Unmarshal(extJson, re.Ext)
	if err != nil {
		return err
	}
	prebidJson, hasPrebid := re.Ext["prebid"]
	if hasPrebid {
		re.Prebid = &ExtRequestPrebid{}
		err = json.Unmarshal(prebidJson, re.Prebid)
	}

	return nil
}

func (re *RequestExt) Marshal() (json.RawMessage, error) {
	if re.PrebidDirty {
		prebidJson, err := json.Marshal(re.Prebid)
		if err != nil {
			return nil, err
		}
		re.Ext["prebid"] = json.RawMessage(prebidJson)
	}

	// Device

	return json.Marshal(re.Ext)

}

type DeviceExt struct {
	Ext         map[string]json.RawMessage
	Dirty       bool
	Prebid      *ExtDevicePrebid
	PrebidDirty bool
}

func (de *DeviceExt) Extract(extJson json.RawMessage) (*DeviceExt, error) {
	newDE := &DeviceExt{}
	err := newDE.Unmarshal(extJson)
	return newDE, err
}

func (de *DeviceExt) Unmarshal(extJson json.RawMessage) error {
	if len(extJson) == 0 || len(de.Ext) != 0 {
		return nil
	}
	err := json.Unmarshal(extJson, de.Ext)
	if err != nil {
		return err
	}
	prebidJson, hasPrebid := de.Ext["prebid"]
	if hasPrebid {
		de.Prebid = &ExtDevicePrebid{}
		err = json.Unmarshal(prebidJson, de.Prebid)
	}

	return nil
}

func (de *DeviceExt) Marshal() (json.RawMessage, error) {
	if de.PrebidDirty {
		prebidJson, err := json.Marshal(de.Prebid)
		if err != nil {
			return nil, err
		}
		de.Ext["prebid"] = json.RawMessage(prebidJson)
	}

	// Device

	return json.Marshal(de.Ext)

}

type SiteExt struct {
	Ext ExtSite
}

func (se *SiteExt) Extract(extJson json.RawMessage) (*SiteExt, error) {
	newSE := &SiteExt{}
	err := newSE.Unmarshal(extJson)
	return newSE, err
}

func (se *SiteExt) Unmarshal(extJson json.RawMessage) error {
	if len(extJson) == 0 || len(se.Ext) != 0 {
		return nil
	}
	err := json.Unmarshal(extJson, de.Ext)
	if err != nil {
		return err
	}
	return nil
}

func (se *SiteExt) Marshal() (json.RawMessage, error) {
	return json.Marshal(se.Ext)
}

func (rw *RequestWrapper) ExtractUserExt() error {
	if rw.UserExt != nil || rw.Request.User.Ext == nil {
		return nil
	}
	var err error
	rw.UserExt, err = rw.UserExt.Extract(rw.Request.User.Ext)
	return err
}

func (rw *RequestWrapper) ExtractDeviceExt() error {
	if rw.DeviceExt != nil || rw.Request.Device.Ext == nil {
		return nil
	}
	var err error
	rw.DeviceExt, err = rw.DeviceExt.Extract(rw.Request.Device.Ext)
	return err
}

func (rw *RequestWrapper) ExtractRequestExt() error {
	if rw.RequestExt != nil || rw.Request.Ext == nil {
		return nil
	}
	var err error
	rw.RequestExt, err = rw.RequestExt.Extract(rw.Request.Ext)
	return err
}

func (rw *RequestWrapper) ExtractSiteExt() error {
	if rw.SiteExt != nil || rw.Request.Site.Ext == nil {
		return nil
	}
	var err error
	rw.SiteExt, err = rw.SiteExt.Extract(rw.Request.Site.Ext)
	return err
}

func (rw *RequestWrapper) Marshal() (json.RawMessage, error) {
	if rw.UserExt.Dirty {
		userJson, err := rw.UserExt.Marshal()
		if err != nil {
			return nil, err
		}
		rw.Request.User.Ext = userJson
	}
	if rw.DeviceExt.Dirty {
		deviceJson, err := rw.DeviceExt.Marshal()
		if err != nil {
			return nil, err
		}
		rw.Request.Device.Ext = deviceJson
	}
	return json.Marshal(rw.Request)
}
