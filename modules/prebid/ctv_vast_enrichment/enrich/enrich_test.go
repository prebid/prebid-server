package enrich

import (
	"testing"

	"github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment"
	"github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEnricher(t *testing.T) {
	enricher := NewEnricher()
	assert.NotNil(t, enricher)
}

func TestEnrich_NilAd(t *testing.T) {
	enricher := NewEnricher()
	meta := vast.CanonicalMeta{}
	cfg := vast.ReceiverConfig{}

	warnings, err := enricher.Enrich(nil, meta, cfg)
	assert.NoError(t, err)
	assert.Empty(t, warnings)
}

func TestEnrich_WrapperAd(t *testing.T) {
	enricher := NewEnricher()
	ad := &model.Ad{
		ID:      "wrapper",
		Wrapper: &model.Wrapper{},
	}
	meta := vast.CanonicalMeta{Price: 5.0}
	cfg := vast.ReceiverConfig{}

	warnings, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "not InLine")
}

func TestEnrich_Pricing_VastWins_ExistingNotOverwritten(t *testing.T) {
	enricher := NewEnricher()
	ad := createTestAd()
	ad.InLine.Pricing = &model.Pricing{
		Model:    "CPM",
		Currency: "EUR",
		Value:    "10.00",
	}

	meta := vast.CanonicalMeta{
		Price:    5.0,
		Currency: "USD",
	}
	cfg := vast.ReceiverConfig{
		DefaultCurrency: "USD",
		Placement: vast.PlacementRules{
			PricingPlacement: vast.PlacementVastPricing,
		},
	}

	warnings, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)

	// Should have warning about VAST_WINS
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "VAST_WINS")

	// Original pricing should be preserved
	assert.Equal(t, "EUR", ad.InLine.Pricing.Currency)
	assert.Equal(t, "10.00", ad.InLine.Pricing.Value)
}

func TestEnrich_Pricing_AddedWhenMissing(t *testing.T) {
	enricher := NewEnricher()
	ad := createTestAd()
	ad.InLine.Pricing = nil

	meta := vast.CanonicalMeta{
		Price:    5.5,
		Currency: "USD",
	}
	cfg := vast.ReceiverConfig{
		DefaultCurrency: "USD",
		Placement: vast.PlacementRules{
			PricingPlacement: vast.PlacementVastPricing,
		},
	}

	warnings, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)
	assert.Empty(t, warnings)

	// Pricing should be added
	require.NotNil(t, ad.InLine.Pricing)
	assert.Equal(t, "CPM", ad.InLine.Pricing.Model)
	assert.Equal(t, "USD", ad.InLine.Pricing.Currency)
	assert.Equal(t, "5.5", ad.InLine.Pricing.Value)
}

func TestEnrich_Pricing_AsExtension(t *testing.T) {
	enricher := NewEnricher()
	ad := createTestAd()
	ad.InLine.Pricing = nil

	meta := vast.CanonicalMeta{
		Price:    3.25,
		Currency: "EUR",
	}
	cfg := vast.ReceiverConfig{
		Placement: vast.PlacementRules{
			PricingPlacement: vast.PlacementExtension,
		},
	}

	warnings, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)
	assert.Empty(t, warnings)

	// Pricing should be nil (not added to VAST element)
	assert.Nil(t, ad.InLine.Pricing)

	// Should have extension with pricing
	require.NotNil(t, ad.InLine.Extensions)
	found := false
	for _, ext := range ad.InLine.Extensions.Extension {
		if ext.Type == "pricing" {
			found = true
			assert.Contains(t, ext.InnerXML, "3.25")
			assert.Contains(t, ext.InnerXML, "EUR")
			assert.Contains(t, ext.InnerXML, "CPM")
		}
	}
	assert.True(t, found, "pricing extension not found")
}

func TestEnrich_Pricing_ZeroPriceNotAdded(t *testing.T) {
	enricher := NewEnricher()
	ad := createTestAd()
	ad.InLine.Pricing = nil

	meta := vast.CanonicalMeta{
		Price: 0,
	}
	cfg := vast.ReceiverConfig{}

	warnings, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)
	assert.Empty(t, warnings)
	assert.Nil(t, ad.InLine.Pricing)
}

func TestEnrich_Advertiser_VastWins_ExistingNotOverwritten(t *testing.T) {
	enricher := NewEnricher()
	ad := createTestAd()
	ad.InLine.Advertiser = "Original Advertiser"

	meta := vast.CanonicalMeta{
		Adomain: "newadvertiser.com",
	}
	cfg := vast.ReceiverConfig{
		Placement: vast.PlacementRules{
			AdvertiserPlacement: vast.PlacementAdvertiserTag,
		},
	}

	warnings, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)

	// Should have warning about VAST_WINS
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "VAST_WINS")

	// Original advertiser should be preserved
	assert.Equal(t, "Original Advertiser", ad.InLine.Advertiser)
}

