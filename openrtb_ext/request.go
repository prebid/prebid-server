package openrtb_ext

// ExtRequest defines the contract for bidrequest.ext
type ExtRequest struct {
	Prebid ExtRequestPrebid `json:"prebid"`
}

// ExtRequestPrebid defines the contract for bidrequest.ext.prebid
type ExtRequestPrebid struct {
	Targeting *ExtRequestTargeting`json:"targeting"`
}

// ExtRequestTargeting defines the contract for bidrequest.ext.prebid.targeting
type ExtRequestTargeting struct {
	PriceGranularity PriceGranularity `json:"pricegranularity"`
	MaxLength        int              `json:"lengthmax"`
}

// PriceGranularity defines the allowed values for bidrequest.ext.prebid.targeting.pricegranularity
type PriceGranularity string

const (
	PriceGranularityLow    PriceGranularity = "low"
	PriceGranularityMedium PriceGranularity = "med"
	PriceGranularityHigh   PriceGranularity = "high"
	PriceGranularityAuto   PriceGranularity = "auto"
	PriceGranularityDense  PriceGranularity = "dense"
)
