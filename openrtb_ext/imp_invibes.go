package openrtb_ext

type ExtImpInvibes struct {
	PlacementId string             `json:"placementId,omitempty"`
	Debug       ExtImpInvibesDebug `json:"debug,omitempty"`
}

type ExtImpInvibesDebug struct {
	TestIp   string `json:"testIp,omitempty"`
	TestBvid string `json:"testBvid,omitempty"`
	TestAmp  string `json:"testAmp,omitempty"`
}