func TestEnrich_Advertiser_AddedWhenMissing(t *testing.T) {
	enricher := NewEnricher()
	ad := createTestAd()
	ad.InLine.Advertiser = ""

	meta := vast.CanonicalMeta{
		Adomain: "example.com",
	}
	cfg := vast.ReceiverConfig{
		Placement: vast.PlacementRules{
			AdvertiserPlacement: vast.PlacementAdvertiserTag,
		},
	}

	warnings, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)
	assert.Empty(t, warnings)
	assert.Equal(t, "example.com", ad.InLine.Advertiser)
}

func TestEnrich_Advertiser_AsExtension(t *testing.T) {
	enricher := NewEnricher()
	ad := createTestAd()
	ad.InLine.Advertiser = ""

	meta := vast.CanonicalMeta{
		Adomain: "example.com",
	}
	cfg := vast.ReceiverConfig{
		Placement: vast.PlacementRules{
			AdvertiserPlacement: vast.PlacementExtension,
		},
	}

	warnings, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)
	assert.Empty(t, warnings)

	// Advertiser tag should be empty
	assert.Equal(t, "", ad.InLine.Advertiser)

	// Should have extension with advertiser
	require.NotNil(t, ad.InLine.Extensions)
	found := false
	for _, ext := range ad.InLine.Extensions.Extension {
		if ext.Type == "advertiser" {
			found = true
			assert.Contains(t, ext.InnerXML, "example.com")
		}
	}
	assert.True(t, found, "advertiser extension not found")
}

func TestEnrich_Duration_VastWins_ExistingNotOverwritten(t *testing.T) {
	enricher := NewEnricher()
	ad := createTestAd()
	ad.InLine.Creatives.Creative[0].Linear.Duration = "00:00:30"

	meta := vast.CanonicalMeta{
		DurSec: 15,
	}
	cfg := vast.ReceiverConfig{}

	warnings, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)

	// Should have warning about VAST_WINS
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "VAST_WINS")

	// Original duration should be preserved
	assert.Equal(t, "00:00:30", ad.InLine.Creatives.Creative[0].Linear.Duration)
}

func TestEnrich_Duration_AddedWhenMissing(t *testing.T) {
	enricher := NewEnricher()
	ad := createTestAd()
	ad.InLine.Creatives.Creative[0].Linear.Duration = ""

	meta := vast.CanonicalMeta{
		DurSec: 15,
	}
	cfg := vast.ReceiverConfig{}

	warnings, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)
	assert.Empty(t, warnings)
	assert.Equal(t, "00:00:15", ad.InLine.Creatives.Creative[0].Linear.Duration)
}

func TestEnrich_Duration_ZeroNotAdded(t *testing.T) {
	enricher := NewEnricher()
	ad := createTestAd()
	ad.InLine.Creatives.Creative[0].Linear.Duration = ""

	meta := vast.CanonicalMeta{
		DurSec: 0,
	}
	cfg := vast.ReceiverConfig{}

	warnings, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)
	assert.Empty(t, warnings)
	assert.Equal(t, "", ad.InLine.Creatives.Creative[0].Linear.Duration)
}

func TestEnrich_Categories_AddedAsExtension(t *testing.T) {
	enricher := NewEnricher()
	ad := createTestAd()

	meta := vast.CanonicalMeta{
		Cats: []string{"IAB1", "IAB2-1", "IAB3"},
	}
	cfg := vast.ReceiverConfig{}

	warnings, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)
	assert.Empty(t, warnings)

	// Should have extension with categories
	require.NotNil(t, ad.InLine.Extensions)
	found := false
	for _, ext := range ad.InLine.Extensions.Extension {
		if ext.Type == "iab_category" {
			found = true
			assert.Contains(t, ext.InnerXML, "<Category>IAB1</Category>")
			assert.Contains(t, ext.InnerXML, "<Category>IAB2-1</Category>")
			assert.Contains(t, ext.InnerXML, "<Category>IAB3</Category>")
		}
	}
	assert.True(t, found, "iab_category extension not found")
}

func TestEnrich_Categories_EmptyNotAdded(t *testing.T) {
	enricher := NewEnricher()
	ad := createTestAd()

	meta := vast.CanonicalMeta{
		Cats: []string{},
	}
	cfg := vast.ReceiverConfig{}

	warnings, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)
	assert.Empty(t, warnings)

	// Should not have category extension
	if ad.InLine.Extensions != nil {
		for _, ext := range ad.InLine.Extensions.Extension {
			assert.NotEqual(t, "iab_category", ext.Type)
		}
	}
}

