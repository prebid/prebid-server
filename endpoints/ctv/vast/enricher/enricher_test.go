package enricher

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/endpoints/ctv/vast/model"
)

func TestEnricher_ExtractMetadata(t *testing.T) {
	config := DefaultConfig()
	enricher := NewEnricher(config).(*DefaultEnricher)

	bid := &openrtb2.Bid{
		ID:      "bid123",
		ImpID:   "imp1",
		Price:   5.50,
		ADomain: []string{"example.com"},
		Cat:     []string{"IAB1-1", "IAB1-2"},
		DealID:  "deal456",
		Dur:     30,
	}

	response := &openrtb2.BidResponse{
		Cur: "EUR",
	}

	metadata := enricher.extractMetadata(bid, "bidder1", response)

	if metadata.Price != 5.50 {
		t.Errorf("Expected price 5.50, got %f", metadata.Price)
	}

	if metadata.Currency != "EUR" {
		t.Errorf("Expected currency EUR, got %s", metadata.Currency)
	}

	if metadata.Advertiser != "example.com" {
		t.Errorf("Expected advertiser example.com, got %s", metadata.Advertiser)
	}

	if len(metadata.Categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(metadata.Categories))
	}

	if metadata.Duration != 30 {
		t.Errorf("Expected duration 30, got %d", metadata.Duration)
	}

	if metadata.BidID != "bid123" {
		t.Errorf("Expected bid ID bid123, got %s", metadata.BidID)
	}

	if metadata.DealID != "deal456" {
		t.Errorf("Expected deal ID deal456, got %s", metadata.DealID)
	}

	if metadata.Seat != "bidder1" {
		t.Errorf("Expected seat bidder1, got %s", metadata.Seat)
	}
}

func TestEnricher_CreateNewAd(t *testing.T) {
	config := DefaultConfig()
	enricher := NewEnricher(config).(*DefaultEnricher)

	bid := &openrtb2.Bid{
		ID:      "bid123",
		ImpID:   "imp1",
		Price:   5.50,
		ADomain: []string{"example.com"},
		Cat:     []string{"IAB1-1"},
	}

	metadata := BidMetadata{
		Price:      5.50,
		Currency:   "USD",
		Advertiser: "example.com",
		Categories: []string{"IAB1-1"},
		Duration:   30,
		BidID:      "bid123",
		Seat:       "bidder1",
	}

	ad := enricher.createNewAd(bid, metadata, 1)

	if ad.ID != "bid123" {
		t.Errorf("Expected ad ID bid123, got %s", ad.ID)
	}

	if ad.Sequence != 1 {
		t.Errorf("Expected sequence 1, got %d", ad.Sequence)
	}

	if ad.InLine == nil {
		t.Fatal("Expected InLine element")
	}

	if ad.InLine.Pricing == nil {
		t.Fatal("Expected Pricing element")
	}

	if ad.InLine.Pricing.Value != "5.50" {
		t.Errorf("Expected price 5.50, got %s", ad.InLine.Pricing.Value)
	}

	if ad.InLine.Pricing.Currency != "USD" {
		t.Errorf("Expected currency USD, got %s", ad.InLine.Pricing.Currency)
	}

	if ad.InLine.Advertiser != "example.com" {
		t.Errorf("Expected advertiser example.com, got %s", ad.InLine.Advertiser)
	}

	// Categories should be in EXTENSIONS by default, not INLINE
	if len(ad.InLine.Category) != 0 {
		t.Errorf("Expected 0 inline categories (should be in extensions), got %d", len(ad.InLine.Category))
	}

	// Check extensions for categories
	if ad.InLine.Extensions == nil || len(ad.InLine.Extensions.Extension) == 0 {
		t.Error("Expected extensions with categories")
	}
}

func TestEnricher_EnrichExistingVAST(t *testing.T) {
	config := DefaultConfig()
	config.CollisionPolicy = CollisionPolicyVASTWins
	enricher := NewEnricher(config)

	bid := &openrtb2.Bid{
		ID:      "bid123",
		Price:   7.50,
		ADomain: []string{"new-advertiser.com"},
		AdM:     "", // No AdM for this test
	}

	response := &openrtb2.BidResponse{
		Cur: "USD",
	}

	targetVAST := model.NewEmptyVAST("4.0")

	// Since AdM is empty, it will create new ad, not enrich existing
	// Let's test with valid VAST in AdM
	vastXML := `<VAST version="4.0">
		<Ad id="existing-ad">
			<InLine>
				<AdSystem>ExistingSystem</AdSystem>
				<AdTitle>Existing Title</AdTitle>
				<Advertiser>Existing Advertiser</Advertiser>
			</InLine>
		</Ad>
	</VAST>`

	bid.AdM = vastXML

	err := enricher.Enrich(targetVAST, bid, "bidder1", response, 1)
	if err != nil {
		t.Fatalf("Enrich failed: %v", err)
	}

	if len(targetVAST.Ad) != 1 {
		t.Fatalf("Expected 1 ad, got %d", len(targetVAST.Ad))
	}

	ad := targetVAST.Ad[0]

	// With VAST_WINS, existing advertiser should not be overwritten
	if ad.InLine.Advertiser != "Existing Advertiser" {
		t.Errorf("Expected existing advertiser to be preserved, got %s", ad.InLine.Advertiser)
	}

	// Pricing should be added since it didn't exist
	if ad.InLine.Pricing == nil {
		t.Error("Expected pricing to be added")
	} else if ad.InLine.Pricing.Value != "7.50" {
		t.Errorf("Expected price 7.50, got %s", ad.InLine.Pricing.Value)
	}
}

