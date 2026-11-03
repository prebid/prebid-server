package openrtb_ext

type ExtImpFeedAd struct {
	ClientToken string                  `json:"clientToken"`
	Decoration  string                  `json:"decoration"`
	PlacementId string                  `json:"placementId"`
	SdkOptions  *ExtImpFeedAdSdkOptions `json:"sdkOptions"`
}

type ExtImpFeedAdSdkOptions struct {
	AdvertisingId   string `json:"advertising_id"`
	AppName         string `json:"app_name"`
	BundleId        string `json:"bundle_id"`
	HybridApp       bool   `json:"hybrid_app"`
	HybridPlatform  string `json:"hybrid_platform"`
	LimitAdTracking bool   `json:"limit_ad_tracking"`
}
