package model

import (
	"strings"
	"testing"
)

// Mock config for testing ParseVastOrSkeleton
type mockReceiverConfig struct {
	allowSkeletonVast  bool
	vastVersionDefault string
}

func (m mockReceiverConfig) GetAllowSkeletonVast() bool {
	return m.allowSkeletonVast
}

func (m mockReceiverConfig) GetVastVersionDefault() string {
	return m.vastVersionDefault
}

func TestParseVastAdm_Success(t *testing.T) {
	tests := []struct {
		name        string
		adm         string
		checkResult func(*testing.T, *Vast)
	}{
		{
			name: "Valid VAST 3.0 with version",
			adm: `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="3.0">
  <Ad id="test1">
    <InLine>
      <AdSystem>TestSystem</AdSystem>
      <AdTitle>Test Ad</AdTitle>
    </InLine>
  </Ad>
</VAST>`,
			checkResult: func(t *testing.T, v *Vast) {
				if v.Version != "3.0" {
					t.Errorf("Expected version 3.0, got %s", v.Version)
				}
				if len(v.Ads) != 1 {
					t.Fatalf("Expected 1 ad, got %d", len(v.Ads))
				}
				if v.Ads[0].ID != "test1" {
					t.Errorf("Expected ad ID test1, got %s", v.Ads[0].ID)
				}
			},
		},
		{
			name: "Valid VAST without version attribute",
			adm: `<VAST>
  <Ad id="ad1">
    <InLine>
      <AdSystem>System</AdSystem>
      <AdTitle>Title</AdTitle>
    </InLine>
  </Ad>
</VAST>`,
			checkResult: func(t *testing.T, v *Vast) {
				// Empty version is allowed
				if len(v.Ads) != 1 {
					t.Fatalf("Expected 1 ad, got %d", len(v.Ads))
				}
			},
		},
		{
			name: "VAST with Pricing",
			adm: `<VAST version="4.0">
  <Ad id="ad1">
    <InLine>
      <AdSystem>System</AdSystem>
      <AdTitle>Title</AdTitle>
      <Pricing model="CPM" currency="USD">5.50</Pricing>
    </InLine>
  </Ad>
</VAST>`,
			checkResult: func(t *testing.T, v *Vast) {
				if v.Ads[0].InLine == nil {
					t.Fatal("Expected InLine element")
				}
				if v.Ads[0].InLine.Pricing == nil {
					t.Fatal("Expected Pricing element")
				}
				if v.Ads[0].InLine.Pricing.Model != "CPM" {
					t.Errorf("Expected pricing model CPM, got %s", v.Ads[0].InLine.Pricing.Model)
				}
				if v.Ads[0].InLine.Pricing.Currency != "USD" {
					t.Errorf("Expected currency USD, got %s", v.Ads[0].InLine.Pricing.Currency)
				}
				if v.Ads[0].InLine.Pricing.Value != "5.50" {
					t.Errorf("Expected value 5.50, got %s", v.Ads[0].InLine.Pricing.Value)
				}
			},
		},
		{
			name: "VAST with Creatives and Duration",
			adm: `<VAST version="3.0">
  <Ad id="ad1">
    <InLine>
      <AdSystem>System</AdSystem>
      <AdTitle>Title</AdTitle>
      <Creatives>
        <Creative id="cr1">
          <Linear>
            <Duration>00:00:30</Duration>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`,
			checkResult: func(t *testing.T, v *Vast) {
				if v.Ads[0].InLine == nil || v.Ads[0].InLine.Creatives == nil {
					t.Fatal("Expected InLine with Creatives")
				}
				if len(v.Ads[0].InLine.Creatives.Creatives) != 1 {
					t.Fatalf("Expected 1 creative, got %d", len(v.Ads[0].InLine.Creatives.Creatives))
				}
				creative := v.Ads[0].InLine.Creatives.Creatives[0]
				if creative.Linear == nil {
					t.Fatal("Expected Linear element")
				}
				if creative.Linear.Duration != "00:00:30" {
					t.Errorf("Expected duration 00:00:30, got %s", creative.Linear.Duration)
				}
			},
		},
		{
			name: "VAST with Extensions",
			adm: `<VAST version="3.0">
  <Ad id="ad1">
    <InLine>
      <AdSystem>System</AdSystem>
      <AdTitle>Title</AdTitle>
      <Extensions>
        <Extension type="prebid">
          <BidPrice>5.50</BidPrice>
        </Extension>
      </Extensions>
    </InLine>
  </Ad>
</VAST>`,
			checkResult: func(t *testing.T, v *Vast) {
				if v.Ads[0].InLine == nil || v.Ads[0].InLine.Extensions == nil {
					t.Fatal("Expected InLine with Extensions")
				}
				if len(v.Ads[0].InLine.Extensions.Extensions) != 1 {
					t.Fatalf("Expected 1 extension, got %d", len(v.Ads[0].InLine.Extensions.Extensions))
				}
				ext := v.Ads[0].InLine.Extensions.Extensions[0]
				if ext.Type != "prebid" {
					t.Errorf("Expected type prebid, got %s", ext.Type)
				}
			},
		},
		{
			name: "VAST with unknown elements preserved",
			adm: `<VAST version="3.0">
  <Ad id="ad1">
    <InLine>
      <AdSystem>System</AdSystem>
      <AdTitle>Title</AdTitle>
      <CustomElement>Custom Data</CustomElement>
    </InLine>
  </Ad>
</VAST>`,
			checkResult: func(t *testing.T, v *Vast) {
				if v.Ads[0].InLine == nil {
					t.Fatal("Expected InLine element")
				}
				// InnerXML should preserve the custom element
				if v.Ads[0].InLine.InnerXML == "" {
					t.Error("Expected InnerXML to preserve custom element")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vast, err := ParseVastAdm(tt.adm)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
			if vast == nil {
				t.Fatal("Expected non-nil VAST")
			}
			tt.checkResult(t, vast)
		})
	}
}

func TestParseVastAdm_Errors(t *testing.T) {
	tests := []struct {
		name        string
		adm         string
		expectError string
	}{
		{
			name:        "Missing VAST tag",
			adm:         `<Something>Not VAST</Something>`,
			expectError: "does not contain <VAST tag",
		},
		{
			name:        "Empty string",
			adm:         "",
			expectError: "does not contain <VAST tag",
		},
		{
			name:        "Invalid XML",
			adm:         `<VAST version="3.0"><Ad><InLine></VAST>`,
			expectError: "failed to parse VAST XML",
		},
		{
			name:        "Malformed XML",
			adm:         `<VAST version="3.0" unclosed`,
			expectError: "failed to parse VAST XML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vast, err := ParseVastAdm(tt.adm)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if vast != nil {
				t.Errorf("Expected nil VAST on error, got: %+v", vast)
			}
			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing %q, got: %v", tt.expectError, err)
			}
		})
	}
}

