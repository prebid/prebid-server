package enricher

import (
	"encoding/json"
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/endpoints/ctv/vast/model"
)

// CollisionPolicy defines how to handle existing VAST fields
type CollisionPolicy string

const (
	// CollisionPolicyVASTWins means don't overwrite existing non-empty VAST fields
	CollisionPolicyVASTWins CollisionPolicy = "VAST_WINS"
	// CollisionPolicyOpenRTBWins means always overwrite with OpenRTB data
	CollisionPolicyOpenRTBWins CollisionPolicy = "OPENRTB_WINS"
)

// PlacementRules defines where to place enrichment data
type PlacementRules struct {
	Price       Placement
	Currency    Placement
	Advertiser  Placement
	Categories  Placement
	Duration    Placement
	IDs         Placement
	DealID      Placement
}

// Placement defines where a field should be placed
type Placement string

const (
	// PlacementInline places data in standard VAST elements
	PlacementInline Placement = "INLINE"
	// PlacementExtensions places data in Extensions
	PlacementExtensions Placement = "EXTENSIONS"
	// PlacementSkip skips this enrichment
	PlacementSkip Placement = "SKIP"
)

// Config holds enricher configuration
type Config struct {
	CollisionPolicy CollisionPolicy
	PlacementRules  PlacementRules
	DefaultCurrency string
	IncludeDebugIDs bool
}

// DefaultConfig returns default enricher configuration
func DefaultConfig() Config {
	return Config{
		CollisionPolicy: CollisionPolicyVASTWins,
		PlacementRules: PlacementRules{
			Price:      PlacementInline,
			Currency:   PlacementInline,
			Advertiser: PlacementInline,
			Categories: PlacementExtensions,
			Duration:   PlacementInline,
			IDs:        PlacementExtensions,
			DealID:     PlacementExtensions,
		},
		DefaultCurrency: "USD",
		IncludeDebugIDs: false,
	}
}

// BidMetadata contains canonicalized bid metadata
type BidMetadata struct {
	Price      float64
	Currency   string
	Advertiser string
	Categories []string
	Duration   int // in seconds
	BidID      string
	ImpID      string
	DealID     string
	Seat       string
}

// Enricher enriches VAST with OpenRTB bid data
type Enricher interface {
	// Enrich enriches a VAST Ad with bid metadata
	Enrich(vast *model.VAST, bid *openrtb2.Bid, seat string, response *openrtb2.BidResponse, sequence int) error
}

// DefaultEnricher implements the Enricher interface
type DefaultEnricher struct {
	config Config
}

// NewEnricher creates a new DefaultEnricher
func NewEnricher(config Config) Enricher {
	return &DefaultEnricher{config: config}
}

// Enrich implements Enricher.Enrich
func (e *DefaultEnricher) Enrich(vast *model.VAST, bid *openrtb2.Bid, seat string, response *openrtb2.BidResponse, sequence int) error {
	if vast == nil || bid == nil {
		return fmt.Errorf("vast or bid is nil")
	}

	// Extract metadata from bid
	metadata := e.extractMetadata(bid, seat, response)

	// Parse existing VAST from bid.AdM if present
	existingVAST, err := e.parseAdM(bid.AdM)
	if err == nil && existingVAST != nil {
		// Use existing VAST structure, enrich it
		return e.enrichExisting(existingVAST, metadata, sequence, vast)
	}

	// Create new Ad if no valid VAST in AdM
	ad := e.createNewAd(bid, metadata, sequence)
	vast.AddAd(ad)

	return nil
}

// extractMetadata extracts canonical metadata from bid
func (e *DefaultEnricher) extractMetadata(bid *openrtb2.Bid, seat string, response *openrtb2.BidResponse) BidMetadata {
	metadata := BidMetadata{
		Price:    bid.Price,
		Currency: e.config.DefaultCurrency,
		BidID:    bid.ID,
		ImpID:    bid.ImpID,
		DealID:   bid.DealID,
		Seat:     seat,
		Duration: 0,
	}

	// Get currency from response
	if response != nil && len(response.Cur) > 0 {
		metadata.Currency = response.Cur
	}

	// Get advertiser from ADomain
	if len(bid.ADomain) > 0 {
		metadata.Advertiser = bid.ADomain[0]
	}

	// Get categories
	if len(bid.Cat) > 0 {
		metadata.Categories = bid.Cat
	}

	// Get duration if present (in seconds)
	if bid.Dur > 0 {
		metadata.Duration = int(bid.Dur)
	}

	return metadata
}