func TestEnricher_CollisionPolicyOpenRTBWins(t *testing.T) {
	config := DefaultConfig()
	config.CollisionPolicy = CollisionPolicyOpenRTBWins
	enricher := NewEnricher(config)

	vastXML := `<VAST version="4.0">
		<Ad id="existing-ad">
			<InLine>
				<AdSystem>ExistingSystem</AdSystem>
				<AdTitle>Existing Title</AdTitle>
				<Advertiser>Existing Advertiser</Advertiser>
			</InLine>
		</Ad>
	</VAST>`

	bid := &openrtb2.Bid{
		ID:      "bid123",
		Price:   7.50,
		ADomain: []string{"new-advertiser.com"},
		AdM:     vastXML,
	}

	response := &openrtb2.BidResponse{
		Cur: "USD",
	}

	targetVAST := model.NewEmptyVAST("4.0")

	err := enricher.Enrich(targetVAST, bid, "bidder1", response, 1)
	if err != nil {
		t.Fatalf("Enrich failed: %v", err)
	}

	ad := targetVAST.Ad[0]

	// With OPENRTB_WINS, advertiser should be overwritten
	if ad.InLine.Advertiser != "new-advertiser.com" {
		t.Errorf("Expected new advertiser, got %s", ad.InLine.Advertiser)
	}
}

func TestEnricher_PlacementExtensions(t *testing.T) {
	config := DefaultConfig()
	config.PlacementRules.Price = PlacementExtensions
	config.PlacementRules.Categories = PlacementExtensions
	config.PlacementRules.IDs = PlacementExtensions
	config.IncludeDebugIDs = true
	enricher := NewEnricher(config)

	bid := &openrtb2.Bid{
		ID:      "bid123",
		ImpID:   "imp1",
		Price:   5.50,
		ADomain: []string{"example.com"},
		Cat:     []string{"IAB1-1"},
	}

	response := &openrtb2.BidResponse{
		Cur: "USD",
	}

	targetVAST := model.NewEmptyVAST("4.0")

	err := enricher.Enrich(targetVAST, bid, "bidder1", response, 1)
	if err != nil {
		t.Fatalf("Enrich failed: %v", err)
	}

	ad := targetVAST.Ad[0]

	// Price should NOT be in inline pricing
	if ad.InLine.Pricing != nil {
		t.Error("Expected no inline pricing with PlacementExtensions")
	}

	// Should be in extensions
	if ad.InLine.Extensions == nil {
		t.Fatal("Expected extensions")
	}

	if len(ad.InLine.Extensions.Extension) == 0 {
		t.Error("Expected extension data")
	}
}

func TestEnricher_DefaultCurrency(t *testing.T) {
	config := DefaultConfig()
	config.DefaultCurrency = "EUR"
	enricher := NewEnricher(config).(*DefaultEnricher)

	bid := &openrtb2.Bid{
		ID:    "bid123",
		Price: 5.50,
	}

	// Response with no currency
	response := &openrtb2.BidResponse{}

	metadata := enricher.extractMetadata(bid, "bidder1", response)

	// Should use default
	if metadata.Currency != "EUR" {
		t.Errorf("Expected default currency EUR, got %s", metadata.Currency)
	}
}

func TestEnricher_NilInputs(t *testing.T) {
	config := DefaultConfig()
	enricher := NewEnricher(config)

	// Nil VAST
	err := enricher.Enrich(nil, &openrtb2.Bid{}, "seat", nil, 1)
	if err == nil {
		t.Error("Expected error for nil VAST")
	}

	// Nil bid
	vast := model.NewEmptyVAST("4.0")
	err = enricher.Enrich(vast, nil, "seat", nil, 1)
	if err == nil {
		t.Error("Expected error for nil bid")
	}
}

func TestEnrichWithDuration(t *testing.T) {
	vast := &model.VAST{
		Version: "4.0",
		Ad: []*model.Ad{
			{
				ID: "ad1",
				InLine: &model.InLine{
					Creatives: &model.Creatives{
						Creative: []*model.Creative{
							{
								ID: "creative1",
								Linear: &model.Linear{
									Duration: "", // Empty duration
								},
							},
						},
					},
				},
			},
		},
	}

	EnrichWithDuration(vast, 30)

	duration := vast.Ad[0].InLine.Creatives.Creative[0].Linear.Duration
	if duration != "00:00:30" {
		t.Errorf("Expected duration 00:00:30, got %s", duration)
	}
}

func TestValidateDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration int
		minDur   int
		maxDur   int
		expected bool
	}{
		{"Within range", 30, 15, 60, true},
		{"Below min", 10, 15, 60, false},
		{"Above max", 70, 15, 60, false},
		{"No min", 30, 0, 60, true},
		{"No max", 30, 15, 0, true},
		{"No limits", 30, 0, 0, true},
		{"Exact min", 15, 15, 60, true},
		{"Exact max", 60, 15, 60, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateDuration(tt.duration, tt.minDur, tt.maxDur)
			if result != tt.expected {
				t.Errorf("ValidateDuration(%d, %d, %d) = %v; expected %v",
					tt.duration, tt.minDur, tt.maxDur, result, tt.expected)
			}
		})
	}
}
