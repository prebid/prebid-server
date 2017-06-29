package pbs

// PBSBid is a bid from the auction. These are produced by Adapters, and target a particular Ad Unit.
type PBSBid struct {
	// BidID identifies the Bid Request within the Ad Unit which this Bid targets. It should match one of
	// the values inside PBSRequest.AdUnits[i].Bids[j].BidID.
	BidID string `json:"bid_id"`
	// AdUnitCode identifies the AdUnit which this Bid targets.
	// It should match one of PBSRequest.AdUnits[i].Code, where "i" matches the AdUnit used in
	// as BidID.
	AdUnitCode  string `json:"code"`
	Creative_id string `json:"creative_id,omitempty"`
	BidderCode  string `json:"bidder"`
	// BidHash is the hash of the bidder's unique bid identifier for blockchain. It should not be sent to browser.
	BidHash string `json:"-"`
	// Price is the cpm, in US Dollars, which the bidder is willing to pay if this bid is chosen.
	// TODO: Add support for other currencies someday.
	Price float64 `json:"price"`
	NURL  string  `json:"nurl,omitempty"`
	// Adm is the payload which should be used to deliver the ad, if this bid is chosen.
	Adm string `json:"adm,omitempty"`
	// Width is the intended width which Adm should be shown, in pixels.
	Width uint64 `json:"width,omitempty"`
	// Height is the intended width which Adm should be shown, in pixels.
	Height            uint64            `json:"height,omitempty"`
	DealId            string            `json:"deal_id,omitempty"`
	CacheID           string            `json:"cache_id,omitempty"`
	ResponseTime      int               `json:"response_time_ms,omitempty"`
	AdServerTargeting map[string]string `json:"ad_server_targeting,omitempty"`
}

// PBSBidSlice attaches the methods of sort.Interface to []PBSBid, ordering them by response times.
// For more information, see https://golang.org/pkg/sort/#Interface
type PBSBidSlice []*PBSBid

func (bids PBSBidSlice) Len() int {
	return len(bids)
}

func (bids PBSBidSlice) Less(i, j int) bool {
	bidiResponseTimeInNanos := (float64(bids[i].ResponseTime) / 1000000000.0)
	bidjResponseTimeInNanos := (float64(bids[j].ResponseTime) / 1000000000.0)
	return bids[i].Price-bidiResponseTimeInNanos < bids[j].Price-bidjResponseTimeInNanos
}

func (bids PBSBidSlice) Swap(i, j int) {
	bids[i], bids[j] = bids[j], bids[i]
}

type BidderDebug struct {
	RequestURI   string `json:"request_uri,omitempty"`
	RequestBody  string `json:"request_body,omitempty"`
	ResponseBody string `json:"response_body,omitempty"`
	StatusCode   int    `json:"status_code,omitempty"`
}

type UsersyncInfo struct {
	URL         string `json:"url,omitempty"`
	Type        string `json:"type,omitempty"`
	SupportCORS bool   `json:"supportCORS,omitempty"`
}

type PBSResponse struct {
	TID          string       `json:"tid,omitempty"`
	Status       string       `json:"status,omitempty"`
	BidderStatus []*PBSBidder `json:"bidder_status,omitempty"`
	Bids         PBSBidSlice  `json:"bids,omitempty"`
	BUrl         string       `json:"burl,omitempty"`
}
