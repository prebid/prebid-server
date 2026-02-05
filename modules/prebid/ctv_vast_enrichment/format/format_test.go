package format

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment"
	"github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFormatter(t *testing.T) {
	formatter := NewFormatter()
	assert.NotNil(t, formatter)
}

func TestFormat_EmptyAds_ReturnsNoAdVast(t *testing.T) {
	formatter := NewFormatter()
	cfg := vast.ReceiverConfig{
		VastVersionDefault: "4.0",
	}

	xmlBytes, warnings, err := formatter.Format([]vast.EnrichedAd{}, cfg)
	require.NoError(t, err)
	assert.Empty(t, warnings)

	expected := loadGolden(t, "no_ad.xml")
	assertXMLEqual(t, expected, xmlBytes)
}

func TestFormat_SingleAd(t *testing.T) {
	formatter := NewFormatter()
	cfg := vast.ReceiverConfig{
		VastVersionDefault: "4.0",
	}

	ads := []vast.EnrichedAd{
		{
			Ad:       createTestAd("bid-123", "TestAdServer", "Test Ad", "advertiser.com", "5.5", "00:00:30", "creative1", "https://example.com/video.mp4", []string{"IAB1"}),
			Meta:     vast.CanonicalMeta{BidID: "bid-123"},
			Sequence: 1,
		},
	}

	xmlBytes, warnings, err := formatter.Format(ads, cfg)
	require.NoError(t, err)
	assert.Empty(t, warnings)

	expected := loadGolden(t, "single_ad.xml")
	assertXMLEqual(t, expected, xmlBytes)
}

func TestFormat_PodWithTwoAds(t *testing.T) {
	formatter := NewFormatter()
	cfg := vast.ReceiverConfig{
		VastVersionDefault: "4.0",
	}

	ads := []vast.EnrichedAd{
		{
			Ad:       createTestAd("bid-001", "TestAdServer", "First Ad", "first.com", "10", "00:00:15", "creative1", "https://example.com/first.mp4", nil),
			Meta:     vast.CanonicalMeta{BidID: "bid-001"},
			Sequence: 1,
		},
		{
			Ad:       createTestAd("bid-002", "TestAdServer", "Second Ad", "second.com", "8", "00:00:30", "creative2", "https://example.com/second.mp4", nil),
			Meta:     vast.CanonicalMeta{BidID: "bid-002"},
			Sequence: 2,
		},
	}

	xmlBytes, warnings, err := formatter.Format(ads, cfg)
	require.NoError(t, err)
	assert.Empty(t, warnings)

	expected := loadGolden(t, "pod_two_ads.xml")
	assertXMLEqual(t, expected, xmlBytes)
}

func TestFormat_PodWithThreeAds(t *testing.T) {
	formatter := NewFormatter()
	cfg := vast.ReceiverConfig{
		VastVersionDefault: "4.0",
	}

	ads := []vast.EnrichedAd{
		{
			Ad:       createMinimalAd("bid-alpha", "AdServer1", "Alpha Ad", "15", "USD", "00:00:10"),
			Meta:     vast.CanonicalMeta{BidID: "bid-alpha"},
			Sequence: 1,
		},
		{
			Ad:       createMinimalAd("bid-beta", "AdServer2", "Beta Ad", "12", "EUR", "00:00:20"),
			Meta:     vast.CanonicalMeta{BidID: "bid-beta"},
			Sequence: 2,
		},
		{
			Ad:       createMinimalAd("bid-gamma", "AdServer3", "Gamma Ad", "9", "USD", "00:00:15"),
			Meta:     vast.CanonicalMeta{BidID: "bid-gamma"},
			Sequence: 3,
		},
	}

	xmlBytes, warnings, err := formatter.Format(ads, cfg)
	require.NoError(t, err)
	assert.Empty(t, warnings)

	expected := loadGolden(t, "pod_three_ads.xml")
	assertXMLEqual(t, expected, xmlBytes)
}

