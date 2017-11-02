package openrtb_ext

// ExtRequest defines the contract for bidrequest.ext
type ExtRequest struct {
	Prebid ExtRequestPrebid `json:"prebid"`
}

// ExtRequestPrebid defines the contract for bidrequest.ext.prebid
type ExtRequestPrebid struct {
	Cache *ExtRequestCache        `json:"cache"`
	Targeting *ExtRequestTargeting`json:"targeting"`
}

// ExtRequestCache defines the contract for bidrequest.ext.prebid.cache
type ExtRequestCache struct {
	Markup CacheMarkup `json:"markup"`
}

// ExtRequestTargeting defines the contract for bidrequest.ext.prebid.targeting
type ExtRequestTargeting struct {
	PriceGranularity PriceGranularity `json:"pricegranularity"`
	MaxLength        int              `json:"lengthmax"`
}

// CacheMarkup defines the allowed values for bidrequest.ext.prebid.cache.markup
type CacheMarkup int

const (
	No  CacheMarkup = 0
	Yes CacheMarkup = 1
)

// SortType defines the allowed values for bidrequest.ext.sort.type
type SortType string

const (
	None SortType = "none"
	Cpm  SortType = "cpm"
)

// PriceGranularity defines the allowed values for bidrequest.ext.targeting.pricegranularity
type PriceGranularity string

const (
	PriceGranularityLow    PriceGranularity = "low"
	PriceGranularityMedium PriceGranularity = "med"
	PriceGranularityHigh   PriceGranularity = "high"
	PriceGranularityAuto   PriceGranularity = "auto"
	PriceGranularityDense  PriceGranularity = "dense"
)
