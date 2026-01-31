package vast

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	
	"github.com/prebid/prebid-server/v3/modules/ctv/vast/model"
)

func TestFormatter_Format_EmptyAds(t *testing.T) {
	formatter := NewFormatter()
	cfg := ReceiverConfig{
		VastVersionDefault: "3.0",
	}

	xml, warnings, err := formatter.Format([]*model.VastAd{}, cfg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings, got: %v", warnings)
	}
	if len(xml) == 0 {
		t.Error("Expected non-empty XML")
	}

	xmlStr := string(xml)
	if !strings.Contains(xmlStr, "<VAST") {
		t.Error("Expected <VAST tag in output")
	}
	if !strings.Contains(xmlStr, `version="3.0"`) {
		t.Error("Expected version 3.0 in output")
	}
}

func TestFormatter_Format_SingleAd(t *testing.T) {
	formatter := NewFormatter()
	cfg := ReceiverConfig{
		VastVersionDefault: "3.0",
		Receiver:           "GAM_SSU",
	}

	ad := &model.Ad{
		ID: "bid-12345",
		InLine: &model.InLine{
			AdSystem:   "TestBidder",
			AdTitle:    "Test Creative",
			Advertiser: "test.com",
			Pricing: &model.Pricing{
				Model:    "CPM",
				Currency: "USD",
				Value:    "5.50",
			},
			Creatives: &model.Creatives{
				Creatives: []model.Creative{
					{
						ID: "creative1",
						Linear: &model.Linear{
							Duration: "00:00:30",
						},
					},
				},
			},
		},
	}

	xml, warnings, err := formatter.Format([]*model.VastAd{ad}, cfg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings, got: %v", warnings)
	}

	xmlStr := string(xml)

	// Verify structure
	if !strings.Contains(xmlStr, `<VAST version="3.0">`) {
		t.Error("Expected VAST version 3.0")
	}
	if !strings.Contains(xmlStr, `<Ad id="bid-12345"`) {
		t.Error("Expected Ad with id bid-12345")
	}
	// Single ad should NOT have sequence attribute
	if strings.Contains(xmlStr, `sequence=`) {
		t.Error("Single ad should not have sequence attribute")
	}
	if !strings.Contains(xmlStr, `<AdSystem>TestBidder</AdSystem>`) {
		t.Error("Expected AdSystem")
	}
	if !strings.Contains(xmlStr, `<AdTitle>Test Creative</AdTitle>`) {
		t.Error("Expected AdTitle")
	}
	if !strings.Contains(xmlStr, `<Pricing model="CPM" currency="USD">5.50</Pricing>`) {
		t.Error("Expected Pricing element")
	}
}