func TestFormat_NilAdsInList(t *testing.T) {
	formatter := NewFormatter()
	cfg := vast.ReceiverConfig{
		VastVersionDefault: "4.0",
	}

	ads := []vast.EnrichedAd{
		{
			Ad:       nil, // nil ad
			Meta:     vast.CanonicalMeta{BidID: "bid-nil"},
			Sequence: 1,
		},
		{
			Ad:       createMinimalAd("bid-valid", "AdServer", "Valid Ad", "5", "USD", "00:00:15"),
			Meta:     vast.CanonicalMeta{BidID: "bid-valid"},
			Sequence: 2,
		},
	}

	xmlBytes, warnings, err := formatter.Format(ads, cfg)
	require.NoError(t, err)
	assert.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "skipping nil ad")

	// Should still produce valid VAST with the non-nil ad
	xmlStr := string(xmlBytes)
	assert.Contains(t, xmlStr, `<VAST version="4.0">`)
	assert.Contains(t, xmlStr, `<Ad id="bid-valid"`)
}

func TestFormat_AllNilAds_ReturnsNoAd(t *testing.T) {
	formatter := NewFormatter()
	cfg := vast.ReceiverConfig{
		VastVersionDefault: "4.0",
	}

	ads := []vast.EnrichedAd{
		{Ad: nil, Meta: vast.CanonicalMeta{BidID: "bid1"}, Sequence: 1},
		{Ad: nil, Meta: vast.CanonicalMeta{BidID: "bid2"}, Sequence: 2},
	}

	xmlBytes, warnings, err := formatter.Format(ads, cfg)
	require.NoError(t, err)
	assert.Len(t, warnings, 3) // 2 for nil ads, 1 for returning no-ad

	expected := loadGolden(t, "no_ad.xml")
	assertXMLEqual(t, expected, xmlBytes)
}

func TestFormat_DefaultVersion(t *testing.T) {
	formatter := NewFormatter()
	cfg := vast.ReceiverConfig{
		VastVersionDefault: "", // empty
	}

	ads := []vast.EnrichedAd{}

	xmlBytes, _, err := formatter.Format(ads, cfg)
	require.NoError(t, err)

	xmlStr := string(xmlBytes)
	assert.Contains(t, xmlStr, `version="4.0"`) // defaults to 4.0
}

func TestFormat_Version30(t *testing.T) {
	formatter := NewFormatter()
	cfg := vast.ReceiverConfig{
		VastVersionDefault: "3.0",
	}

	ads := []vast.EnrichedAd{}

	xmlBytes, _, err := formatter.Format(ads, cfg)
	require.NoError(t, err)

	xmlStr := string(xmlBytes)
	assert.Contains(t, xmlStr, `version="3.0"`)
}

func TestFormat_AdIDFromBidID(t *testing.T) {
	formatter := NewFormatter()
	cfg := vast.ReceiverConfig{
		VastVersionDefault: "4.0",
	}

	ads := []vast.EnrichedAd{
		{
			Ad:   createMinimalAd("", "AdServer", "Test", "5", "USD", "00:00:15"),
			Meta: vast.CanonicalMeta{BidID: "my-bid-id"},
		},
	}

	xmlBytes, _, err := formatter.Format(ads, cfg)
	require.NoError(t, err)

	xmlStr := string(xmlBytes)
	assert.Contains(t, xmlStr, `id="my-bid-id"`)
}

func TestFormat_AdIDFallbackToImpID(t *testing.T) {
	formatter := NewFormatter()
	cfg := vast.ReceiverConfig{
		VastVersionDefault: "4.0",
	}

	ads := []vast.EnrichedAd{
		{
			Ad:   createMinimalAd("", "AdServer", "Test", "5", "USD", "00:00:15"),
			Meta: vast.CanonicalMeta{BidID: "", ImpID: "imp-456"},
		},
	}

	xmlBytes, _, err := formatter.Format(ads, cfg)
	require.NoError(t, err)

	xmlStr := string(xmlBytes)
	assert.Contains(t, xmlStr, `id="imp-imp-456"`)
}

