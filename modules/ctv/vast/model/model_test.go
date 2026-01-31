package model

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestSecToHHMMSS(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int
		expected string
	}{
		{"Zero", 0, "00:00:00"},
		{"30 seconds", 30, "00:00:30"},
		{"1 minute", 60, "00:01:00"},
		{"1 minute 30 seconds", 90, "00:01:30"},
		{"1 hour", 3600, "01:00:00"},
		{"1 hour 1 minute 1 second", 3661, "01:01:01"},
		{"Complex duration", 7384, "02:03:04"},
		{"Negative becomes zero", -10, "00:00:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SecToHHMMSS(tt.seconds)
			if result != tt.expected {
				t.Errorf("SecToHHMMSS(%d) = %s; want %s", tt.seconds, result, tt.expected)
			}
		})
	}
}

func TestBuildNoAdVast(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{"Default version", ""},
		{"Version 3.0", "3.0"},
		{"Version 4.0", "4.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xml := BuildNoAdVast(tt.version)
			
			if len(xml) == 0 {
				t.Error("Expected non-empty XML")
			}
			
			xmlStr := string(xml)
			if !strings.Contains(xmlStr, "<VAST") {
				t.Error("Expected <VAST tag in output")
			}
			if !strings.Contains(xmlStr, "version=") {
				t.Error("Expected version attribute in output")
			}
			
			expectedVersion := tt.version
			if expectedVersion == "" {
				expectedVersion = "3.0"
			}
			if !strings.Contains(xmlStr, expectedVersion) {
				t.Errorf("Expected version %s in output, got: %s", expectedVersion, xmlStr)
			}
		})
	}
}

func TestBuildSkeletonInlineVast(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{"Default version", ""},
		{"Version 3.0", "3.0"},
		{"Version 4.0", "4.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vast := BuildSkeletonInlineVast(tt.version)
			
			if vast == nil {
				t.Fatal("Expected non-nil VAST")
			}
			
			expectedVersion := tt.version
			if expectedVersion == "" {
				expectedVersion = "3.0"
			}
			if vast.Version != expectedVersion {
				t.Errorf("Expected version %s, got %s", expectedVersion, vast.Version)
			}
			
			if len(vast.Ads) != 1 {
				t.Fatalf("Expected 1 ad, got %d", len(vast.Ads))
			}
			
			ad := vast.Ads[0]
			if ad.ID != "1" {
				t.Errorf("Expected ad ID '1', got %s", ad.ID)
			}
			if ad.Sequence != 1 {
				t.Errorf("Expected ad sequence 1, got %d", ad.Sequence)
			}
			
			if ad.InLine == nil {
				t.Fatal("Expected InLine element")
			}
			if ad.InLine.AdSystem != "Prebid" {
				t.Errorf("Expected AdSystem 'Prebid', got %s", ad.InLine.AdSystem)
			}
			if ad.InLine.AdTitle != "Ad" {
				t.Errorf("Expected AdTitle 'Ad', got %s", ad.InLine.AdTitle)
			}
			
			if ad.InLine.Creatives == nil {
				t.Fatal("Expected Creatives element")
			}
			if len(ad.InLine.Creatives.Creatives) != 1 {
				t.Fatalf("Expected 1 creative, got %d", len(ad.InLine.Creatives.Creatives))
			}
			
			creative := ad.InLine.Creatives.Creatives[0]
			if creative.Linear == nil {
				t.Fatal("Expected Linear element")
			}
			if creative.Linear.Duration != "00:00:00" {
				t.Errorf("Expected duration '00:00:00', got %s", creative.Linear.Duration)
			}
		})
	}
}

func TestVast_Marshal(t *testing.T) {
	vast := &Vast{
		Version: "3.0",
		Ads: []Ad{
			{
				ID:       "ad1",
				Sequence: 1,
				InLine: &InLine{
					AdSystem: "TestSystem",
					AdTitle:  "Test Ad",
					Pricing: &Pricing{
						Model:    "CPM",
						Currency: "USD",
						Value:    "5.50",
					},
					Creatives: &Creatives{
						Creatives: []Creative{
							{
								ID: "creative1",
								Linear: &Linear{
									Duration: "00:00:30",
								},
							},
						},
					},
				},
			},
		},
	}

	xmlBytes, err := xml.MarshalIndent(vast, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal VAST: %v", err)
	}

	xmlStr := string(xmlBytes)
	
	// Verify structure
	if !strings.Contains(xmlStr, `<VAST`) {
		t.Error("Expected <VAST tag")
	}
	if !strings.Contains(xmlStr, `version="3.0"`) {
		t.Error("Expected version attribute")
	}
	if !strings.Contains(xmlStr, `<Ad id="ad1"`) {
		t.Error("Expected Ad with id attribute")
	}
	if !strings.Contains(xmlStr, `<InLine>`) {
		t.Error("Expected InLine element")
	}
	if !strings.Contains(xmlStr, `<AdSystem>TestSystem</AdSystem>`) {
		t.Error("Expected AdSystem element")
	}
	if !strings.Contains(xmlStr, `<AdTitle>Test Ad</AdTitle>`) {
		t.Error("Expected AdTitle element")
	}
	if !strings.Contains(xmlStr, `<Pricing`) {
		t.Error("Expected Pricing element")
	}
	if !strings.Contains(xmlStr, `currency="USD"`) {
		t.Error("Expected currency attribute")
	}
	if !strings.Contains(xmlStr, `<Duration>00:00:30</Duration>`) {
		t.Error("Expected Duration element")
	}
}

