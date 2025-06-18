package openrtb_ext

type ExtImpRiseMediaTech struct {
	BidFloor    *float64 `json:"bidfloor,omitempty"`
	Mimes       []string `json:"mimes,omitempty"`
	MinDuration int64    `json:"minduration,omitempty"`
	MaxDuration int64    `json:"maxduration,omitempty"`
	StartDelay  int64    `json:"startdelay,omitempty"`
	MaxSeq      int64    `json:"maxseq,omitempty"`
	PodDur      int64    `json:"poddur,omitempty"`
	Protocols   []int64  `json:"protocols,omitempty"`
}