func TestFormat_SingleAdNoSequence(t *testing.T) {
	formatter := NewFormatter()
	cfg := vast.ReceiverConfig{
		VastVersionDefault: "4.0",
	}

	ads := []vast.EnrichedAd{
		{
			Ad:       createMinimalAd("", "AdServer", "Single", "5", "USD", "00:00:15"),
			Meta:     vast.CanonicalMeta{BidID: "bid-single"},
			Sequence: 1, // even with sequence set
		},
	}

	xmlBytes, _, err := formatter.Format(ads, cfg)
	require.NoError(t, err)

	xmlStr := string(xmlBytes)
	// Single ad should NOT have sequence attribute
	assert.NotContains(t, xmlStr, `sequence=`)
}

func TestFormat_PodHasSequence(t *testing.T) {
	formatter := NewFormatter()
	cfg := vast.ReceiverConfig{
		VastVersionDefault: "4.0",
	}

	ads := []vast.EnrichedAd{
		{
			Ad:       createMinimalAd("", "AdServer", "First", "5", "USD", "00:00:15"),
			Meta:     vast.CanonicalMeta{BidID: "bid-1"},
			Sequence: 1,
		},
		{
			Ad:       createMinimalAd("", "AdServer", "Second", "4", "USD", "00:00:15"),
			Meta:     vast.CanonicalMeta{BidID: "bid-2"},
			Sequence: 2,
		},
	}

	xmlBytes, _, err := formatter.Format(ads, cfg)
	require.NoError(t, err)

	xmlStr := string(xmlBytes)
	// Pod ads should have sequence attributes
	assert.Contains(t, xmlStr, `sequence="1"`)
	assert.Contains(t, xmlStr, `sequence="2"`)
}

func TestFormat_PreservesTracking(t *testing.T) {
	formatter := NewFormatter()
	cfg := vast.ReceiverConfig{
		VastVersionDefault: "4.0",
	}

	// Create ad with tracking events
	ad := createMinimalAd("", "AdServer", "WithTracking", "5", "USD", "00:00:15")
	ad.InLine.Creatives.Creative[0].Linear.TrackingEvents = &model.TrackingEvents{
		Tracking: []model.Tracking{
			{Event: "start", Value: "https://tracker.example.com/start"},
			{Event: "complete", Value: "https://tracker.example.com/complete"},
		},
	}

	ads := []vast.EnrichedAd{
		{
			Ad:   ad,
			Meta: vast.CanonicalMeta{BidID: "bid-track"},
		},
	}

	xmlBytes, _, err := formatter.Format(ads, cfg)
	require.NoError(t, err)

	xmlStr := string(xmlBytes)
	assert.Contains(t, xmlStr, `<Tracking event="start">`)
	assert.Contains(t, xmlStr, "https://tracker.example.com/start")
	assert.Contains(t, xmlStr, `<Tracking event="complete">`)
	assert.Contains(t, xmlStr, "https://tracker.example.com/complete")
}

func TestFormat_PreservesExtensions(t *testing.T) {
	formatter := NewFormatter()
	cfg := vast.ReceiverConfig{
		VastVersionDefault: "4.0",
	}

	ad := createMinimalAd("", "AdServer", "WithExtensions", "5", "USD", "00:00:15")
	ad.InLine.Extensions = &model.Extensions{
		Extension: []model.ExtensionXML{
			{Type: "openrtb", InnerXML: "<BidID>abc123</BidID><Seat>bidder1</Seat>"},
			{Type: "custom", InnerXML: "<Data>custom data</Data>"},
		},
	}

	ads := []vast.EnrichedAd{
		{
			Ad:   ad,
			Meta: vast.CanonicalMeta{BidID: "bid-ext"},
		},
	}

	xmlBytes, _, err := formatter.Format(ads, cfg)
	require.NoError(t, err)

	xmlStr := string(xmlBytes)
	assert.Contains(t, xmlStr, `<Extension type="openrtb">`)
	assert.Contains(t, xmlStr, "<BidID>abc123</BidID>")
	assert.Contains(t, xmlStr, `<Extension type="custom">`)
	assert.Contains(t, xmlStr, "<Data>custom data</Data>")
}

