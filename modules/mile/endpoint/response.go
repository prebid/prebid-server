package endpoint

import (
	"encoding/json"

	"github.com/prebid/openrtb/v20/openrtb2"
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

// transformToMileResponse picks the highest CPM bid per impression and maps it to MileResponse.
func transformToMileResponse(br *openrtb2.BidResponse) MileResponse {
	if br == nil {
		return MileResponse{Bids: []MileBid{}}
	}

	if len(br.SeatBid) == 0 {
		return MileResponse{Bids: []MileBid{}, Ext: br.Ext}
	}

	type bidKey struct {
		impID string
	}

	// Track best bid per impression
	best := make(map[bidKey]struct {
		bid  *openrtb2.Bid
		seat string
	})

	for _, sb := range br.SeatBid {
		for i := range sb.Bid {
			b := &sb.Bid[i]
			if b.Price <= 0 || b.ImpID == "" {
				continue
			}
			key := bidKey{impID: b.ImpID}
			prev, ok := best[key]
			if !ok || b.Price > prev.bid.Price {
				best[key] = struct {
					bid  *openrtb2.Bid
					seat string
				}{bid: b, seat: sb.Seat}
			}
		}
	}

	resp := MileResponse{Bids: make([]MileBid, 0, len(best)), Ext: br.Ext}
	for _, v := range best {
		b := v.bid
		resp.Bids = append(resp.Bids, MileBid{
			RequestID:  b.ImpID,
			CPM:        b.Price,
			Currency:   br.Cur,
			Width:      b.W,
			Height:     b.H,
			Ad:         b.AdM,
			TTL:        fallbackTTL(b.Exp),
			CreativeID: b.CrID,
			NetRevenue: true,
			Bidder:     v.seat,
			MediaType:  inferMediaType(b),
		})
	}

	return resp
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