func TestFormatter_Format_MultipleAds_Pod(t *testing.T) {
	formatter := NewFormatter()
	cfg := ReceiverConfig{
		VastVersionDefault: "4.0",
		Receiver:           "GAM_SSU",
		MaxAdsInPod:        3,
	}

	ads := []*model.VastAd{
		{
			ID: "bid-001",
			InLine: &model.InLine{
				AdSystem:   "Bidder1",
				AdTitle:    "Ad 1",
				Advertiser: "advertiser1.com",
				Pricing: &model.Pricing{
					Model:    "CPM",
					Currency: "USD",
					Value:    "10.00",
				},
				Creatives: &model.Creatives{
					Creatives: []model.Creative{
						{
							ID: "cr1",
							Linear: &model.Linear{
								Duration: "00:00:15",
							},
						},
					},
				},
			},
		},
		{
			ID: "bid-002",
			InLine: &model.InLine{
				AdSystem:   "Bidder2",
				AdTitle:    "Ad 2",
				Advertiser: "advertiser2.com",
				Pricing: &model.Pricing{
					Model:    "CPM",
					Currency: "USD",
					Value:    "8.50",
				},
				Creatives: &model.Creatives{
					Creatives: []model.Creative{
						{
							ID: "cr2",
							Linear: &model.Linear{
								Duration: "00:00:30",
							},
						},
					},
				},
			},
		},
		{
			ID: "bid-003",
			InLine: &model.InLine{
				AdSystem:   "Bidder3",
				AdTitle:    "Ad 3",
				Advertiser: "advertiser3.com",
				Pricing: &model.Pricing{
					Model:    "CPM",
					Currency: "EUR",
					Value:    "7.25",
				},
				Creatives: &model.Creatives{
					Creatives: []model.Creative{
						{
							ID: "cr3",
							Linear: &model.Linear{
								Duration: "00:00:30",
							},
						},
					},
				},
			},
		},
	}

	xml, warnings, err := formatter.Format(ads, cfg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings, got: %v", warnings)
	}

	xmlStr := string(xml)

	// Verify structure
	if !strings.Contains(xmlStr, `<VAST version="4.0">`) {
		t.Error("Expected VAST version 4.0")
	}

	// Verify all three ads are present
	if !strings.Contains(xmlStr, `<Ad id="bid-001" sequence="1">`) {
		t.Error("Expected Ad 1 with sequence 1")
	}
	if !strings.Contains(xmlStr, `<Ad id="bid-002" sequence="2">`) {
		t.Error("Expected Ad 2 with sequence 2")
	}
	if !strings.Contains(xmlStr, `<Ad id="bid-003" sequence="3">`) {
		t.Error("Expected Ad 3 with sequence 3")
	}

	// Verify each ad has its content
	if !strings.Contains(xmlStr, `<AdSystem>Bidder1</AdSystem>`) {
		t.Error("Expected Bidder1")
	}
	if !strings.Contains(xmlStr, `<AdSystem>Bidder2</AdSystem>`) {
		t.Error("Expected Bidder2")
	}
	if !strings.Contains(xmlStr, `<AdSystem>Bidder3</AdSystem>`) {
		t.Error("Expected Bidder3")
	}
}

func TestFormatter_Format_WithPresetSequence(t *testing.T) {
	formatter := NewFormatter()
	cfg := ReceiverConfig{
		VastVersionDefault: "3.0",
	}

	ads := []*model.VastAd{
		{
			ID:       "bid-001",
			Sequence: 2, // Pre-set sequence
			InLine: &model.InLine{
				AdSystem: "Bidder1",
				AdTitle:  "Ad 1",
			},
		},
		{
			ID:       "bid-002",
			Sequence: 1, // Pre-set sequence
			InLine: &model.InLine{
				AdSystem: "Bidder2",
				AdTitle:  "Ad 2",
			},
		},
	}

	xml, warnings, err := formatter.Format(ads, cfg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings, got: %v", warnings)
	}

	xmlStr := string(xml)

	// Verify pre-set sequences are preserved
	if !strings.Contains(xmlStr, `<Ad id="bid-001" sequence="2">`) {
		t.Error("Expected Ad 1 to preserve sequence 2")
	}
	if !strings.Contains(xmlStr, `<Ad id="bid-002" sequence="1">`) {
		t.Error("Expected Ad 2 to preserve sequence 1")
	}
}

