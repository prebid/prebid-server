package formatter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prebid/prebid-server/v3/endpoints/ctv/vast/model"
)

// TestGoldenFiles tests VAST formatting against golden files
func TestGoldenFiles(t *testing.T) {
	testDataDir := "../testdata"
	
	tests := []struct {
		name       string
		goldenFile string
		buildVAST  func() *model.VAST
		receiver   ReceiverProfile
	}{
		{
			name:       "Empty VAST",
			goldenFile: "empty_vast.xml",
			buildVAST: func() *model.VAST {
				return model.NewEmptyVAST("4.0")
			},
			receiver: ReceiverGeneric,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vast := tt.buildVAST()
			
			config := Config{
				Profile:        tt.receiver,
				DefaultVersion: vast.Version,
			}
			
			factory := NewFormatterFactory()
			formatter := factory.CreateFormatter(config)
			
			output, err := formatter.Format(vast)
			if err != nil {
				t.Fatalf("Format failed: %v", err)
			}
			
			// Read golden file
			goldenPath := filepath.Join(testDataDir, tt.goldenFile)
			expected, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("Failed to read golden file: %v", err)
			}
			
			// Normalize whitespace for comparison
			normalizedOutput := model.NormalizeWhitespace(string(output))
			normalizedExpected := model.NormalizeWhitespace(string(expected))
			
			if normalizedOutput != normalizedExpected {
				t.Errorf("Output doesn't match golden file.\nExpected:\n%s\n\nGot:\n%s", normalizedExpected, normalizedOutput)
				
				// Optionally update golden file in development
				if os.Getenv("UPDATE_GOLDEN") == "1" {
					err := os.WriteFile(goldenPath, output, 0644)
					if err != nil {
						t.Logf("Failed to update golden file: %v", err)
					} else {
						t.Log("Updated golden file")
					}
				}
			}
		})
	}
}

// TestVASTRoundTrip tests that VAST can be marshaled and unmarshaled without loss
func TestVASTRoundTrip(t *testing.T) {
	original := &model.VAST{
		Version: "4.0",
		Ad: []*model.Ad{
			{
				ID:       "test-ad",
				Sequence: 1,
				InLine: &model.InLine{
					AdSystem: &model.AdSystem{
						Version: "1.0",
						Value:   "TestSystem",
					},
					AdTitle:    "Test Ad",
					Advertiser: "Test Advertiser",
					Impression: []model.Impression{
						{ID: "imp1", Value: "http://example.com/imp"},
					},
					Pricing: &model.Pricing{
						Model:    "CPM",
						Currency: "USD",
						Value:    "5.50",
					},
					Category: []model.Category{
						{Authority: "IAB", Value: "IAB1-1"},
					},
					Creatives: &model.Creatives{
						Creative: []*model.Creative{
							{
								ID:       "creative1",
								Sequence: 1,
								Linear: &model.Linear{
									Duration: "00:00:30",
									MediaFiles: &model.MediaFiles{
										MediaFile: []model.MediaFile{
											{
												ID:       "media1",
												Delivery: "progressive",
												Type:     "video/mp4",
												Width:    1920,
												Height:   1080,
												Value:    "http://example.com/video.mp4",
											},
										},
									},
								},
							},
						},
					},
					Extensions: &model.Extensions{
						Extension: []model.Extension{
							{
								Type:     "prebid",
								InnerXML: `{"test":"value"}`,
							},
						},
					},
				},
			},
		},
	}

	// Marshal
	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	parsed, err := model.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify key fields
	if parsed.Version != original.Version {
		t.Errorf("Version mismatch: expected %s, got %s", original.Version, parsed.Version)
	}

	if len(parsed.Ad) != len(original.Ad) {
		t.Fatalf("Ad count mismatch: expected %d, got %d", len(original.Ad), len(parsed.Ad))
	}

	ad := parsed.Ad[0]
	origAd := original.Ad[0]

	if ad.ID != origAd.ID {
		t.Errorf("Ad ID mismatch: expected %s, got %s", origAd.ID, ad.ID)
	}

	if ad.Sequence != origAd.Sequence {
		t.Errorf("Ad Sequence mismatch: expected %d, got %d", origAd.Sequence, ad.Sequence)
	}

	if ad.InLine == nil {
		t.Fatal("InLine is nil")
	}

	if ad.InLine.AdTitle != origAd.InLine.AdTitle {
		t.Errorf("AdTitle mismatch: expected %s, got %s", origAd.InLine.AdTitle, ad.InLine.AdTitle)
	}

	if ad.InLine.Advertiser != origAd.InLine.Advertiser {
		t.Errorf("Advertiser mismatch: expected %s, got %s", origAd.InLine.Advertiser, ad.InLine.Advertiser)
	}

	if ad.InLine.Pricing == nil {
		t.Fatal("Pricing is nil")
	}

	if ad.InLine.Pricing.Value != origAd.InLine.Pricing.Value {
		t.Errorf("Pricing value mismatch: expected %s, got %s", origAd.InLine.Pricing.Value, ad.InLine.Pricing.Value)
	}
}

// TestVASTPreservesUnknownElements tests that unknown XML elements are preserved
func TestVASTPreservesUnknownElements(t *testing.T) {
	// VAST with custom/unknown elements in Linear
	vastXML := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.0">
  <Ad id="test">
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Test</AdTitle>
      <Impression>http://test.com</Impression>
      <Creatives>
        <Creative id="c1">
          <Linear>
            <Duration>00:00:30</Duration>
            <TrackingEvents>
              <Tracking event="start">http://test.com/start</Tracking>
            </TrackingEvents>
            <VideoClicks>
              <ClickThrough>http://test.com/click</ClickThrough>
            </VideoClicks>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4">http://test.com/video.mp4</MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	// Parse
	vast, err := model.ParseString(vastXML)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Marshal back
	output, err := vast.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	outputStr := string(output)

	// Verify tracking events are preserved
	if !strings.Contains(outputStr, "<TrackingEvents>") {
		t.Error("TrackingEvents not preserved")
	}

	if !strings.Contains(outputStr, `event="start"`) {
		t.Error("Tracking event not preserved")
	}

	// Verify video clicks are preserved
	if !strings.Contains(outputStr, "<VideoClicks>") {
		t.Error("VideoClicks not preserved")
	}

	if !strings.Contains(outputStr, "<ClickThrough>") {
		t.Error("ClickThrough not preserved")
	}
}