func TestParseVastOrSkeleton_ParseSuccess(t *testing.T) {
	adm := `<VAST version="3.0">
  <Ad id="test1">
    <InLine>
      <AdSystem>TestSystem</AdSystem>
      <AdTitle>Test Ad</AdTitle>
    </InLine>
  </Ad>
</VAST>`

	cfg := mockReceiverConfig{
		allowSkeletonVast:  false,
		vastVersionDefault: "3.0",
	}

	vast, warnings, err := ParseVastOrSkeleton(adm, cfg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if vast == nil {
		t.Fatal("Expected non-nil VAST")
	}
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings, got: %v", warnings)
	}
	if vast.Version != "3.0" {
		t.Errorf("Expected version 3.0, got %s", vast.Version)
	}
	if len(vast.Ads) != 1 {
		t.Errorf("Expected 1 ad, got %d", len(vast.Ads))
	}
}

func TestParseVastOrSkeleton_FallbackToSkeleton(t *testing.T) {
	tests := []struct {
		name               string
		adm                string
		vastVersionDefault string
	}{
		{
			name:               "Invalid XML with fallback",
			adm:                `<VAST><Invalid>XML`,
			vastVersionDefault: "3.0",
		},
		{
			name:               "No VAST tag with fallback",
			adm:                `<Something>Not VAST</Something>`,
			vastVersionDefault: "4.0",
		},
		{
			name:               "Empty string with fallback",
			adm:                "",
			vastVersionDefault: "3.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := mockReceiverConfig{
				allowSkeletonVast:  true,
				vastVersionDefault: tt.vastVersionDefault,
			}

			vast, warnings, err := ParseVastOrSkeleton(tt.adm, cfg)

			if err != nil {
				t.Fatalf("Expected no error with skeleton fallback, got: %v", err)
			}
			if vast == nil {
				t.Fatal("Expected non-nil VAST skeleton")
			}
			if len(warnings) == 0 {
				t.Error("Expected warnings about skeleton fallback")
			}
			if !strings.Contains(warnings[0], "failed to parse VAST") {
				t.Errorf("Expected warning about parse failure, got: %s", warnings[0])
			}

			// Verify skeleton structure
			if vast.Version != tt.vastVersionDefault {
				t.Errorf("Expected skeleton version %s, got %s", tt.vastVersionDefault, vast.Version)
			}
			if len(vast.Ads) != 1 {
				t.Errorf("Expected skeleton with 1 ad, got %d", len(vast.Ads))
			}
		})
	}
}