func TestFormatter_Format_NilAd(t *testing.T) {
	formatter := NewFormatter()
	cfg := ReceiverConfig{
		VastVersionDefault: "3.0",
	}

	ads := []*model.VastAd{
		{
			ID: "bid-001",
			InLine: &model.InLine{
				AdSystem: "Bidder1",
				AdTitle:  "Ad 1",
			},
		},
		nil, // Nil ad
		{
			ID: "bid-003",
			InLine: &model.InLine{
				AdSystem: "Bidder3",
				AdTitle:  "Ad 3",
			},
		},
	}

	xml, warnings, err := formatter.Format(ads, cfg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(warnings) != 1 {
		t.Errorf("Expected 1 warning about nil ad, got: %v", warnings)
	}
	if !strings.Contains(warnings[0], "nil ad") {
		t.Errorf("Expected warning about nil ad, got: %s", warnings[0])
	}

	xmlStr := string(xml)

	// Should have 2 ads, not 3
	adCount := strings.Count(xmlStr, "<Ad id=")
	if adCount != 2 {
		t.Errorf("Expected 2 ads in output, found %d", adCount)
	}
}

func TestFormatter_Format_MissingInLine(t *testing.T) {
	formatter := NewFormatter()
	cfg := ReceiverConfig{
		VastVersionDefault: "3.0",
	}

	ads := []*model.VastAd{
		{
			ID:     "bid-001",
			InLine: nil, // Missing InLine
		},
	}

	xml, warnings, err := formatter.Format(ads, cfg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(warnings) != 1 {
		t.Errorf("Expected 1 warning about missing InLine, got: %v", warnings)
	}
	if !strings.Contains(warnings[0], "no InLine content") {
		t.Errorf("Expected warning about InLine, got: %s", warnings[0])
	}

	// Should still produce valid XML
	xmlStr := string(xml)
	if !strings.Contains(xmlStr, "<VAST") {
		t.Error("Expected valid VAST structure")
	}
}

func TestFormatter_Format_DefaultVersion(t *testing.T) {
	formatter := NewFormatter()
	cfg := ReceiverConfig{
		VastVersionDefault: "", // Empty version
	}

	ad := &model.Ad{
		ID: "bid-001",
		InLine: &model.InLine{
			AdSystem: "Bidder",
			AdTitle:  "Ad",
		},
	}

	xml, warnings, err := formatter.Format([]*model.VastAd{ad}, cfg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings, got: %v", warnings)
	}

	xmlStr := string(xml)

	// Should default to 3.0
	if !strings.Contains(xmlStr, `<VAST version="3.0">`) {
		t.Error("Expected default version 3.0")
	}
}

func TestFormatter_Format_PreservesExtensions(t *testing.T) {
	formatter := NewFormatter()
	cfg := ReceiverConfig{
		VastVersionDefault: "3.0",
	}

	ad := &model.Ad{
		ID: "bid-001",
		InLine: &model.InLine{
			AdSystem: "Bidder",
			AdTitle:  "Ad",
			Extensions: &model.Extensions{
				Extensions: []model.Extension{
					{
						Type:     "prebid",
						InnerXML: `<BidPrice>5.50</BidPrice><Debug>test</Debug>`,
					},
				},
			},
		},
	}

	xml, warnings, err := formatter.Format([]*model.VastAd{ad}, cfg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings, got: %v", warnings)
	}

	xmlStr := string(xml)

	// Verify Extensions are preserved
	if !strings.Contains(xmlStr, `<Extensions>`) {
		t.Error("Expected Extensions element")
	}
	if !strings.Contains(xmlStr, `<Extension type="prebid">`) {
		t.Error("Expected Extension with type")
	}
	if !strings.Contains(xmlStr, `<BidPrice>5.50</BidPrice>`) {
		t.Error("Expected BidPrice in extension")
	}
}

// Golden tests using testdata files
func TestFormatter_Format_Golden_SingleAd(t *testing.T) {
	formatter := NewFormatter()
	cfg := ReceiverConfig{
		VastVersionDefault: "3.0",
		Receiver:           "GAM_SSU",
	}

	ad := &model.Ad{
		ID: "bid-abc123",
		InLine: &model.InLine{
			AdSystem:   "PrebidBidder",
			AdTitle:    "Premium Video Ad",
			Advertiser: "example-advertiser.com",
			Pricing: &model.Pricing{
				Model:    "CPM",
				Currency: "USD",
				Value:    "12.50",
			},
			Creatives: &model.Creatives{
				Creatives: []model.Creative{
					{
						ID: "creative-456",
						Linear: &model.Linear{
							Duration: "00:00:30",
						},
					},
				},
			},
		},
	}

	xml, warnings, err := formatter.Format([]*model.VastAd{ad}, cfg)

	if err != nil {
		t.Fatalf("Failed to format: %v", err)
	}
	if len(warnings) != 0 {
		t.Logf("Warnings: %v", warnings)
	}

	// Save golden file if UPDATE_GOLDEN env var is set
	goldenPath := filepath.Join("testdata", "golden_single_ad.xml")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		os.MkdirAll("testdata", 0755)
		if err := os.WriteFile(goldenPath, xml, 0644); err != nil {
			t.Fatalf("Failed to update golden file: %v", err)
		}
		t.Log("Updated golden file:", goldenPath)
	}

	// Compare with golden file
	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("Failed to read golden file %s: %v (run with UPDATE_GOLDEN=1 to create)", goldenPath, err)
	}

	if string(xml) != string(expected) {
		t.Errorf("Output does not match golden file.\nGot:\n%s\n\nExpected:\n%s", string(xml), string(expected))
	}
}

