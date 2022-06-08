package openrtb_ext

type ExtImpSpotX struct {
	ChannelID  string  `json:"channel_id"`
	AdUnit     string  `json:"ad_unit"`
	Secure     bool    `json:"secure,omitempty"`
	AdVolume   float64 `json:"ad_volume,omitempty"`
	PriceFloor int     `json:"price_floor,omitempty"`
	HideSkin   bool    `json:"hide_skin,omitempty"`
}
