package openrtb_ext

// ExtImpMoloco defines the contract for bidrequest.imp[i].ext.moloco
type ExtImpMoloco struct {
	PlacementType  string `json:"placementtype"`
	Region         string `json:"region"`          // this field added to support multiple moloco endpoints
	SKADNSupported bool   `json:"skadn_supported"` // enable skadn ext parameters
	MRAIDSupported bool   `json:"mraid_supported"`
}