func TestParseVastOrSkeleton_ErrorWhenSkeletonDisabled(t *testing.T) {
	tests := []struct {
		name string
		adm  string
	}{
		{
			name: "Invalid XML no fallback",
			adm:  `<VAST><Invalid>XML`,
		},
		{
			name: "No VAST tag no fallback",
			adm:  `<Something>Not VAST</Something>`,
		},
		{
			name: "Empty string no fallback",
			adm:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := mockReceiverConfig{
				allowSkeletonVast:  false,
				vastVersionDefault: "3.0",
			}

			vast, warnings, err := ParseVastOrSkeleton(tt.adm, cfg)

			if err == nil {
				t.Fatal("Expected error when skeleton disabled, got nil")
			}
			if vast != nil {
				t.Errorf("Expected nil VAST on error, got: %+v", vast)
			}
			if len(warnings) != 0 {
				t.Errorf("Expected no warnings on error, got: %v", warnings)
			}
			if !strings.Contains(err.Error(), "skeleton fallback disabled") {
				t.Errorf("Expected error about disabled fallback, got: %v", err)
			}
		})
	}
}

func TestParseVastOrSkeleton_PreservesParseSuccess(t *testing.T) {
	// Test that successful parse doesn't trigger skeleton even if allowed
	adm := `<VAST version="4.0">
  <Ad id="test1">
    <InLine>
      <AdSystem>TestSystem</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <Pricing model="CPM" currency="EUR">10.00</Pricing>
    </InLine>
  </Ad>
</VAST>`

	cfg := mockReceiverConfig{
		allowSkeletonVast:  true, // Even though skeleton is allowed
		vastVersionDefault: "3.0",
	}

	vast, warnings, err := ParseVastOrSkeleton(adm, cfg)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if vast == nil {
		t.Fatal("Expected non-nil VAST")
	}
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings on successful parse, got: %v", warnings)
	}

	// Verify we got the parsed VAST, not a skeleton
	if vast.Version != "4.0" {
		t.Errorf("Expected parsed version 4.0, got %s (might be skeleton)", vast.Version)
	}
	if vast.Ads[0].InLine == nil || vast.Ads[0].InLine.Pricing == nil {
		t.Error("Expected parsed VAST with Pricing, might have gotten skeleton instead")
	}
}
