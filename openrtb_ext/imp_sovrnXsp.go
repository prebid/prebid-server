package openrtb_ext

type ExtImpSovrnXsp struct {
	PubID    string `json:"pub_id,omitempty"`
	MedID    string `json:"med_id,omitempty"`
	ZoneID   string `json:"zone_id,omitempty"`
	ForceBid bool   `json:"force_bid,omitempty"`
}
