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
	Cache     *ExtRequestPrebidCache `json:"cache"`
	Targeting *ExtRequestTargeting   `json:"targeting"`
}

// ExtRequestPrebidCache defines the contract for bidrequest.ext.prebid.cache
type ExtRequestPrebidCache struct {
	Bids *ExtRequestPrebidCacheBids `json:"bids"`
}

// UnmarhshalJSON prevents nil bids arguments.
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
	MaxLength        int              `json:"lengthmax"`
}

// ExtRequestTargeting without Unmashall override to prevent infinite loops
type ExtRequestTargetingPlain struct {
	PriceGranularity PriceGranularity `json:"pricegranularity"`
	MaxLength        int              `json:"lengthmax"`
}

// Make an unmashaller that will set a default PriceGranularity
func (ert *ExtRequestTargeting) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	ertRaw := &ExtRequestTargetingPlain{}
	err := json.Unmarshal(b, ertRaw)
	ert.PriceGranularity = ertRaw.PriceGranularity
	ert.MaxLength = ertRaw.MaxLength
	if err == nil {
		// set default value
		if ert.PriceGranularity == "" {
			ert.PriceGranularity = PriceGranularityMedium
		}
	}
	return err
}

// PriceGranularity defines the allowed values for bidrequest.ext.prebid.targeting.pricegranularity
type PriceGranularity string

const (
	PriceGranularityLow    PriceGranularity = "low"
	PriceGranularityMedium PriceGranularity = "medium"
	// Seems that PBS was written with medium = "med", so hacking that in
	PriceGranularityMedPBS PriceGranularity = "med"
	PriceGranularityHigh   PriceGranularity = "high"
	PriceGranularityAuto   PriceGranularity = "auto"
	PriceGranularityDense  PriceGranularity = "dense"
)