func TestFormatter_Format_Golden_AdPod(t *testing.T) {
	formatter := NewFormatter()
	cfg := ReceiverConfig{
		VastVersionDefault: "4.0",
		Receiver:           "GAM_SSU",
		MaxAdsInPod:        3,
	}

	ads := []*model.VastAd{
		{
			ID:       "bid-001-xyz",
			Sequence: 1,
			InLine: &model.InLine{
				AdSystem:   "BidderA",
				AdTitle:    "Pre-Roll Ad",
				Advertiser: "advertiser-a.com",
				Pricing: &model.Pricing{
					Model:    "CPM",
					Currency: "USD",
					Value:    "15.00",
				},
				Creatives: &model.Creatives{
					Creatives: []model.Creative{
						{
							ID: "creative-a1",
							Linear: &model.Linear{
								Duration: "00:00:15",
							},
						},
					},
				},
			},
		},
		{
			ID:       "bid-002-xyz",
			Sequence: 2,
			InLine: &model.InLine{
				AdSystem:   "BidderB",
				AdTitle:    "Mid-Roll Ad",
				Advertiser: "advertiser-b.com",
				Pricing: &model.Pricing{
					Model:    "CPM",
					Currency: "USD",
					Value:    "12.00",
				},
				Creatives: &model.Creatives{
					Creatives: []model.Creative{
						{
							ID: "creative-b1",
							Linear: &model.Linear{
								Duration: "00:00:30",
							},
						},
					},
				},
			},
		},
		{
			ID:       "bid-003-xyz",
			Sequence: 3,
			InLine: &model.InLine{
				AdSystem:   "BidderC",
				AdTitle:    "Post-Roll Ad",
				Advertiser: "advertiser-c.com",
				Pricing: &model.Pricing{
					Model:    "CPM",
					Currency: "EUR",
					Value:    "10.50",
				},
				Creatives: &model.Creatives{
					Creatives: []model.Creative{
						{
							ID: "creative-c1",
							Linear: &model.Linear{
								Duration: "00:00:30",
							},
						},
					},
				},
			},
		},
	}

	xml, warnings, err := formatter.Format(ads, cfg)

	if err != nil {
		t.Fatalf("Failed to format: %v", err)
	}
	if len(warnings) != 0 {
		t.Logf("Warnings: %v", warnings)
	}

	// Save golden file if UPDATE_GOLDEN env var is set
	goldenPath := filepath.Join("testdata", "golden_ad_pod.xml")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		os.MkdirAll("testdata", 0755)
		if err := os.WriteFile(goldenPath, xml, 0644); err != nil {
			t.Fatalf("Failed to update golden file: %v", err)
		}
		t.Log("Updated golden file:", goldenPath)
	}

	// Compare with golden file
	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("Failed to read golden file %s: %v (run with UPDATE_GOLDEN=1 to create)", goldenPath, err)
	}

	if string(xml) != string(expected) {
		t.Errorf("Output does not match golden file.\nGot:\n%s\n\nExpected:\n%s", string(xml), string(expected))
	}
}
