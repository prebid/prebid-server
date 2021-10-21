package openrtb_ext

// ExtImpTapjoy defines the contract for bidrequest.imp[i].ext.tapjoy
type ExtImpTapjoy struct {
	Device     TJDevice     `json:"device"`
	Extensions TJExtensions `json:"extensions"`

	Region         string `json:"region"`
	SKADNSupported bool   `json:"skadn_supported"`
	MRAIDSupported bool   `json:"mraid_supported"`
}

type TJDevice struct {
	OS         string `json:"os"`
	OSV        string `json:"osv"`
	HWV        string `json:"hwv"`
	Make       string `json:"make"`
	Model      string `json:"model"`
	DeviceType int8   `json:"device_type"`
}

type TJExtensions struct {
	AppExt       TJAppExt       `json:"app_ext"`
	ImpExt       TJImpExt       `json:"imp_ext"`
	RegsExt      TJRegsExt      `json:"regs_ext"`
	UserExt      TJUserExt      `json:"user_ext"`
	VideoExt     TJVideoExt     `json:"video_ext"`
	DeviceExt    TJDeviceExt    `json:"device_ext"`
	SourceExt    TJSourceExt    `json:"source_ext"`
	RequestExt   TJRequestExt   `json:"request_ext"`
	PublisherExt TJPublisherExt `json:"publisher_ext"`
}

type TJRequestExt struct {
	AdViewID string `json:"ad_view_id"`
}

type TJAppExt struct {
	ID       string     `json:"app_id"`
	Currency TJCurrency `json:"currency"`
}

type TJPublisherExt struct {
	MarginRate  *float64 `json:"margin_rate,omitempty"`
	PublisherID string   `json:"pub_id"`
}

type TJImpExt struct {
	ActionID    string `json:"action_id"`
	SKAdNetwork TJSKAN `json:"skadn,omitempty"`
}

type TJVideoExt struct {
	Rewarded int `json:"rewarded,omitempty"`
}

type TJRegsExt struct {
	GDPR      int    `json:"gdpr,omitempty"`
	USPrivacy string `json:"us_privacy,omitempty"`
}

type TJUserExt struct {
	Consent string          `json:"consent"`
	PubUser TJPublisherUser `json:"pub_user,omitempty"`
}

type TJDeviceExt struct {
	ATTS                  int               `json:"atts"`
	Name                  string            `json:"name"`
	UDID                  string            `json:"udid"`
	Volume                float64           `json:"volume"`
	AndroidID             string            `json:"android_id"`
	ParsedApp             map[string]string `json:"parsed_app"`
	VendorIDs             map[string]string `json:"vendor_ids"`
	MacAddress            string            `json:"mac_address"`
	DeviceModel           TJDeviceModel     `json:"device_model"`
	IsJailbroken          int               `json:"is_jailbroken"`
	AdvertisingID         string            `json:"advertising_id"`
	IsAdminDevice         int               `json:"is_admin_device"`
	AppsSdkVersions       map[string]string `json:"apps_sdk_versions"`
	OptOutOfferTypes      []string          `json:"opt_out_offer_types"`
	ScreenLayoutSize      string            `json:"screen_layout_size"`
	MobileCarrierCode     string            `json:"mobile_carrier_code"`
	IsAdvertisingIDDevice int               `json:"is_advertising_id_device"`
}

type TJPublisherUser struct {
	ID      string        `json:"id"`
	Devices []TJDeviceExt `json:"devices,omitempty"`
}

type TJSourceExt struct {
	OMIDPN   string `json:"omidpn,omitempty"`
	OMIDPV   string `json:"omidpv,omitempty"`
	Mediator string `json:"mediator"`
}

type TJDeviceModel struct {
	Model        string `json:"model,omitempty"`
	Category     string `json:"category,omitempty"`
	Manufacturer string `json:"manufacturer,omitempty"`
}

type TJSKAN struct {
	SKANIDs   []string `json:"skadnetids,omitempty"`
	Version   string   `json:"version"`
	SourceApp string   `json:"sourceapp,omitempty"`
}

type TJCurrency struct {
	ID                          string          `json:"id"`
	AppID                       string          `json:"app_id"`
	Rewarded                    int             `json:"rewarded"`
	AllowList                   []string        `json:"allowlist"`
	PartnerID                   string          `json:"partner_id"`
	DirectPlay                  int             `json:"direct_play"`
	MarginRate                  *float64        `json:"margin_rate"`
	AppsNetwork                 []TJAppsNetwork `json:"apps_networks"`
	OfferFilter                 string          `json:"offer_filter"`
	MatureRating                uint            `json:"mature_rating"`
	MaxAgeRating                uint            `json:"max_age_rating"`
	UseAllowlist                int             `json:"use_allowlist"`
	OnlyFreeOffers              int             `json:"only_free_offers"`
	DisableOfferIDs             []string        `json:"disabled_offer_ids"`
	MinimumDisplayBid           int             `json:"minimum_display_bid"`
	DisabledPartnerIDs          []string        `json:"disabled_partner_ids"`
	MinimumFeaturedBid          int             `json:"minimum_featured_bid"`
	MinimumOfferwallBid         int             `json:"minimum_offerwall_bid"`
	MinimumDisplayBidExponent   int             `json:"minimum_display_bid_exponent"`
	MinimumFeaturedBidExponent  int             `json:"minimum_featured_bid_exponent"`
	MinimumOfferwallBidExponent int             `json:"minimum_offerwall_bid_exponent"`
}

type TJAppsNetwork struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	BlockListOfferIDs    []string `json:"blocklist_offer_ids"`
	BlockListCategories  []string `json:"blocklist_categories"`
	BlockListOfferTitles []string `json:"blocklist_offer_titles"`
}