func TestEnrich_DebugExtension(t *testing.T) {
	enricher := NewEnricher()
	ad := createTestAd()

	meta := vast.CanonicalMeta{
		BidID:    "bid123",
		ImpID:    "imp456",
		DealID:   "deal789",
		Seat:     "bidder1",
		Price:    2.5,
		Currency: "USD",
	}
	cfg := vast.ReceiverConfig{
		Debug: true,
	}

	warnings, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)
	assert.Empty(t, warnings)

	// Should have openrtb debug extension
	require.NotNil(t, ad.InLine.Extensions)
	found := false
	for _, ext := range ad.InLine.Extensions.Extension {
		if ext.Type == "openrtb" {
			found = true
			assert.Contains(t, ext.InnerXML, "<BidID>bid123</BidID>")
			assert.Contains(t, ext.InnerXML, "<ImpID>imp456</ImpID>")
			assert.Contains(t, ext.InnerXML, "<DealID>deal789</DealID>")
			assert.Contains(t, ext.InnerXML, "<Seat>bidder1</Seat>")
			assert.Contains(t, ext.InnerXML, "<Price>2.5</Price>")
			assert.Contains(t, ext.InnerXML, "<Currency>USD</Currency>")
		}
	}
	assert.True(t, found, "openrtb extension not found")
}

func TestEnrich_DebugExtension_NoDealID(t *testing.T) {
	enricher := NewEnricher()
	ad := createTestAd()

	meta := vast.CanonicalMeta{
		BidID:    "bid123",
		ImpID:    "imp456",
		DealID:   "", // No deal
		Seat:     "bidder1",
		Price:    2.5,
		Currency: "USD",
	}
	cfg := vast.ReceiverConfig{
		Debug: true,
	}

	_, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)

	// Should have openrtb debug extension without DealID
	require.NotNil(t, ad.InLine.Extensions)
	for _, ext := range ad.InLine.Extensions.Extension {
		if ext.Type == "openrtb" {
			assert.NotContains(t, ext.InnerXML, "<DealID>")
		}
	}
}

func TestEnrich_DebugExtension_PlacementDebug(t *testing.T) {
	enricher := NewEnricher()
	ad := createTestAd()

	meta := vast.CanonicalMeta{
		BidID: "bid123",
	}
	cfg := vast.ReceiverConfig{
		Debug: false, // Global debug off
		Placement: vast.PlacementRules{
			Debug: true, // Placement debug on
		},
	}

	_, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)

	// Should have openrtb debug extension
	found := false
	for _, ext := range ad.InLine.Extensions.Extension {
		if ext.Type == "openrtb" {
			found = true
		}
	}
	assert.True(t, found, "openrtb extension not found when placement debug enabled")
}

func TestEnrich_FullEnrichment(t *testing.T) {
	enricher := NewEnricher()
	ad := createTestAd()
	ad.InLine.Pricing = nil
	ad.InLine.Advertiser = ""
	ad.InLine.Creatives.Creative[0].Linear.Duration = ""

	meta := vast.CanonicalMeta{
		BidID:    "bid123",
		ImpID:    "imp456",
		Seat:     "bidder1",
		Price:    5.5,
		Currency: "USD",
		Adomain:  "advertiser.com",
		Cats:     []string{"IAB1", "IAB2"},
		DurSec:   30,
	}
	cfg := vast.ReceiverConfig{
		DefaultCurrency: "USD",
		Debug:           true,
		Placement: vast.PlacementRules{
			PricingPlacement:    vast.PlacementVastPricing,
			AdvertiserPlacement: vast.PlacementAdvertiserTag,
		},
	}

	warnings, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)
	assert.Empty(t, warnings)

	// Check all enrichments
	require.NotNil(t, ad.InLine.Pricing)
	assert.Equal(t, "5.5", ad.InLine.Pricing.Value)
	assert.Equal(t, "advertiser.com", ad.InLine.Advertiser)
	assert.Equal(t, "00:00:30", ad.InLine.Creatives.Creative[0].Linear.Duration)

	// Check extensions
	require.NotNil(t, ad.InLine.Extensions)
	hasCategory := false
	hasOpenRTB := false
	for _, ext := range ad.InLine.Extensions.Extension {
		if ext.Type == "iab_category" {
			hasCategory = true
		}
		if ext.Type == "openrtb" {
			hasOpenRTB = true
		}
	}
	assert.True(t, hasCategory)
	assert.True(t, hasOpenRTB)
}

