package endpoint

import (
	"encoding/json"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// MileResponse is the bidder-style response expected by the Prebid.js adapter.
type MileResponse struct {
	Bids []MileBid       `json:"bids"`
	Ext  json.RawMessage `json:"ext,omitempty"`
}

// MileBid represents a single winning bid in Prebid.js adapter format.
type MileBid struct {
	RequestID  string  `json:"requestId"`
	CPM        float64 `json:"cpm"`
	Currency   string  `json:"currency"`
	Width      int64   `json:"width"`
	Height     int64   `json:"height"`
	Ad         string  `json:"ad"`
	TTL        int     `json:"ttl"`
	CreativeID string  `json:"creativeId"`
	NetRevenue bool    `json:"netRevenue"`
	Bidder     string  `json:"bidder"`
	MediaType  string  `json:"mediaType"`
}

// transformToMileResponse identifies the winning bid per impression using PBS built-in targeting keys and maps it to MileResponse.
func transformToMileResponse(br *openrtb2.BidResponse) MileResponse {
	if br == nil {
		return MileResponse{Bids: []MileBid{}}
	}

	if len(br.SeatBid) == 0 {
		return MileResponse{Bids: []MileBid{}, Ext: br.Ext}
	}

	winners := make([]MileBid, 0)

	for _, sb := range br.SeatBid {
		for i := range sb.Bid {
			b := &sb.Bid[i]
			if b.Price <= 0 || b.ImpID == "" {
				continue
			}

			// Check if this bid is a built-in winner by looking for targeting keys
			if isWinner(b) {
				winners = append(winners, MileBid{
					RequestID:  b.ImpID,
					CPM:        b.Price,
					Currency:   br.Cur,
					Width:      b.W,
					Height:     b.H,
					Ad:         b.AdM,
					TTL:        fallbackTTL(b.Exp),
					CreativeID: b.CrID,
					NetRevenue: true,
					Bidder:     sb.Seat,
					MediaType:  inferMediaType(b),
				})
			}
		}
	}

	return MileResponse{Bids: winners, Ext: br.Ext}
}

func isWinner(bid *openrtb2.Bid) bool {
	if bid.Ext == nil {
		return false
	}

	var bidExt openrtb_ext.ExtBid
	if err := json.Unmarshal(bid.Ext, &bidExt); err != nil {
		return false
	}

	if bidExt.Prebid == nil || bidExt.Prebid.Targeting == nil {
		return false
	}

	// PBS Core adds targeting keys like "hb_bidder" for the overall winner of an impression.
	// The key is formed by prefix + targetingKey. Default is "hb_bidder".
	// We check for the presence of the bidder key without a bidder suffix.
	for key := range bidExt.Prebid.Targeting {
		// Default prefix is "hb". TargetingKey for bidder is "_bidder".
		// So we look for "hb_bidder".
		if key == "hb_bidder" {
			return true
		}
	}

	return false
}

func fallbackTTL(exp int64) int {
	if exp <= 0 {
		return 300
	}
	return int(exp)
}

func inferMediaType(b *openrtb2.Bid) string {
	if b == nil {
		return "banner"
	}
	// Basic inference; extend if native/video present in bid ext if needed
	if b.H != 0 && b.W != 0 {
		return "banner"
	}
	return "banner"
}