// parseAdM attempts to parse bid.AdM as VAST XML
func (e *DefaultEnricher) parseAdM(adm string) (*model.VAST, error) {
	if adm == "" {
		return nil, fmt.Errorf("empty adm")
	}

	// Try to parse as VAST
	vast, err := model.ParseString(adm)
	if err != nil {
		return nil, err
	}

	return vast, nil
}

// enrichExisting enriches existing VAST and adds it to target
func (e *DefaultEnricher) enrichExisting(existing *model.VAST, metadata BidMetadata, sequence int, target *model.VAST) error {
	// Process each ad in existing VAST
	for _, ad := range existing.Ad {
		if ad.Sequence == 0 && sequence > 0 {
			ad.Sequence = sequence
		}

		// Enrich InLine
		if ad.InLine != nil {
			e.enrichInLine(ad.InLine, metadata)
		}

		// Enrich Wrapper
		if ad.Wrapper != nil {
			e.enrichWrapper(ad.Wrapper, metadata)
		}

		target.AddAd(ad)
	}

	return nil
}

// enrichInLine enriches InLine element with metadata
func (e *DefaultEnricher) enrichInLine(inline *model.InLine, metadata BidMetadata) {
	// Enrich price
	if e.config.PlacementRules.Price == PlacementInline {
		if e.shouldSetField(inline.Pricing != nil) {
			if inline.Pricing == nil {
				inline.Pricing = &model.Pricing{}
			}
			inline.Pricing.Model = "CPM"
			inline.Pricing.Currency = metadata.Currency
			inline.Pricing.Value = fmt.Sprintf("%.2f", metadata.Price)
		}
	}

	// Enrich advertiser
	if e.config.PlacementRules.Advertiser == PlacementInline && metadata.Advertiser != "" {
		if e.shouldSetField(inline.Advertiser != "") {
			inline.Advertiser = metadata.Advertiser
		}
	}

	// Enrich categories
	if e.config.PlacementRules.Categories == PlacementInline && len(metadata.Categories) > 0 {
		if e.shouldSetField(len(inline.Category) > 0) {
			for _, cat := range metadata.Categories {
				inline.Category = append(inline.Category, model.Category{
					Authority: "IAB",
					Value:     cat,
				})
			}
		}
	}

	// Enrich duration in creatives
	if e.config.PlacementRules.Duration == PlacementInline && metadata.Duration > 0 {
		if inline.Creatives != nil {
			for _, creative := range inline.Creatives.Creative {
				if creative.Linear != nil {
					if e.shouldSetField(creative.Linear.Duration != "") {
						creative.Linear.Duration = model.FormatDuration(metadata.Duration)
					}
				}
			}
		}
	}

	// Add extensions if needed
	if e.shouldAddExtensions(metadata) {
		if inline.Extensions == nil {
			inline.Extensions = &model.Extensions{}
		}
		e.addExtensionData(inline.Extensions, metadata)
	}
}

// enrichWrapper enriches Wrapper element with metadata
func (e *DefaultEnricher) enrichWrapper(wrapper *model.Wrapper, metadata BidMetadata) {
	// Add extensions if needed
	if e.shouldAddExtensions(metadata) {
		if wrapper.Extensions == nil {
			wrapper.Extensions = &model.Extensions{}
		}
		e.addExtensionData(wrapper.Extensions, metadata)
	}
}

