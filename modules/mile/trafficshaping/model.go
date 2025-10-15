package trafficshaping

// TrafficShapingData represents the remote JSON configuration structure
type TrafficShapingData struct {
	Meta     Meta     `json:"meta"`
	Response Response `json:"response"`
}

// Meta contains metadata about the configuration
type Meta struct {
	CreatedAt int64 `json:"createdAt"`
}

// Response contains the shaping rules
type Response struct {
	Schema        Schema                               `json:"schema"`
	SkipRate      int                                  `json:"skipRate"`
	UserIdVendors []string                             `json:"userIdVendors"`
	Values        map[string]map[string]map[string]int `json:"values"` // gpid -> bidder -> size -> flag
}

// Schema describes the fields used in the config
type Schema struct {
	Fields []string `json:"fields"`
}

// ShapingConfig is the preprocessed config for fast lookup
type ShapingConfig struct {
	SkipRate      int
	UserIdVendors map[string]struct{}
	GPIDRules     map[string]*GPIDRule
}

// GPIDRule contains the allowed bidders and sizes for a specific GPID
type GPIDRule struct {
	AllowedBidders map[string]struct{}
	AllowedSizes   map[BannerSize]struct{}
}

// BannerSize represents a width x height banner size
type BannerSize struct {
	W int64
	H int64
}
