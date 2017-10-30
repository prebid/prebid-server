package openrtb_ext

// ExtRequest defines the contract for bidrequest.ext
type ExtRequest struct {
	Prebid ExtSeatBidPrebid `json:"prebid"`
}

// ExtRequestPrebid defines the contract for bidrequest.ext.prebid
type ExtRequestPrebid struct {
	Cache *ExtRequestCache `json:"cache,omitempty"`
}

// ExtRequestCache defines the contract for bidrequest.ext.prebid.cache
type ExtRequestCache struct {
	Markup CacheMarkup `json:"markup"`
}

// ExtRequestSort defines the contract for bidrequest.ext.prebid.sort
type ExtRequestSort struct {
	Type SortType `json:"type"`
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
	Yes             = 1
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
	PriceGranularityMedium                  = "med"
	PriceGranularityHigh                    = "high"
	PriceGranularityAuto                    = "auto"
	PriceGranularityDense                   = "dense"
)