// createNewAd creates a new minimal Ad when no VAST in AdM
func (e *DefaultEnricher) createNewAd(bid *openrtb2.Bid, metadata BidMetadata, sequence int) *model.Ad {
	ad := &model.Ad{
		ID:       bid.ID,
		Sequence: sequence,
		InLine: &model.InLine{
			AdSystem: &model.AdSystem{
				Value: metadata.Seat,
			},
			AdTitle: fmt.Sprintf("Ad %s", bid.ID),
			Impression: []model.Impression{
				{Value: ""}, // Placeholder - should be filled by receiver
			},
		},
	}

	// Set pricing if inline
	if e.config.PlacementRules.Price == PlacementInline {
		ad.InLine.Pricing = &model.Pricing{
			Model:    "CPM",
			Currency: metadata.Currency,
			Value:    fmt.Sprintf("%.2f", metadata.Price),
		}
	}

	// Set advertiser if inline
	if e.config.PlacementRules.Advertiser == PlacementInline && metadata.Advertiser != "" {
		ad.InLine.Advertiser = metadata.Advertiser
	}

	// Set categories if inline
	if e.config.PlacementRules.Categories == PlacementInline && len(metadata.Categories) > 0 {
		ad.InLine.Category = make([]model.Category, 0, len(metadata.Categories))
		for _, cat := range metadata.Categories {
			ad.InLine.Category = append(ad.InLine.Category, model.Category{
				Authority: "IAB",
				Value:     cat,
			})
		}
	}

	// Create minimal creative structure with AdM content
	ad.InLine.Creatives = &model.Creatives{
		Creative: []*model.Creative{
			{
				ID: fmt.Sprintf("creative-%s", bid.ID),
				Linear: &model.Linear{
					Duration: model.FormatDuration(metadata.Duration),
					MediaFiles: &model.MediaFiles{
						MediaFile: []model.MediaFile{
							// Placeholder - actual media from bid.AdM or bid.NURL
							{
								Delivery: "progressive",
								Type:     "video/mp4",
								Value:    bid.NURL, // Use NURL if available
							},
						},
					},
				},
			},
		},
	}

	// Add extensions if needed
	if e.shouldAddExtensions(metadata) {
		ad.InLine.Extensions = &model.Extensions{}
		e.addExtensionData(ad.InLine.Extensions, metadata)
	}

	return ad
}

// shouldSetField determines if a field should be set based on collision policy
func (e *DefaultEnricher) shouldSetField(fieldExists bool) bool {
	if e.config.CollisionPolicy == CollisionPolicyOpenRTBWins {
		return true
	}
	// VAST_WINS: only set if field doesn't exist or is empty
	return !fieldExists
}

// shouldAddExtensions determines if extensions should be added
func (e *DefaultEnricher) shouldAddExtensions(metadata BidMetadata) bool {
	rules := e.config.PlacementRules
	return (rules.IDs == PlacementExtensions && e.config.IncludeDebugIDs) ||
		rules.DealID == PlacementExtensions ||
		rules.Price == PlacementExtensions ||
		rules.Categories == PlacementExtensions
}

// addExtensionData adds enrichment data to extensions
func (e *DefaultEnricher) addExtensionData(extensions *model.Extensions, metadata BidMetadata) {
	extData := make(map[string]interface{})

	rules := e.config.PlacementRules

	if rules.Price == PlacementExtensions {
		extData["price"] = metadata.Price
		extData["currency"] = metadata.Currency
	}

	if rules.IDs == PlacementExtensions && e.config.IncludeDebugIDs {
		extData["bid_id"] = metadata.BidID
		extData["imp_id"] = metadata.ImpID
		extData["seat"] = metadata.Seat
	}

	if rules.DealID == PlacementExtensions && metadata.DealID != "" {
		extData["deal_id"] = metadata.DealID
	}

	if rules.Categories == PlacementExtensions && len(metadata.Categories) > 0 {
		extData["categories"] = metadata.Categories
	}

	if len(extData) > 0 {
		jsonData, err := json.Marshal(extData)
		if err == nil {
			extensions.Extension = append(extensions.Extension, model.Extension{
				Type:     "prebid",
				InnerXML: string(jsonData),
			})
		}
	}
}

// EnrichWithDuration is a helper to set duration on existing VAST
func EnrichWithDuration(vast *model.VAST, durationSeconds int) {
	if vast == nil || durationSeconds <= 0 {
		return
	}

	durationStr := model.FormatDuration(durationSeconds)

	for _, ad := range vast.Ad {
		if ad.InLine != nil && ad.InLine.Creatives != nil {
			for _, creative := range ad.InLine.Creatives.Creative {
				if creative.Linear != nil && creative.Linear.Duration == "" {
					creative.Linear.Duration = durationStr
				}
			}
		}
	}
}

// ValidateDuration validates duration is within acceptable range
func ValidateDuration(durationSeconds int, minDur int, maxDur int) bool {
	if minDur > 0 && durationSeconds < minDur {
		return false
	}
	if maxDur > 0 && durationSeconds > maxDur {
		return false
	}
	return true
}
