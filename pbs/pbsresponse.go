package pbs

// PBSBid is a bid from the auction. These are produced by Adapters, and target a particular Ad Unit.
//
// This JSON format is a contract with both Prebid.js and Prebid-mobile.
// All changes *must* be backwards compatible, since clients cannot be forced to update their code.
type PBSBid struct {
	// BidID identifies the Bid Request within the Ad Unit which this Bid targets. It should match one of
	// the values inside PBSRequest.AdUnits[i].Bids[j].BidID.
	BidID string `json:"bid_id"`
	// AdUnitCode identifies the AdUnit which this Bid targets.
	// It should match one of PBSRequest.AdUnits[i].Code, where "i" matches the AdUnit used in
	// as BidID.
	AdUnitCode string `json:"code"`
	// Creative_id uniquely identifies the creative being served. It is not used by prebid-server, but
	// it helps publishers and bidders identify and communicate about malicious or inappropriate ads.
	// This project simply passes it along with the bid.
	Creative_id string `json:"creative_id,omitempty"`
	// CreativeMediaType shows whether the creative is a video or banner.
	CreativeMediaType string `json:"media_type,omitempty"`
	// BidderCode is the PBSBidder.BidderCode of the PBSBidder who made this bid.
	BidderCode string `json:"bidder"`
	// BidHash is the hash of the bidder's unique bid identifier for blockchain. It should not be sent to browser.
	BidHash string `json:"-"`
	// Price is the cpm, in US Dollars, which the bidder is willing to pay if this bid is chosen.
	// TODO: Add support for other currencies someday.
	Price float64 `json:"price"`
	// NURL is a URL which returns ad markup, and should be called if the bid wins.
	// If NURL and Adm are both defined, then Adm takes precedence.
	NURL string `json:"nurl,omitempty"`
	// Adm is the ad markup which should be used to deliver the ad, if this bid is chosen.
	// If NURL and Adm are both defined, then Adm takes precedence.
	Adm string `json:"adm,omitempty"`
	// Width is the intended width which Adm should be shown, in pixels.
	Width uint64 `json:"width,omitempty"`
	// Height is the intended width which Adm should be shown, in pixels.
	Height uint64 `json:"height,omitempty"`
	// DealId is not used by prebid-server, but may be used by buyers and sellers who make special
	// deals with each other. We simply pass this information along with the bid.
	DealId string `json:"deal_id,omitempty"`
	// CacheId is an ID in prebid-cache which can be used to fetch this ad's content.
	// This supports prebid-mobile, which requires that the content be available from a URL.
	CacheID string `json:"cache_id,omitempty"`
	// ResponseTime is the number of milliseconds it took for the adapter to return a bid.
	ResponseTime      int               `json:"response_time_ms,omitempty"`
	AdServerTargeting map[string]string `json:"ad_server_targeting,omitempty"`
}

// PBSBidSlice attaches the methods of sort.Interface to []PBSBid, ordering them by price.
// If two prices are equal, then the response time will be used as a tiebreaker.
// For more information, see https://golang.org/pkg/sort/#Interface
type PBSBidSlice []*PBSBid

func (bids PBSBidSlice) Len() int {
	return len(bids)
}

func (bids PBSBidSlice) Less(i, j int) bool {
	bidiResponseTimeInTerras := (float64(bids[i].ResponseTime) / 1000000000.0)
	bidjResponseTimeInTerras := (float64(bids[j].ResponseTime) / 1000000000.0)
	return bids[i].Price-bidiResponseTimeInTerras > bids[j].Price-bidjResponseTimeInTerras
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
