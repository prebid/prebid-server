package openrtb_ext

// ExtImpFilteringSubCategory - Impression Filtering SubCategory Extension
type ExtImpFilteringSubCategory struct {
	Name  string   `json:"name,omitempty"`
	Value []string `json:"value,omitempty"`
}

// ExtImpPreferred - Impression Preferred Extension
type ExtImpPreferred struct {
	ProductID string  `json:"pid,omitempty"`
	Rating    float64 `json:"rating,omitempty"`
}

// ExtImpFiltering - Impression Filtering Extension
type ExtImpFiltering struct {
	Category    []string                      `json:"category,omitempty"`
	Brand       []string                      `json:"brand,omitempty"`
	SubCategory []*ExtImpFilteringSubCategory `json:"subcategory,omitempty"`
}

// ExtImpTargeting - Impression Targeting Extension
type ExtImpTargeting struct {
	Name  string `json:"name,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

type ExtCustomConfig struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
	Type  int    `json:"type,omitempty"`
}

// ImpExtensionCommerce - Impression Commerce Extension
type CommerceParams struct {
	SlotsRequested int                `json:"slots_requested,omitempty"`
	TestRequest    bool               `json:"test_request,omitempty"`
	SearchTerm     string             `json:"search_term,omitempty"`
	SearchType     string             `json:"search_type,omitempty"`
	Preferred      []*ExtImpPreferred `json:"preferred,omitempty"`
	Filtering      *ExtImpFiltering   `json:"filtering,omitempty"`
	Targeting      []*ExtImpTargeting `json:"targeting,omitempty"`
}

// ImpExtensionCommerce - Impression Commerce Extension
type ExtImpCommerce struct {
	ComParams *CommerceParams `json:"commerce,omitempty"`
	Bidder *ExtBidderCommerce          `json:"bidder,omitempty"`
}
// UserExtensionCommerce - User Commerce Extension
type ExtUserCommerce struct {
	IsAuthenticated bool    `json:"is_authenticated,omitempty"`
	Consent         string `json:"consent,omitempty"`
}

// SiteExtensionCommerce - Site Commerce Extension
type ExtSiteCommerce struct {
	Page string `json:"page_name,omitempty"`
}

// AppExtensionCommerce - App Commerce Extension
type ExtAppCommerce struct {
	Page string `json:"page_name,omitempty"`
}

type ExtBidderCommerce struct {
	PrebidBidderName string             `json:"prebidname,omitempty"`
	BidderCode       string             `json:"biddercode,omitempty"`
	CustomConfig     []*ExtCustomConfig `json:"config,omitempty"`
}

type ExtBidCommerce struct {
	ProductId        string               `json:"productid,omitempty"`
	ClickUrl         string               `json:"curl,omitempty"`
	ConversionUrl    string               `json:"purl,omitempty"`
	ClickPrice       float64              `json:"clickprice,omitempty"`
	Rate             float64              `json:"rate,omitempty"`
	ProductDetails   map[string]interface{}  `json:"productdetails,omitempty"`
}

