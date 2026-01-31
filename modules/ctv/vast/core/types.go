package core

import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/ctv/vast/model"
)

// VastResult contains the output of the VAST generation pipeline
type VastResult struct {
	VastXML  []byte         // The final VAST XML document
	NoAd     bool           // True if no ads were selected (response is no-ad VAST)
	Warnings []string       // Non-fatal warnings during generation
	Errors   []error        // Fatal errors encountered
	Selected []SelectedBid  // The bids that were selected and included
}

// SelectedBid represents a bid that was selected by the BidSelector
// along with metadata needed for enrichment and formatting
type SelectedBid struct {
	Bid      openrtb2.Bid  // The original OpenRTB bid
	Seat     string        // The bidder seat ID from seatbid
	Meta     CanonicalMeta // Normalized metadata extracted from bid and seatbid
	Sequence int           // Position in ad pod (1-based), 0 for single ad
}

// CanonicalMeta holds normalized OpenRTB metadata in a receiver-agnostic format
// This is the "canonical" representation that enrichers use to populate VAST fields
type CanonicalMeta struct {
	// Bid-level fields
	BidID         string   // From bid.id
	AdID          string   // From bid.adid (may be empty)
	ImpID         string   // From bid.impid
	Price         float64  // From bid.price
	Currency      string   // From seatbid.bid.ext.currency or top-level
	DealID        string   // From bid.dealid
	Seat          string   // From seatbid.seat
	CampaignID    string   // From bid.ext.meta.campaignId or similar
	CreativeID    string   // From bid.crid
	Adomain       string   // From bid.adomain[0] (first domain)
	AdvertiserID  string   // From bid.ext.meta.advertiserId
	AdvertiserName string  // From bid.ext.meta.advertiserName
	Cats          []string // From bid.cat (IAB categories)
	DurSec        int      // From bid.dur (duration in seconds)
	SlotInPod     int      // Desired position in ad pod (0 for no preference)

	// Top-level response fields (optional)
	BidID_ResponseLevel string // From bidresponse.bidid
	Currency_TopLevel   string // From bidresponse.cur[0]
}

// BidSelector chooses which bids from a BidResponse to include in the VAST response
type BidSelector interface {
	// Select examines the bid response and selects bids according to configured strategy
	// Returns:
	//   - selected: The bids chosen for inclusion (may be empty)
	//   - warnings: Non-fatal issues encountered during selection
	//   - error: Fatal error that prevents selection
	Select(req *openrtb2.BidRequest, resp *openrtb2.BidResponse, cfg ReceiverConfig) ([]SelectedBid, []string, error)
}

// Enricher adds metadata to a VAST Ad structure based on OpenRTB bid information
type Enricher interface {
	// Enrich takes a VAST ad and canonical metadata and populates VAST fields
	// according to receiver-specific rules (e.g., GAM_SSU format)
	// Returns:
	//   - warnings: Non-fatal issues encountered during enrichment
	//   - error: Fatal error that prevents enrichment
	Enrich(vast *model.VastAd, meta CanonicalMeta, cfg ReceiverConfig) ([]string, error)
}

// Formatter takes enriched VAST ads and produces the final XML document
type Formatter interface {
	// Format takes one or more VAST ads and produces final XML
	// Handles:
	//   - Single ad vs ad pod formatting
	//   - Sequence numbering for ad pods
	//   - VAST version and structure
	// Returns:
	//   - xml: The final VAST XML bytes
	//   - warnings: Non-fatal issues encountered during formatting
	//   - error: Fatal error that prevents formatting
	Format(vasts []*model.VastAd, cfg ReceiverConfig) ([]byte, []string, error)
}

// ReceiverConfig holds configuration for a specific VAST receiver
// This defines how OpenRTB bids should be transformed into VAST XML
type ReceiverConfig struct {
	// Core settings
	Receiver           string // Receiver identifier (e.g., "GAM_SSU", "FREEWHEEL")
	VastVersionDefault string // VAST version to use if not specified in bid (e.g., "3.0", "4.3")
	DefaultCurrency    string // Currency to assume if not in bid response (e.g., "USD")
	MaxAdsInPod        int    // Maximum ads to include in ad pod (0 = unlimited)

	// Selection strategy
	SelectionStrategy string // How to choose bids: "SINGLE", "TOP_N", "ALL"

	// VAST structure rules
	AllowSkeletonVast bool            // If true, create minimal VAST when bid.adm is empty
	CollisionPolicy   CollisionPolicy // How to handle conflicts between bid VAST and enrichment
	PlacementRules    PlacementRules  // Where to place different metadata fields
	EnableDebug       bool            // If true, include debug information in VAST Extensions

	// Enrichment settings
	TrackingPrefixes    map[string]string // URL prefixes for tracking pixels by event type
	PlacementPriorities []string          // Preferred placement types in priority order
	ExtraMetadata       map[string]string // Additional receiver-specific metadata
}

// GetAllowSkeletonVast returns whether skeleton VAST is allowed
func (c ReceiverConfig) GetAllowSkeletonVast() bool {
	return c.AllowSkeletonVast
}

// GetVastVersionDefault returns the default VAST version
func (c ReceiverConfig) GetVastVersionDefault() string {
	return c.VastVersionDefault
}

// CollisionPolicy defines how to handle conflicts when both bid VAST and enrichment
// want to populate the same field
type CollisionPolicy string

const (
	CollisionPolicyError      CollisionPolicy = "ERROR"        // Fail request on collision
	CollisionPolicyVastWins   CollisionPolicy = "VAST_WINS"    // Prefer existing VAST value
	CollisionPolicyOpenRTBWins CollisionPolicy = "OPENRTB_WINS" // Prefer OpenRTB enrichment value
	CollisionPolicyEnrichWins CollisionPolicy = "ENRICH_WINS"  // Prefer enrichment value (alias)
	CollisionPolicyMerge      CollisionPolicy = "MERGE"        // Attempt to merge both values
)

// PlacementRules defines where different metadata should be placed in VAST structure
type PlacementRules struct {
	PricingPlacement    Placement // Where to place pricing info
	AdvertiserPlacement Placement // Where to place advertiser info
	CategoriesPlacement Placement // Where to place category info
	DebugPlacement      Placement // Where to place debug/diagnostic info
}

// Placement specifies where in VAST XML a piece of metadata should go
type Placement string

const (
	PlacementInline     Placement = "INLINE"     // In <InLine> element
	PlacementWrapper    Placement = "WRAPPER"    // In <Wrapper> element (for wrappers)
	PlacementExtensions Placement = "EXTENSIONS" // In <Extensions> element
	PlacementSkip       Placement = "SKIP"       // Don't include (alias for OMIT)
	PlacementOmit       Placement = "OMIT"       // Don't include
)
