package openrtb_ext

type ExtImpInvibes struct {
	PlacementID string             `json:"placementId,omitempty"`
	DomainID    int                `json:"domainId"`
	Debug       ExtImpInvibesDebug `json:"debug,omitempty"`
}

type ExtImpInvibesDebug struct {
	TestBvid string `json:"testBvid,omitempty"`
	TestLog  bool   `json:"testLog,omitempty"`
}
