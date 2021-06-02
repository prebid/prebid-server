package openrtb_ext

import (
	"encoding/json"
	"errors"
)

// FirstPartyDataExtKey defines a field name within request.ext and request.imp.ext reserved for first party data.
const FirstPartyDataExtKey = "data"

// FirstPartyDataContextExtKey defines a field name within request.ext and request.imp.ext reserved for first party data.
const FirstPartyDataContextExtKey = "context"

// SKAdNExtKey defines the field name within request.ext reserved for Apple's SKAdNetwork.
const SKAdNExtKey = "skadn"

// NativeExchangeSpecificLowerBound defines the lower threshold of exchange specific types for native ads. There is no upper bound.
const NativeExchangeSpecificLowerBound = 500

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

	CurrencyConversions *ExtRequestCurrency `json:"currency,omitempty"`
}

type ExtRequestCurrency struct {
	ConversionRates map[string]map[string]float64 `json:"rates"`
	UsePBSRates     *bool                         `json:"usepbsrates"`
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
