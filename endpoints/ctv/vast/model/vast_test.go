package model

import (
	"testing"
)

func TestParseVAST(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.0">
  <Ad id="test-ad-1">
    <InLine>
      <AdSystem version="1.0">TestSystem</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <Impression><![CDATA[http://example.com/impression]]></Impression>
      <Creatives>
        <Creative id="creative-1">
          <Linear>
            <Duration>00:00:30</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080">
                <![CDATA[http://example.com/video.mp4]]>
              </MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	vast, err := ParseString(xml)
	if err != nil {
		t.Fatalf("Failed to parse VAST: %v", err)
	}

	if vast.Version != "4.0" {
		t.Errorf("Expected version 4.0, got %s", vast.Version)
	}

	if len(vast.Ad) != 1 {
		t.Fatalf("Expected 1 ad, got %d", len(vast.Ad))
	}

	ad := vast.Ad[0]
	if ad.ID != "test-ad-1" {
		t.Errorf("Expected ad id 'test-ad-1', got '%s'", ad.ID)
	}

	if ad.InLine == nil {
		t.Fatal("Expected InLine element")
	}

	if ad.InLine.AdTitle != "Test Ad" {
		t.Errorf("Expected AdTitle 'Test Ad', got '%s'", ad.InLine.AdTitle)
	}
}

func TestMarshalVAST(t *testing.T) {
	vast := &VAST{
		Version: "4.0",
		Ad: []*Ad{
			{
				ID: "test-ad",
				InLine: &InLine{
					AdSystem: &AdSystem{
						Version: "1.0",
						Value:   "TestSystem",
					},
					AdTitle: "Test Ad",
					Impression: []Impression{
						{Value: "http://example.com/impression"},
					},
				},
			},
		},
	}

	_, err := vast.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal VAST: %v", err)
	}
}

func TestNewEmptyVAST(t *testing.T) {
	vast := NewEmptyVAST("3.0")
	if vast.Version != "3.0" {
		t.Errorf("Expected version 3.0, got %s", vast.Version)
	}

	if !vast.IsEmpty() {
		t.Error("Expected empty VAST")
	}

	// Test default version
	vast2 := NewEmptyVAST("")
	if vast2.Version != "4.0" {
		t.Errorf("Expected default version 4.0, got %s", vast2.Version)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  int
		expected string
	}{
		{0, "00:00:00"},
		{30, "00:00:30"},
		{90, "00:01:30"},
		{3661, "01:01:01"},
		{-5, "00:00:00"}, // Negative should become 0
	}

	for _, tt := range tests {
		result := FormatDuration(tt.seconds)
		if result != tt.expected {
			t.Errorf("FormatDuration(%d) = %s; expected %s", tt.seconds, result, tt.expected)
		}
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		duration string
		expected int
		hasError bool
	}{
		{"00:00:00", 0, false},
		{"00:00:30", 30, false},
		{"00:01:30", 90, false},
		{"01:01:01", 3661, false},
		{"invalid", 0, true},
		{"30", 0, true},
	}

	for _, tt := range tests {
		result, err := ParseDuration(tt.duration)
		if tt.hasError {
			if err == nil {
				t.Errorf("ParseDuration(%s) expected error but got none", tt.duration)
			}
		} else {
			if err != nil {
				t.Errorf("ParseDuration(%s) unexpected error: %v", tt.duration, err)
			}
			if result != tt.expected {
				t.Errorf("ParseDuration(%s) = %d; expected %d", tt.duration, result, tt.expected)
			}
		}
	}
}

func TestAddAd(t *testing.T) {
	vast := NewEmptyVAST("4.0")
	
	ad := &Ad{
		ID: "test-ad",
		InLine: &InLine{
			AdTitle: "Test",
		},
	}
	
	vast.AddAd(ad)
	
	if vast.IsEmpty() {
		t.Error("Expected VAST to have ads after AddAd")
	}
	
	if len(vast.Ad) != 1 {
		t.Errorf("Expected 1 ad, got %d", len(vast.Ad))
	}
}

func TestGetFirstAd(t *testing.T) {
	vast := NewEmptyVAST("4.0")
	
	// Empty VAST
	if vast.GetFirstAd() != nil {
		t.Error("Expected nil for empty VAST")
	}
	
	// With ad
	ad := &Ad{ID: "first"}
	vast.AddAd(ad)
	
	firstAd := vast.GetFirstAd()
	if firstAd == nil {
		t.Fatal("Expected ad, got nil")
	}
	
	if firstAd.ID != "first" {
		t.Errorf("Expected first ad id 'first', got '%s'", firstAd.ID)
	}
}

func TestNormalizeWhitespace(t *testing.T) {
	input := `
	
	  <VAST>
	    <Ad>
	    </Ad>
	  </VAST>
	  
	`
	
	expected := "<VAST>\n<Ad>\n</Ad>\n</VAST>"
	
	result := NormalizeWhitespace(input)
	if result != expected {
		t.Errorf("NormalizeWhitespace mismatch.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestVASTWithPricing(t *testing.T) {
	vast := &VAST{
		Version: "4.0",
		Ad: []*Ad{
			{
				ID: "test-ad",
				InLine: &InLine{
					AdSystem: &AdSystem{Value: "TestSystem"},
					AdTitle:  "Test Ad",
					Pricing: &Pricing{
						Model:    "CPM",
						Currency: "USD",
						Value:    "5.00",
					},
				},
			},
		},
	}

	data, err := vast.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal VAST with pricing: %v", err)
	}

	// Parse it back
	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse marshaled VAST: %v", err)
	}

	if parsed.Ad[0].InLine.Pricing == nil {
		t.Fatal("Expected pricing in parsed VAST")
	}

	pricing := parsed.Ad[0].InLine.Pricing
	if pricing.Model != "CPM" || pricing.Currency != "USD" || pricing.Value != "5.00" {
		t.Errorf("Pricing mismatch: got %+v", pricing)
	}
}

func TestVASTWithAdvertiser(t *testing.T) {
	vast := &VAST{
		Version: "4.0",
		Ad: []*Ad{
			{
				ID: "test-ad",
				InLine: &InLine{
					AdSystem:   &AdSystem{Value: "TestSystem"},
					AdTitle:    "Test Ad",
					Advertiser: "Example Corp",
				},
			},
		},
	}

	data, err := vast.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal VAST with advertiser: %v", err)
	}

	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse marshaled VAST: %v", err)
	}

	if parsed.Ad[0].InLine.Advertiser != "Example Corp" {
		t.Errorf("Expected advertiser 'Example Corp', got '%s'", parsed.Ad[0].InLine.Advertiser)
	}
}

func TestVASTWithCategories(t *testing.T) {
	vast := &VAST{
		Version: "4.0",
		Ad: []*Ad{
			{
				ID: "test-ad",
				InLine: &InLine{
					AdSystem: &AdSystem{Value: "TestSystem"},
					AdTitle:  "Test Ad",
					Category: []Category{
						{Authority: "IAB", Value: "IAB1-1"},
						{Authority: "IAB", Value: "IAB1-2"},
					},
				},
			},
		},
	}

	data, err := vast.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal VAST with categories: %v", err)
	}

	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse marshaled VAST: %v", err)
	}

	if len(parsed.Ad[0].InLine.Category) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(parsed.Ad[0].InLine.Category))
	}
}
