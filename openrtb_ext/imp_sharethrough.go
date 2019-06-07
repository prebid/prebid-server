package openrtb_ext

import "encoding/json"

type ExtImpSharethrough struct {
	PlacementKey string `json:"pkey"`
	Iframe       bool   `json:"iframe"`
}

// ExtImpSharethrough defines the contract for bidrequest.imp[i].ext.sharethrough
type ExtImpSharethroughResponse struct {
	AdServerRequestID string                       `json:"adserverRequestId"`
	BidID             string                       `json:"bidId"`
	CookieSyncUrls    []string                     `json:"cookieSyncUrls"`
	Creatives         []ExtImpSharethroughCreative `json:"creatives"`
	Placement         ExtImpSharethroughPlacement  `json:"placement"`
	StxUserID         string                       `json:"stxUserId"`
}
type ExtImpSharethroughCreative struct {
	AuctionWinID string                             `json:"auctionWinId"`
	CPM          float64                            `json:"cpm"`
	Metadata     ExtImpSharethroughCreativeMetadata `json:"creative"`
	Version      int                                `json:"version"`
}

type ExtImpSharethroughCreativeMetadata struct {
	Action                 string                            `json:"action"`
	Advertiser             string                            `json:"advertiser"`
	AdvertiserKey          string                            `json:"advertiser_key"`
	Beacons                ExtImpSharethroughCreativeBeacons `json:"beacons"`
	BrandLogoURL           string                            `json:"brand_logo_url"`
	CampaignKey            string                            `json:"campaign_key"`
	CreativeKey            string                            `json:"creative_key"`
	CustomEngagementAction string                            `json:"custom_engagement_action"`
	CustomEngagementLabel  string                            `json:"custom_engagement_label"`
	CustomEngagementURL    string                            `json:"custom_engagement_url"`
	DealID                 string                            `json:"deal_id"`
	Description            string                            `json:"description"`
	ForceClickToPlay       bool                              `json:"force_click_to_play"`
	IconURL                string                            `json:"icon_url"`
	ImpressionHTML         string                            `json:"impression_html"`
	InstantPlayMobileCount int                               `json:"instant_play_mobile_count"`
	InstantPlayMobileURL   string                            `json:"instant_play_mobile_url"`
	MediaURL               string                            `json:"media_url"`
	ShareURL               string                            `json:"share_url"`
	SourceID               string                            `json:"source_id"`
	ThumbnailURL           string                            `json:"thumbnail_url"`
	Title                  string                            `json:"title"`
	VariantKey             string                            `json:"variant_key"`
}

type ExtImpSharethroughCreativeBeacons struct {
	Click           []string `json:"click"`
	Impression      []string `json:"impression"`
	Play            []string `json:"play"`
	Visible         []string `json:"visible"`
	WinNotification []string `json:"win-notification"`
}

type ExtImpSharethroughPlacement struct {
	AllowInstantPlay      bool                                  `json:"allow_instant_play"`
	ArticlesBeforeFirstAd int                                   `json:"articles_before_first_ad"`
	ArticlesBetweenAds    int                                   `json:"articles_between_ads"`
	Layout                string                                `json:"layout"`
	Metadata              json.RawMessage                       `json:"metadata"`
	PlacementAttributes   ExtImpSharethroughPlacementAttributes `json:"placementAttributes"`
	Status                string                                `json:"status"`
}

type ExtImpSharethroughPlacementThirdPartyPartner struct {
	Key string `json:"key"`
	Tag string `json:"tag"`
}

type ExtImpSharethroughPlacementAttributes struct {
	AdServerKey              string                                         `json:"ad_server_key"`
	AdServerPath             string                                         `json:"ad_server_path"`
	AllowDynamicCropping     bool                                           `json:"allow_dynamic_cropping"`
	AppThirdPartyPartners    []string                                       `json:"app_third_party_partners"`
	CustomCardCSS            string                                         `json:"custom_card_css"`
	DFPPath                  string                                         `json:"dfp_path"`
	DirectSellPromotedByText string                                         `json:"direct_sell_promoted_by_text"`
	Domain                   string                                         `json:"domain"`
	EnableLinkRedirection    bool                                           `json:"enable_link_redirection"`
	FeaturedContent          json.RawMessage                                `json:"featured_content"`
	MaxHeadlineLength        int                                            `json:"max_headline_length"`
	MultiAdPlacement         bool                                           `json:"multi_ad_placement"`
	PromotedByText           string                                         `json:"promoted_by_text"`
	PublisherKey             string                                         `json:"publisher_key"`
	RenderingPixelOffset     int                                            `json:"rendering_pixel_offset"`
	SafeFrameSize            []int                                          `json:"safe_frame_size"`
	SiteKey                  string                                         `json:"site_key"`
	StrOptOutURL             string                                         `json:"str_opt_out_url"`
	Template                 string                                         `json:"template"`
	ThirdPartyPartners       []ExtImpSharethroughPlacementThirdPartyPartner `json:"third_party_partners"`
}

type ExtImpSharethroughExt struct {
	Pkey       string `json:"pkey"`
	Iframe     bool   `json:"iframe"`
	IframeSize []int  `json:"iframeSize"`
}