func TestFormatPrice(t *testing.T) {
	tests := []struct {
		price    float64
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{1.5, "1.5"},
		{1.50, "1.5"},
		{1.55, "1.55"},
		{1.555, "1.555"},
		{1.5555, "1.5555"},
		{1.55555, "1.5555"}, // Truncates to 4 decimals
		{10.00, "10"},
		{0.001, "0.001"},
		{0.0001, "0.0001"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatPrice(tt.price)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"a & b", "a &amp; b"},
		{"<tag>", "&lt;tag&gt;"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"it's", "it&apos;s"},
		{"<a & 'b'>", "&lt;a &amp; &apos;b&apos;&gt;"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeXML(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnrich_XMLMarshalRoundTrip(t *testing.T) {
	enricher := NewEnricher()

	// Parse sample VAST
	sampleVAST := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="3.0">
  <Ad id="test-ad">
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <Creatives>
        <Creative>
          <Linear>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080">
                <![CDATA[https://example.com/video.mp4]]>
              </MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	parsedVast, err := model.ParseVastAdm(sampleVAST)
	require.NoError(t, err)
	require.Len(t, parsedVast.Ads, 1)

	ad := &parsedVast.Ads[0]
	meta := vast.CanonicalMeta{
		BidID:    "bid123",
		ImpID:    "imp456",
		Price:    5.0,
		Currency: "USD",
		Adomain:  "advertiser.com",
		Cats:     []string{"IAB1"},
		DurSec:   30,
	}
	cfg := vast.ReceiverConfig{
		Debug: true,
	}

	warnings, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)
	assert.Empty(t, warnings)

	// Marshal back to XML
	xmlBytes, err := parsedVast.Marshal()
	require.NoError(t, err)

	xmlStr := string(xmlBytes)
	assert.Contains(t, xmlStr, "Pricing")
	assert.Contains(t, xmlStr, "advertiser.com")
	assert.Contains(t, xmlStr, "00:00:30")
	assert.Contains(t, xmlStr, "iab_category")
	assert.Contains(t, xmlStr, "openrtb")
}

// createTestAd creates a test Ad with InLine and Linear creative
func createTestAd() *model.Ad {
	return &model.Ad{
		ID: "test-ad",
		InLine: &model.InLine{
			AdSystem: &model.AdSystem{Value: "Test"},
			AdTitle:  "Test Ad",
			Creatives: &model.Creatives{
				Creative: []model.Creative{
					{
						ID: "creative1",
						Linear: &model.Linear{
							Duration: "",
						},
					},
				},
			},
		},
	}
}

func TestEnrich_ExistingExtensionsPreserved(t *testing.T) {
	enricher := NewEnricher()
	ad := createTestAd()
	ad.InLine.Extensions = &model.Extensions{
		Extension: []model.ExtensionXML{
			{Type: "existing", InnerXML: "<Data>preserved</Data>"},
		},
	}

	meta := vast.CanonicalMeta{
		Cats: []string{"IAB1"},
	}
	cfg := vast.ReceiverConfig{}

	warnings, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)
	assert.Empty(t, warnings)

	// Should have both existing and new extensions
	require.NotNil(t, ad.InLine.Extensions)
	assert.GreaterOrEqual(t, len(ad.InLine.Extensions.Extension), 2)

	// Check existing is preserved
	found := false
	for _, ext := range ad.InLine.Extensions.Extension {
		if ext.Type == "existing" {
			found = true
			assert.Contains(t, ext.InnerXML, "preserved")
		}
	}
	assert.True(t, found, "existing extension should be preserved")
}

func TestEnrich_DefaultCurrencyFallback(t *testing.T) {
	enricher := NewEnricher()
	ad := createTestAd()
	ad.InLine.Pricing = nil

	meta := vast.CanonicalMeta{
		Price:    5.0,
		Currency: "", // No currency in meta
	}
	cfg := vast.ReceiverConfig{
		DefaultCurrency: "GBP",
	}

	_, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)
	require.NotNil(t, ad.InLine.Pricing)
	assert.Equal(t, "GBP", ad.InLine.Pricing.Currency)
}

func TestEnrich_NoCurrencyDefaultsToUSD(t *testing.T) {
	enricher := NewEnricher()
	ad := createTestAd()
	ad.InLine.Pricing = nil

	meta := vast.CanonicalMeta{
		Price:    5.0,
		Currency: "", // No currency
	}
	cfg := vast.ReceiverConfig{
		DefaultCurrency: "", // No default either
	}

	_, err := enricher.Enrich(ad, meta, cfg)
	assert.NoError(t, err)
	require.NotNil(t, ad.InLine.Pricing)
	assert.Equal(t, "USD", ad.InLine.Pricing.Currency)
}
