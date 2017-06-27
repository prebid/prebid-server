package pbs

type PBSBid struct {
	BidID             string            `json:"bid_id"`
	AdUnitCode        string            `json:"code"`
	Creative_id       string            `json:"creative_id,omitempty"`
	BidderCode        string            `json:"bidder"`
	BidHash           string            `json:"-"` // this is the hash of the bidder's unique bid identifier for blockchain. Should not be sent to browser.
	Price             float64           `json:"price"`
	Currency          string            `json:"currency,omitempty"`
	NURL              string            `json:"nurl,omitempty"`
	Adm               string            `json:"adm,omitempty"`
	Width             uint64            `json:"width,omitempty"`
	Height            uint64            `json:"height,omitempty"`
	DealId            string            `json:"deal_id,omitempty"`
	CacheID           string            `json:"cache_id,omitempty"`
	ResponseTime      int               `json:"response_time_ms,omitempty"`
	AdServerTargeting map[string]string `json:"ad_server_targeting,omitempty"`
}

type PBSBidSlice []*PBSBid

// Implement sort.Interface
func (bids PBSBidSlice) Len() int {
	return len(bids)
}

// Less sorts from lowest to highest, to reverse we want to check which is greater bids[i].Price > bids[j].Price
func (bids PBSBidSlice) Less(i, j int) bool {
	bidiResponseTimeInNanos := (float64(bids[i].ResponseTime) / 1000000000.0)
	bidjResponseTimeInNanos := (float64(bids[j].ResponseTime) / 1000000000.0)
	return bids[i].Price-bidiResponseTimeInNanos > bids[j].Price-bidjResponseTimeInNanos
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
