package bidselect

import (
	"sort"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	vast "github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment"
)

// PriceSelector selects bids based on price-based ranking.
// It implements the vast.BidSelector interface.
type PriceSelector struct {
	// maxBids is the maximum number of bids to return.
	// If 0, uses cfg.MaxAdsInPod from the config.
	maxBids int
}

// NewPriceSelector creates a new PriceSelector.
// If maxBids is 0, the selector will use cfg.MaxAdsInPod.
// If maxBids is 1, it behaves as a SINGLE selector.
func NewPriceSelector(maxBids int) *PriceSelector {
	return &PriceSelector{
		maxBids: maxBids,
	}
}

// bidWithSeat holds a bid along with its seat ID for sorting and selection.
type bidWithSeat struct {
	bid  openrtb2.Bid
	seat string
}

// Select chooses bids from the response based on price-based ranking.
// It implements the vast.BidSelector interface.
//
// Selection process:
// 1. Collect all bids from resp.SeatBid[].Bid[]
// 2. Filter bids: price > 0 and AdM non-empty (unless AllowSkeletonVast is true)
// 3. Sort by: price desc, then deal exists desc, then bid.ID asc for stability
// 4. Return up to maxBids (or cfg.MaxAdsInPod if maxBids is 0)
// 5. Populate CanonicalMeta for each SelectedBid
func (s *PriceSelector) Select(req *openrtb2.BidRequest, resp *openrtb2.BidResponse, cfg vast.ReceiverConfig) ([]vast.SelectedBid, []string, error) {
	var warnings []string

	if resp == nil || len(resp.SeatBid) == 0 {
		return nil, warnings, nil
	}

	// Determine currency from response or config default
	currency := cfg.DefaultCurrency
	if resp.Cur != "" {
		currency = resp.Cur
	}

	// Collect all bids from all seats
	var allBids []bidWithSeat
	for _, seatBid := range resp.SeatBid {
		for _, bid := range seatBid.Bid {
			allBids = append(allBids, bidWithSeat{
				bid:  bid,
				seat: seatBid.Seat,
			})
		}
	}

	// Filter bids
	var filteredBids []bidWithSeat
	for _, bws := range allBids {
		// Filter: price must be > 0
		if bws.bid.Price <= 0 {
			warnings = append(warnings, "bid "+bws.bid.ID+" filtered: price <= 0")
			continue
		}

		// Filter: AdM must be non-empty unless AllowSkeletonVast is true
		if !cfg.AllowSkeletonVast && strings.TrimSpace(bws.bid.AdM) == "" {
			warnings = append(warnings, "bid "+bws.bid.ID+" filtered: empty AdM (skeleton VAST not allowed)")
			continue
		}

		filteredBids = append(filteredBids, bws)
	}

	if len(filteredBids) == 0 {
		return nil, warnings, nil
	}

	// Sort bids: price desc, deal exists desc, bid.ID asc for stability
	sort.Slice(filteredBids, func(i, j int) bool {
		bi, bj := filteredBids[i].bid, filteredBids[j].bid

		// Primary: price descending
		if bi.Price != bj.Price {
			return bi.Price > bj.Price
		}

		// Secondary: deal exists descending (deals first)
		iHasDeal := bi.DealID != ""
		jHasDeal := bj.DealID != ""
		if iHasDeal != jHasDeal {
			return iHasDeal
		}

		// Tertiary: bid ID ascending for stability
		return bi.ID < bj.ID
	})

	// Determine how many bids to return
	maxToReturn := s.maxBids
	if maxToReturn == 0 {
		maxToReturn = cfg.MaxAdsInPod
	}
	if maxToReturn <= 0 {
		maxToReturn = 1 // Safety fallback
	}
	if maxToReturn > len(filteredBids) {
		maxToReturn = len(filteredBids)
	}

	// Select top bids and build SelectedBid with CanonicalMeta
	selectedBids := make([]vast.SelectedBid, maxToReturn)
	for i := 0; i < maxToReturn; i++ {
		bws := filteredBids[i]
		bid := bws.bid

		// Determine sequence (SlotInPod)
		sequence := i + 1
		// Check if bid has explicit slot in pod via Ext or other mechanism
		// For MVP, we use index+1 as sequence

		// Extract primary adomain
		adomain := ""
		if len(bid.ADomain) > 0 {
			adomain = bid.ADomain[0]
		}

		// Extract duration from bid (if available in Dur field for video)
		durSec := 0
		if bid.Dur > 0 {
			durSec = int(bid.Dur)
		}

		selectedBids[i] = vast.SelectedBid{
			Bid:      bid,
			Seat:     bws.seat,
			Sequence: sequence,
			Meta: vast.CanonicalMeta{
				BidID:     bid.ID,
				ImpID:     bid.ImpID,
				DealID:    bid.DealID,
				Seat:      bws.seat,
				Price:     bid.Price,
				Currency:  currency,
				Adomain:   adomain,
				Cats:      bid.Cat,
				DurSec:    durSec,
				SlotInPod: sequence,
			},
		}
	}

	return selectedBids, warnings, nil
}

// Ensure PriceSelector implements BidSelector interface.
var _ vast.BidSelector = (*PriceSelector)(nil)