func TestDeriveAdID(t *testing.T) {
	tests := []struct {
		name     string
		meta     vast.CanonicalMeta
		expected string
	}{
		{
			name:     "with BidID",
			meta:     vast.CanonicalMeta{BidID: "bid-123"},
			expected: "bid-123",
		},
		{
			name:     "BidID takes precedence over ImpID",
			meta:     vast.CanonicalMeta{BidID: "bid-456", ImpID: "imp-789"},
			expected: "bid-456",
		},
		{
			name:     "fallback to ImpID when BidID empty",
			meta:     vast.CanonicalMeta{BidID: "", ImpID: "imp-123"},
			expected: "imp-imp-123",
		},
		{
			name:     "both empty",
			meta:     vast.CanonicalMeta{BidID: "", ImpID: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveAdID(tt.meta)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper functions

func createTestAd(id, adSystem, adTitle, advertiser, price, duration, creativeID, mediaURL string, categories []string) *model.Ad {
	ad := &model.Ad{
		ID: id,
		InLine: &model.InLine{
			AdSystem:   &model.AdSystem{Value: adSystem},
			AdTitle:    adTitle,
			Advertiser: advertiser,
			Pricing: &model.Pricing{
				Model:    "CPM",
				Currency: "USD",
				Value:    price,
			},
			Creatives: &model.Creatives{
				Creative: []model.Creative{
					{
						ID: creativeID,
						Linear: &model.Linear{
							Duration: duration,
							MediaFiles: &model.MediaFiles{
								MediaFile: []model.MediaFile{
									{
										Delivery: "progressive",
										Type:     "video/mp4",
										Width:    1920,
										Height:   1080,
										Value:    mediaURL,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if len(categories) > 0 {
		var catXML string
		for _, cat := range categories {
			catXML += "<Category>" + cat + "</Category>"
		}
		ad.InLine.Extensions = &model.Extensions{
			Extension: []model.ExtensionXML{
				{Type: "iab_category", InnerXML: catXML},
			},
		}
	}

	return ad
}

func createMinimalAd(id, adSystem, adTitle, price, currency, duration string) *model.Ad {
	return &model.Ad{
		ID: id,
		InLine: &model.InLine{
			AdSystem: &model.AdSystem{Value: adSystem},
			AdTitle:  adTitle,
			Pricing: &model.Pricing{
				Model:    "CPM",
				Currency: currency,
				Value:    price,
			},
			Creatives: &model.Creatives{
				Creative: []model.Creative{
					{
						Linear: &model.Linear{
							Duration: duration,
						},
					},
				},
			},
		},
	}
}

func loadGolden(t *testing.T, filename string) []byte {
	t.Helper()
	path := filepath.Join("testdata", filename)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read golden file: %s", path)
	return data
}

// assertXMLEqual compares two XML documents by normalizing whitespace.
func assertXMLEqual(t *testing.T, expected, actual []byte) {
	t.Helper()
	expectedNorm := normalizeXML(string(expected))
	actualNorm := normalizeXML(string(actual))
	assert.Equal(t, expectedNorm, actualNorm)
}

// normalizeXML normalizes XML for comparison by trimming whitespace.
func normalizeXML(xml string) string {
	// Split into lines and trim each
	lines := strings.Split(xml, "\n")
	var normalized []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return strings.Join(normalized, "\n")
}
