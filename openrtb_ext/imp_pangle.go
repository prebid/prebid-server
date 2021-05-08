package openrtb_ext

type ImpExtPangle struct {
	Token       string `json:"token"`
	AppID       string `json:"appid,omitempty"`
	PlacementID string `json:"placementid,omitempty"`
}
