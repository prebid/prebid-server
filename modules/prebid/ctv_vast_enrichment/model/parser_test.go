package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Sample VAST XML strings for testing
const (
	sampleVAST30 = `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="3.0">
  <Ad id="12345" sequence="1">
    <InLine>
      <AdSystem version="1.0">Test Ad Server</AdSystem>
      <AdTitle>Test Video Ad</AdTitle>
      <Advertiser>Test Advertiser Inc</Advertiser>
      <Impression id="imp1"><![CDATA[https://example.com/impression]]></Impression>
      <Creatives>
        <Creative id="creative1" sequence="1">
          <Linear>
            <Duration>00:00:30</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080" bitrate="5000">
                <![CDATA[https://example.com/video.mp4]]>
              </MediaFile>
            </MediaFiles>
            <VideoClicks>
              <ClickThrough><![CDATA[https://example.com/landing]]></ClickThrough>
            </VideoClicks>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	sampleVAST40 = `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.0">
  <Ad id="ad-40" sequence="1">
    <InLine>
      <AdSystem>PBS-CTV</AdSystem>
      <AdTitle>VAST 4.0 Test</AdTitle>
      <Pricing model="cpm" currency="USD">5.50</Pricing>
      <Creatives>
        <Creative id="c1">
          <UniversalAdId idRegistry="ad-id.org" idValue="8465">8465</UniversalAdId>
          <Linear>
            <Duration>00:00:15</Duration>
          </Linear>
        </Creative>
      </Creatives>
      <Extensions>
        <Extension type="waterfall">
          <WaterfallIndex>1</WaterfallIndex>
        </Extension>
      </Extensions>
    </InLine>
  </Ad>
</VAST>`

	sampleVASTWrapper = `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="3.0">
  <Ad id="wrapper-ad">
    <Wrapper>
      <AdSystem>Wrapper System</AdSystem>
      <VASTAdTagURI><![CDATA[https://example.com/vast.xml]]></VASTAdTagURI>
      <Impression><![CDATA[https://example.com/wrapper-impression]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <TrackingEvents>
              <Tracking event="start"><![CDATA[https://example.com/start]]></Tracking>
            </TrackingEvents>
          </Linear>
        </Creative>
      </Creatives>
    </Wrapper>
  </Ad>
</VAST>`

	sampleVASTNoVersion = `<?xml version="1.0" encoding="UTF-8"?>
<VAST>
  <Ad id="no-version">
    <InLine>
      <AdTitle>No Version Ad</AdTitle>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:10</Duration>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	sampleVASTMultipleAds = `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="3.0">
  <Ad id="ad1" sequence="1">
    <InLine>
      <AdTitle>First Ad</AdTitle>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:15</Duration>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
  <Ad id="ad2" sequence="2">
    <InLine>
      <AdTitle>Second Ad</AdTitle>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:30</Duration>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	sampleVASTMinimal = `<VAST version="3.0"><Ad id="1"><InLine><AdTitle>Min</AdTitle><Creatives><Creative><Linear><Duration>00:00:05</Duration></Linear></Creative></Creatives></InLine></Ad></VAST>`

	sampleVASTEmpty = `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="3.0">
</VAST>`

	invalidXML       = `<VAST version="3.0"><Ad><InLine><AdTitle>Broken`
	notVAST          = `<html><body>Not VAST</body></html>`
	emptyString      = ``
	justWhitespace   = `   `
)

func TestParseVastAdm_ValidVAST30(t *testing.T) {
	vast, err := ParseVastAdm(sampleVAST30)
	require.NoError(t, err)
	require.NotNil(t, vast)

	assert.Equal(t, "3.0", vast.Version)
	require.Len(t, vast.Ads, 1)

	ad := vast.Ads[0]
	assert.Equal(t, "12345", ad.ID)
	assert.Equal(t, 1, ad.Sequence)

	require.NotNil(t, ad.InLine)
	assert.Equal(t, "Test Video Ad", ad.InLine.AdTitle)
	assert.Equal(t, "Test Advertiser Inc", ad.InLine.Advertiser)

	require.NotNil(t, ad.InLine.AdSystem)
	assert.Equal(t, "Test Ad Server", ad.InLine.AdSystem.Value)
	assert.Equal(t, "1.0", ad.InLine.AdSystem.Version)

	require.NotNil(t, ad.InLine.Creatives)
	require.Len(t, ad.InLine.Creatives.Creative, 1)

	creative := ad.InLine.Creatives.Creative[0]
	assert.Equal(t, "creative1", creative.ID)

	require.NotNil(t, creative.Linear)
	assert.Equal(t, "00:00:30", creative.Linear.Duration)
}

func TestParseVastAdm_ValidVAST40WithExtensions(t *testing.T) {
	vast, err := ParseVastAdm(sampleVAST40)
	require.NoError(t, err)
	require.NotNil(t, vast)

	assert.Equal(t, "4.0", vast.Version)
	require.Len(t, vast.Ads, 1)

	ad := vast.Ads[0]
	require.NotNil(t, ad.InLine)

	// Check pricing
	require.NotNil(t, ad.InLine.Pricing)
	assert.Equal(t, "cpm", ad.InLine.Pricing.Model)
	assert.Equal(t, "USD", ad.InLine.Pricing.Currency)
	assert.Equal(t, "5.50", ad.InLine.Pricing.Value)

	// Check extensions
	require.NotNil(t, ad.InLine.Extensions)
	require.Len(t, ad.InLine.Extensions.Extension, 1)
	assert.Equal(t, "waterfall", ad.InLine.Extensions.Extension[0].Type)
	assert.Contains(t, ad.InLine.Extensions.Extension[0].InnerXML, "WaterfallIndex")

	// Check UniversalAdId
	require.NotNil(t, ad.InLine.Creatives)
	require.Len(t, ad.InLine.Creatives.Creative, 1)
	creative := ad.InLine.Creatives.Creative[0]
	require.NotNil(t, creative.UniversalAdID)
	assert.Equal(t, "ad-id.org", creative.UniversalAdID.IDRegistry)
	assert.Equal(t, "8465", creative.UniversalAdID.IDValue)
}

func TestParseVastAdm_WrapperAd(t *testing.T) {
	vast, err := ParseVastAdm(sampleVASTWrapper)
	require.NoError(t, err)
	require.NotNil(t, vast)

	require.Len(t, vast.Ads, 1)
	ad := vast.Ads[0]

	assert.Nil(t, ad.InLine)
	require.NotNil(t, ad.Wrapper)
	assert.Equal(t, "Wrapper System", ad.Wrapper.AdSystem.Value)

	assert.True(t, IsWrapperAd(&ad))
	assert.False(t, IsInLineAd(&ad))
}

func TestParseVastAdm_NoVersion(t *testing.T) {
	vast, err := ParseVastAdm(sampleVASTNoVersion)
	require.NoError(t, err)
	require.NotNil(t, vast)

	// Empty version is acceptable
	assert.Equal(t, "", vast.Version)
	require.Len(t, vast.Ads, 1)
	assert.Equal(t, "No Version Ad", vast.Ads[0].InLine.AdTitle)
}

func TestParseVastAdm_MultipleAds(t *testing.T) {
	vast, err := ParseVastAdm(sampleVASTMultipleAds)
	require.NoError(t, err)
	require.NotNil(t, vast)

	require.Len(t, vast.Ads, 2)
	assert.Equal(t, "ad1", vast.Ads[0].ID)
	assert.Equal(t, 1, vast.Ads[0].Sequence)
	assert.Equal(t, "ad2", vast.Ads[1].ID)
	assert.Equal(t, 2, vast.Ads[1].Sequence)
}

func TestParseVastAdm_MinimalVAST(t *testing.T) {
	vast, err := ParseVastAdm(sampleVASTMinimal)
	require.NoError(t, err)
	require.NotNil(t, vast)

	assert.Equal(t, "3.0", vast.Version)
	require.Len(t, vast.Ads, 1)
	assert.Equal(t, "00:00:05", vast.Ads[0].InLine.Creatives.Creative[0].Linear.Duration)
}

func TestParseVastAdm_EmptyVAST(t *testing.T) {
	vast, err := ParseVastAdm(sampleVASTEmpty)
	require.NoError(t, err)
	require.NotNil(t, vast)

	assert.Equal(t, "3.0", vast.Version)
	assert.Empty(t, vast.Ads)
}

func TestParseVastAdm_NotVAST(t *testing.T) {
	vast, err := ParseVastAdm(notVAST)
	assert.ErrorIs(t, err, ErrNotVAST)
	assert.Nil(t, vast)
}

func TestParseVastAdm_EmptyString(t *testing.T) {
	vast, err := ParseVastAdm(emptyString)
	assert.ErrorIs(t, err, ErrNotVAST)
	assert.Nil(t, vast)
}

func TestParseVastAdm_Whitespace(t *testing.T) {
	vast, err := ParseVastAdm(justWhitespace)
	assert.ErrorIs(t, err, ErrNotVAST)
	assert.Nil(t, vast)
}

func TestParseVastAdm_InvalidXML(t *testing.T) {
	vast, err := ParseVastAdm(invalidXML)
	assert.ErrorIs(t, err, ErrVASTParseFailure)
	assert.Nil(t, vast)
}

func TestParseVastOrSkeleton_Success(t *testing.T) {
	cfg := ParserConfig{
		AllowSkeletonVast:  true,
		VastVersionDefault: "3.0",
	}

	vast, warnings, err := ParseVastOrSkeleton(sampleVAST30, cfg)
	require.NoError(t, err)
	require.NotNil(t, vast)
	assert.Empty(t, warnings)
	assert.Equal(t, "3.0", vast.Version)
}

func TestParseVastOrSkeleton_FailWithSkeleton(t *testing.T) {
	cfg := ParserConfig{
		AllowSkeletonVast:  true,
		VastVersionDefault: "4.0",
	}

	vast, warnings, err := ParseVastOrSkeleton(notVAST, cfg)
	require.NoError(t, err)
	require.NotNil(t, vast)

	// Should return skeleton
	assert.Equal(t, "4.0", vast.Version)
	require.Len(t, vast.Ads, 1)
	assert.Equal(t, "PBS-CTV", vast.Ads[0].InLine.AdSystem.Value)

	// Should have warning
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "VAST parse failed")
}

func TestParseVastOrSkeleton_FailWithoutSkeleton(t *testing.T) {
	cfg := ParserConfig{
		AllowSkeletonVast:  false,
		VastVersionDefault: "3.0",
	}

	vast, warnings, err := ParseVastOrSkeleton(notVAST, cfg)
	assert.Error(t, err)
	assert.Nil(t, vast)
	assert.Empty(t, warnings)
}

func TestParseVastOrSkeleton_InvalidXMLWithSkeleton(t *testing.T) {
	cfg := ParserConfig{
		AllowSkeletonVast:  true,
		VastVersionDefault: "3.0",
	}

	vast, warnings, err := ParseVastOrSkeleton(invalidXML, cfg)
	require.NoError(t, err)
	require.NotNil(t, vast)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "VAST parse failed")
}

func TestParseVastOrSkeleton_DefaultVersion(t *testing.T) {
	cfg := ParserConfig{
		AllowSkeletonVast:  true,
		VastVersionDefault: "", // Should default to "3.0"
	}

	vast, _, err := ParseVastOrSkeleton(notVAST, cfg)
	require.NoError(t, err)
	require.NotNil(t, vast)
	assert.Equal(t, "3.0", vast.Version)
}

func TestParseVastFromBytes(t *testing.T) {
	data := []byte(sampleVASTMinimal)
	vast, err := ParseVastFromBytes(data)
	require.NoError(t, err)
	require.NotNil(t, vast)
	assert.Equal(t, "3.0", vast.Version)
}

func TestExtractFirstAd(t *testing.T) {
	tests := []struct {
		name     string
		vast     *Vast
		expectID string
		expectNil bool
	}{
		{
			name:      "nil vast",
			vast:      nil,
			expectNil: true,
		},
		{
			name:      "empty ads",
			vast:      &Vast{Ads: []Ad{}},
			expectNil: true,
		},
		{
			name: "single ad",
			vast: &Vast{Ads: []Ad{{ID: "first"}}},
			expectID: "first",
		},
		{
			name: "multiple ads",
			vast: &Vast{Ads: []Ad{{ID: "first"}, {ID: "second"}}},
			expectID: "first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ad := ExtractFirstAd(tt.vast)
			if tt.expectNil {
				assert.Nil(t, ad)
			} else {
				require.NotNil(t, ad)
				assert.Equal(t, tt.expectID, ad.ID)
			}
		})
	}
}

func TestExtractDuration(t *testing.T) {
	tests := []struct {
		name     string
		xml      string
		expected string
	}{
		{
			name:     "inline with duration",
			xml:      sampleVAST30,
			expected: "00:00:30",
		},
		{
			name:     "minimal vast",
			xml:      sampleVASTMinimal,
			expected: "00:00:05",
		},
		{
			name:     "empty vast",
			xml:      sampleVASTEmpty,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vast, err := ParseVastAdm(tt.xml)
			require.NoError(t, err)
			duration := ExtractDuration(vast)
			assert.Equal(t, tt.expected, duration)
		})
	}
}

func TestParseDurationToSeconds(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		expected int
	}{
		{"empty", "", 0},
		{"zero", "00:00:00", 0},
		{"5 seconds", "00:00:05", 5},
		{"30 seconds", "00:00:30", 30},
		{"1 minute", "00:01:00", 60},
		{"1 minute 30 seconds", "00:01:30", 90},
		{"1 hour", "01:00:00", 3600},
		{"1 hour 30 minutes 45 seconds", "01:30:45", 5445},
		{"with milliseconds", "00:00:30.500", 30},
		{"invalid format", "30", 0},
		{"invalid chars", "00:0a:30", 0},
		{"too few parts", "00:30", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseDurationToSeconds(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsInLineAd(t *testing.T) {
	assert.False(t, IsInLineAd(nil))
	assert.False(t, IsInLineAd(&Ad{}))
	assert.False(t, IsInLineAd(&Ad{Wrapper: &Wrapper{}}))
	assert.True(t, IsInLineAd(&Ad{InLine: &InLine{}}))
}

func TestIsWrapperAd(t *testing.T) {
	assert.False(t, IsWrapperAd(nil))
	assert.False(t, IsWrapperAd(&Ad{}))
	assert.False(t, IsWrapperAd(&Ad{InLine: &InLine{}}))
	assert.True(t, IsWrapperAd(&Ad{Wrapper: &Wrapper{}}))
}

func TestParseVastAdm_PreservesInnerXML(t *testing.T) {
	// Test that unknown elements are preserved via InnerXML
	customVAST := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="3.0">
  <Ad id="custom">
    <InLine>
      <AdTitle>Custom Ad</AdTitle>
      <CustomElement>Custom Value</CustomElement>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:15</Duration>
            <CustomLinearData>Some Data</CustomLinearData>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	vast, err := ParseVastAdm(customVAST)
	require.NoError(t, err)
	require.NotNil(t, vast)

	// InnerXML fields should contain the unknown elements
	require.Len(t, vast.Ads, 1)
	require.NotNil(t, vast.Ads[0].InLine)
	
	// The InnerXML on InLine should contain CustomElement
	assert.Contains(t, vast.Ads[0].InLine.InnerXML, "CustomElement")
}

func TestRoundTrip_ParseMarshalParse(t *testing.T) {
	// Parse original
	vast1, err := ParseVastAdm(sampleVAST30)
	require.NoError(t, err)

	// Marshal back to XML
	xml1, err := vast1.Marshal()
	require.NoError(t, err)

	// Parse again
	vast2, err := ParseVastAdm(string(xml1))
	require.NoError(t, err)

	// Compare key fields
	assert.Equal(t, vast1.Version, vast2.Version)
	require.Len(t, vast2.Ads, len(vast1.Ads))
	assert.Equal(t, vast1.Ads[0].ID, vast2.Ads[0].ID)
	assert.Equal(t, vast1.Ads[0].InLine.AdTitle, vast2.Ads[0].InLine.AdTitle)
}
