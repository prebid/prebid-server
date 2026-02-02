// Package vast provides CTV VAST processing capabilities for Prebid Server.
// It includes bid selection, VAST enrichment, and formatting for various receivers.
package vast

import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/ctv/vast/model"
)

// ReceiverType identifies the downstream ad receiver/player.
type ReceiverType string

const (
	// ReceiverGAMSSU represents Google Ad Manager Server-Side Unified receiver.
	ReceiverGAMSSU ReceiverType = "GAM_SSU"
	// ReceiverGeneric represents a generic VAST-compliant receiver.
	ReceiverGeneric ReceiverType = "GENERIC"
)

// SelectionStrategy defines how bids are selected for ad pods.
type SelectionStrategy string

const (
	// SelectionSingle selects a single best bid.
	SelectionSingle SelectionStrategy = "SINGLE"
	// SelectionTopN selects up to MaxAdsInPod bids.
	SelectionTopN SelectionStrategy = "TOP_N"
	// SelectionMaxRevenue selects bids to maximize total revenue.
	SelectionMaxRevenue SelectionStrategy = "max_revenue"
	// SelectionMinDuration selects bids to minimize total duration.
	SelectionMinDuration SelectionStrategy = "min_duration"
	// SelectionBalanced balances between revenue and duration.
	SelectionBalanced SelectionStrategy = "balanced"
)

// CollisionPolicy defines how to handle competitive separation violations.
type CollisionPolicy string

const (
	// CollisionReject rejects ads that violate competitive separation.
	CollisionReject CollisionPolicy = "reject"
	// CollisionWarn allows ads but adds warnings for violations.
	CollisionWarn CollisionPolicy = "warn"
	// CollisionIgnore ignores competitive separation rules.
	CollisionIgnore CollisionPolicy = "ignore"
)

// VastResult holds the complete result of VAST processing.
type VastResult struct {
	// VastXML contains the final VAST XML output.
	VastXML []byte
	// NoAd indicates if no valid ad was available.
	NoAd bool
	// Warnings contains non-fatal issues encountered during processing.
	Warnings []string
	// Errors contains fatal errors that occurred during processing.
	Errors []error
	// Selected contains the bids that were selected for the ad pod.
	Selected []SelectedBid
}

// SelectedBid represents a bid that was selected for inclusion in the VAST response.
type SelectedBid struct {
	// Bid is the OpenRTB bid object.
	Bid openrtb2.Bid
	// Seat is the seat ID of the bidder.
	Seat string
	// Sequence is the position of this bid in the ad pod (1-indexed).
	Sequence int
	// Meta contains canonical metadata extracted from the bid.
	Meta CanonicalMeta
}

// CanonicalMeta contains normalized metadata for a selected bid.
type CanonicalMeta struct {
	// BidID is the unique identifier for the bid.
	BidID string
	// ImpID is the impression ID this bid is for.
	ImpID string
	// DealID is the deal ID if this bid is from a deal.
	DealID string
	// Seat is the bidder seat ID.
	Seat string
	// Price is the bid price.
	Price float64
	// Currency is the currency code for the price.
	Currency string
	// Adomain is the primary advertiser domain.
	Adomain string
	// Cats contains the IAB content categories.
	Cats []string
	// DurSec is the duration of the creative in seconds.
	DurSec int
	// SlotInPod is the position within the ad pod (1-indexed).
	SlotInPod int
}

// ReceiverConfig holds configuration for VAST processing.
type ReceiverConfig struct {
	// Receiver identifies the downstream ad receiver type.
	Receiver ReceiverType
	// DefaultCurrency is the currency to use when not specified.
	DefaultCurrency string
	// VastVersionDefault is the default VAST version to output.
	VastVersionDefault string
	// MaxAdsInPod is the maximum number of ads allowed in a pod.
	MaxAdsInPod int
	// SelectionStrategy defines how bids are selected.
	SelectionStrategy SelectionStrategy
	// CollisionPolicy defines how competitive separation is handled.
	CollisionPolicy CollisionPolicy
	// Placement contains placement-specific rules.
	Placement PlacementRules
	// AllowSkeletonVast allows bids without AdM content (skeleton VAST).
	AllowSkeletonVast bool
	// Debug enables debug mode with additional output.
	Debug bool
}

// PlacementRules contains rules for validating and filtering bids.
type PlacementRules struct {
	// Pricing contains price floor and ceiling rules.
	Pricing PricingRules
	// Advertiser contains advertiser-based filtering rules.
	Advertiser AdvertiserRules
	// Categories contains category-based filtering rules.
	Categories CategoryRules
	// PricingPlacement defines where to place pricing info: "VAST_PRICING" or "EXTENSION".
	PricingPlacement string
	// AdvertiserPlacement defines where to place advertiser info: "ADVERTISER_TAG" or "EXTENSION".
	AdvertiserPlacement string
	// Debug enables debug output for placement rules.
	Debug bool
}

// PricingRules defines pricing constraints for bid selection.
type PricingRules struct {
	// FloorCPM is the minimum CPM allowed.
	FloorCPM float64
	// CeilingCPM is the maximum CPM allowed (0 = no ceiling).
	CeilingCPM float64
	// Currency is the currency for floor/ceiling values.
	Currency string
}

// AdvertiserRules defines advertiser-based filtering.
type AdvertiserRules struct {
	// BlockedDomains is a list of advertiser domains to reject.
	BlockedDomains []string
	// AllowedDomains is a whitelist of allowed domains (empty = allow all).
	AllowedDomains []string
}

// CategoryRules defines category-based filtering.
type CategoryRules struct {
	// BlockedCategories is a list of IAB categories to reject.
	BlockedCategories []string
	// AllowedCategories is a whitelist of allowed categories (empty = allow all).
	AllowedCategories []string
}

// BidSelector defines the interface for selecting bids from an auction response.
type BidSelector interface {
	// Select chooses bids from the response based on configuration.
	// Returns selected bids, warnings, and any fatal error.
	Select(req *openrtb2.BidRequest, resp *openrtb2.BidResponse, cfg ReceiverConfig) ([]SelectedBid, []string, error)
}

// Enricher defines the interface for enriching VAST ads with additional data.
type Enricher interface {
	// Enrich adds tracking, extensions, and other data to a VAST ad.
	// Returns warnings and any fatal error.
	Enrich(ad *model.Ad, meta CanonicalMeta, cfg ReceiverConfig) ([]string, error)
}

// EnrichedAd pairs a VAST Ad with its associated metadata.
type EnrichedAd struct {
	// Ad is the enriched VAST Ad element.
	Ad *model.Ad
	// Meta contains canonical metadata for this ad.
	Meta CanonicalMeta
	// Sequence is the position in the ad pod (1-indexed).
	Sequence int
}

// Formatter defines the interface for formatting VAST ads into XML.
type Formatter interface {
	// Format converts enriched VAST ads into XML output.
	// Returns the XML bytes, warnings, and any fatal error.
	Format(ads []EnrichedAd, cfg ReceiverConfig) ([]byte, []string, error)
}
