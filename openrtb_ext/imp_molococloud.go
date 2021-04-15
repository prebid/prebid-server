package openrtb_ext

// ExtImpMolocoCloud defines the contract for bidrequest.imp[i].ext.molococloud
type ExtImpMolocoCloud struct {
	PlacementType  string `json:"placementtype"`
	Region         string `json:"region"`
	SKADNSupported bool   `json:"skadn_supported"`
	MRAIDSupported bool   `json:"mraid_supported"`
}
