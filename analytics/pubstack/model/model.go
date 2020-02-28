package model

type BidRequest struct {
	BidderCode string   `json:"bidderCode,omitempty"`
	State      string   `json:"state,omitempty"`
	Cpm        float32  `json:"cpm,omitempty"`
	Size       string   `json:"size,omitempty"`
	Elapsed    uint32   `json:"elapsed,omitempty"`
	BidId      string   `json:"bidId,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

type Auction struct {
	ScopeId string `json:"scopeId,omitempty"`
	TagId   string `json:"tagId,omitempty"`
	AdUnit  string `json:"adUnit,omitempty"`
	Device  string `json:"device,omitempty"`
	Country string `json:"country,omitempty"`
	Domain  string `json:"domain,omitempty"`
	// deduplication an join
	AuctionId string `json:"auctionId,omitempty"`
	// bid objects
	BidRequests []BidRequest `json:"bidRequests,omitempty"`
	// indicate the iteration of the refresh
	Refresh bool `json:"refresh,omitempty"`
	// ex [ "refresh:3" ]
	Tags []string `json:"tags,omitempty"`
	// debug fields
	BrowserName    string `json:"browserName,omitempty"`
	BrowserVersion string `json:"browserVersion,omitempty"`
	OsName         string `json:"osName,omitempty"`
	OsVersion      string `json:"osVersion,omitempty"`
	// extended model fields
	Href  string   `json:"href,omitempty"`
	Sizes []string `json:"sizes,omitempty"`
}
