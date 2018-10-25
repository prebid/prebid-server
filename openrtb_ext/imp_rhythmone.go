package openrtb_ext

// ExtImpRhythmone defines the contract for bidrequest.imp[i].ext.rhythmone
type ExtImpRhythmone struct {
	PlacementId string `json:"placementId"`
	Zone        string `json:"zone"`
	Path        string `json:"path"`
	S2S         bool
}