func TestVast_Unmarshal(t *testing.T) {
	xmlStr := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="3.0">
  <Ad id="test1" sequence="1">
    <InLine>
      <AdSystem>TestSystem</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <Pricing model="CPM" currency="USD">5.50</Pricing>
      <Creatives>
        <Creative id="cr1">
          <Linear>
            <Duration>00:00:30</Duration>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	var vast Vast
	err := xml.Unmarshal([]byte(xmlStr), &vast)
	if err != nil {
		t.Fatalf("Failed to unmarshal VAST: %v", err)
	}

	if vast.Version != "3.0" {
		t.Errorf("Expected version 3.0, got %s", vast.Version)
	}
	if len(vast.Ads) != 1 {
		t.Fatalf("Expected 1 ad, got %d", len(vast.Ads))
	}

	ad := vast.Ads[0]
	if ad.ID != "test1" {
		t.Errorf("Expected ad ID test1, got %s", ad.ID)
	}
	if ad.Sequence != 1 {
		t.Errorf("Expected sequence 1, got %d", ad.Sequence)
	}

	if ad.InLine == nil {
		t.Fatal("Expected InLine element")
	}
	if ad.InLine.AdSystem != "TestSystem" {
		t.Errorf("Expected AdSystem TestSystem, got %s", ad.InLine.AdSystem)
	}
	if ad.InLine.AdTitle != "Test Ad" {
		t.Errorf("Expected AdTitle 'Test Ad', got %s", ad.InLine.AdTitle)
	}

	if ad.InLine.Pricing == nil {
		t.Fatal("Expected Pricing element")
	}
	if ad.InLine.Pricing.Model != "CPM" {
		t.Errorf("Expected pricing model CPM, got %s", ad.InLine.Pricing.Model)
	}
	if ad.InLine.Pricing.Currency != "USD" {
		t.Errorf("Expected currency USD, got %s", ad.InLine.Pricing.Currency)
	}
	if ad.InLine.Pricing.Value != "5.50" {
		t.Errorf("Expected value 5.50, got %s", ad.InLine.Pricing.Value)
	}
}

func TestVast_PreserveUnknownElements(t *testing.T) {
	xmlStr := `<VAST version="4.0">
  <Ad id="test1">
    <InLine>
      <AdSystem>TestSystem</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <CustomElement>Custom Data</CustomElement>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:30</Duration>
            <UnknownLinearElement>Value</UnknownLinearElement>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	var vast Vast
	err := xml.Unmarshal([]byte(xmlStr), &vast)
	if err != nil {
		t.Fatalf("Failed to unmarshal VAST with custom elements: %v", err)
	}

	// Verify InnerXML preserved unknown elements
	if vast.Ads[0].InLine.InnerXML == "" {
		t.Error("Expected InnerXML to preserve custom elements")
	}

	// Marshal back and verify custom elements are preserved
	xmlBytes, err := xml.Marshal(vast)
	if err != nil {
		t.Fatalf("Failed to marshal VAST: %v", err)
	}

	xmlOut := string(xmlBytes)
	if !strings.Contains(xmlOut, "CustomElement") {
		t.Error("Expected custom element to be preserved in output")
	}
}

func TestExtensions(t *testing.T) {
	vast := &Vast{
		Version: "3.0",
		Ads: []Ad{
			{
				ID: "ad1",
				InLine: &InLine{
					AdSystem: "Test",
					AdTitle:  "Test",
					Extensions: &Extensions{
						Extensions: []Extension{
							{
								Type:     "prebid",
								InnerXML: `<BidPrice currency="USD">5.50</BidPrice>`,
							},
							{
								Type:     "debug",
								InnerXML: `<DebugInfo>Test Debug</DebugInfo>`,
							},
						},
					},
				},
			},
		},
	}

	xmlBytes, err := xml.MarshalIndent(vast, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal VAST with extensions: %v", err)
	}

	xmlStr := string(xmlBytes)
	
	if !strings.Contains(xmlStr, `<Extensions>`) {
		t.Error("Expected Extensions element")
	}
	if !strings.Contains(xmlStr, `<Extension type="prebid">`) {
		t.Error("Expected Extension with type attribute")
	}
	if !strings.Contains(xmlStr, `<BidPrice currency="USD">5.50</BidPrice>`) {
		t.Error("Expected inner XML preserved in extension")
	}
}
