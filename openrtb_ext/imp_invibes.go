package openrtb_ext

type ExtImpInvibes struct {
	PlacementId string             `json:"placementId,omitempty"`
	Host        string             `json:"host"`
	Debug       ExtImpInvibesDebug `json:"debug,omitempty"`
}

type ExtImpInvibesDebug struct {
	TestIp   string `json:"testIp,omitempty"`
	TestBvid string `json:"testBvid,omitempty"`
	TestLog  bool   `json:"testLog,omitempty"`
}
