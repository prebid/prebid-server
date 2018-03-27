package openrtb_ext

import (
	"encoding/json"
	"errors"
)

// ExtRequest defines the contract for bidrequest.ext
type ExtRequest struct {
	Prebid ExtRequestPrebid `json:"prebid"`
}

// ExtRequestPrebid defines the contract for bidrequest.ext.prebid
type ExtRequestPrebid struct {
	Aliases       map[string]string      `json:"aliases,omitempty"`
	Cache         *ExtRequestPrebidCache `json:"cache,omitempty"`
	StoredRequest *ExtStoredRequest      `json:"storedrequest,omitempty"`
	Targeting     *ExtRequestTargeting   `json:"targeting,omitempty"`
}

// ExtRequestPrebidCache defines the contract for bidrequest.ext.prebid.cache
type ExtRequestPrebidCache struct {
	Bids *ExtRequestPrebidCacheBids `json:"bids"`
}

// UnmarshalJSON prevents nil bids arguments.
func (ert *ExtRequestPrebidCache) UnmarshalJSON(b []byte) error {
	type typesAlias ExtRequestPrebidCache // Prevents infinite UnmarshalJSON loops
	var proxy typesAlias
	if err := json.Unmarshal(b, &proxy); err != nil {
		return err
	}

	if proxy.Bids == nil {
		return errors.New(`request.ext.prebid.cache missing required property "bids"`)
	}

	*ert = ExtRequestPrebidCache(proxy)
	return nil
}

// ExtRequestPrebidCacheBids defines the contract for bidrequest.ext.prebid.cache.bids
type ExtRequestPrebidCacheBids struct{}

// ExtRequestTargeting defines the contract for bidrequest.ext.prebid.targeting
type ExtRequestTargeting struct {
	PriceGranularity PriceGranularity `json:"pricegranularity"`
	IncludeWinners   bool             `json:"includewinners"`
}

// Make an unmarshaller that will set a default PriceGranularity
func (ert *ExtRequestTargeting) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}

	// define seperate type to prevent infinite recursive calls to UnmarshalJSON
	type extRequestTargetingDefaults ExtRequestTargeting
	defaults := &extRequestTargetingDefaults{
		PriceGranularity: priceGranularityMed,
		IncludeWinners:   true,
	}

	err := json.Unmarshal(b, defaults)
	if err == nil {
		*ert = ExtRequestTargeting(*defaults)
	}

	return err
}

// PriceGranularity defines the allowed values for bidrequest.ext.prebid.targeting.pricegranularity
type PriceGranularity []GranularityRange

type granularityRangeRaw struct {
	Precision *int    `json:"precision,omitempty"`
	Min       float64 `json:"min,omitempty"`
	Max       float64 `json:"max"`
	Increment float64 `json:"increment"`
}

// GranularityRange struct defines a range of prices used by PriceGranularity
type GranularityRange struct {
	Precision int
	Min       float64
	Max       float64
	Increment float64
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
		if len(*pg) > 0 {
			// Only exit if we matched something, else we try processing as custom granularity
			// This way we error as expecting the new custom granularity standard.
			return nil
		}
	}
	// Not legacy, so we do a normal Unmarshal
	gran := []GranularityRange{}
	err = json.Unmarshal(b, &gran)
	if err != nil {
		return err
	}
	if len(gran) > 1 {
		// We only need to loop over the ranges if we have more than one (inter-range checks)
		precision := gran[0].Precision
		var prevMax float64 = 0
		for _, gr := range gran {
			if gr.Precision != precision {
				return errors.New("Price granularity error: precision not consistent across entires")
			}
			if gr.Max < prevMax {
				return errors.New("Price granularity error: range list must be ordered with increasing \"max\"")
			}
			if gr.Min < prevMax {
				return errors.New("Price granularity error: overlapping granularity ranges")
			}
			if gr.Min == 0.0 {
				// Default min to be the previous max
				// On the first entry, we will likely overwrite 0.0 with 0.0, which should be ok. Adding a conditional to skip likely won't save any processing time.
				gr.Min = prevMax
			}
			prevMax = gr.Max
		}
	} else if len(gran) == 0 {
		return errors.New("Price granularity error: empty granularity definition supplied")
	}
	*pg = gran
	return nil
}

// UnmarshalJSON : custom unmarshaller to handle validation and default precision
func (gr *GranularityRange) UnmarshalJSON(b []byte) error {
	two := 2
	grr := granularityRangeRaw{Precision: &two}
	err := json.Unmarshal(b, &grr)
	if err != nil {
		return err
	}
	if grr.Precision != nil {
		gr.Precision = *grr.Precision
	}
	gr.Min = grr.Min
	gr.Max = grr.Max
	gr.Increment = grr.Increment
	if gr.Max < gr.Min {
		return errors.New("Price granularity error: max must be greater than min")
	}
	if gr.Increment <= 0.0 {
		return errors.New("Price granularity error: increment must be a nonzero positive number")
	}
	if gr.Precision < 0 {
		return errors.New("Price granularity error: precision must be non-negative")
	}
	return nil
}

var priceGranulrityLow = PriceGranularity{
	{
		Precision: 2,
		Min:       0,
		Max:       5,
		Increment: 0.5,
	},
}

// PriceGranularityFromString converts a legacy string into the new PriceGranularity
func PriceGranularityFromString(gran string) PriceGranularity {
	switch gran {
	case "low":
		return priceGranulrityLow
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

var priceGranularityMed = PriceGranularity{
	{
		Precision: 2,
		Min:       0,
		Max:       20,
		Increment: 0.1,
	},
}

var priceGranularityHigh = PriceGranularity{
	{
		Precision: 2,
		Min:       0,
		Max:       20,
		Increment: 0.01,
	},
}

var priceGranularityDense = PriceGranularity{
	{
		Precision: 2,
		Min:       0,
		Max:       3,
		Increment: 0.01,
	},
	{
		Precision: 2,
		Min:       3,
		Max:       8,
		Increment: 0.05,
	},
	{
		Precision: 2,
		Min:       8,
		Max:       20,
		Increment: 0.5,
	},
}

var priceGranularityAuto = PriceGranularity{
	{
		Precision: 2,
		Min:       0,
		Max:       5,
		Increment: 0.05,
	},
	{
		Precision: 2,
		Min:       5,
		Max:       10,
		Increment: 0.1,
	},
	{
		Precision: 2,
		Min:       10,
		Max:       20,
		Increment: 0.5,
	},
}
