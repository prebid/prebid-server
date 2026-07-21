package model

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecToHHMMSS(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int
		expected string
	}{
		{"zero", 0, "00:00:00"},
		{"negative", -5, "00:00:00"},
		{"30 seconds", 30, "00:00:30"},
		{"1 minute", 60, "00:01:00"},
		{"1 minute 30 seconds", 90, "00:01:30"},
		{"1 hour", 3600, "01:00:00"},
		{"1 hour 30 minutes 45 seconds", 5445, "01:30:45"},
		{"2 hours", 7200, "02:00:00"},
		{"typical ad 15 seconds", 15, "00:00:15"},
		{"typical ad 30 seconds", 30, "00:00:30"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SecToHHMMSS(tt.seconds)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildNoAdVast(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{"default version", ""},
		{"version 3.0", "3.0"},
		{"version 4.0", "4.0"},
		{"version 4.2", "4.2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildNoAdVast(tt.version)
			require.NotEmpty(t, result)

			// Should contain XML header
			assert.True(t, strings.HasPrefix(string(result), "<?xml"))

			// Should contain VAST element
			assert.Contains(t, string(result), "<VAST")

			// Should have version attribute
			expectedVersion := tt.version
			if expectedVersion == "" {
				expectedVersion = "3.0"
			}
			assert.Contains(t, string(result), `version="`+expectedVersion+`"`)

			// Should be valid XML that can be unmarshalled
			var vast Vast
			err := xml.Unmarshal(result, &vast)
			assert.NoError(t, err)
			assert.Empty(t, vast.Ads)
		})
	}
}

func TestBuildSkeletonInlineVast(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{"default version", ""},
		{"version 3.0", "3.0"},
		{"version 4.0", "4.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vast := BuildSkeletonInlineVast(tt.version)
			require.NotNil(t, vast)

			expectedVersion := tt.version
			if expectedVersion == "" {
				expectedVersion = "3.0"
			}
			assert.Equal(t, expectedVersion, vast.Version)

			require.Len(t, vast.Ads, 1)
			ad := vast.Ads[0]
			assert.Equal(t, "1", ad.ID)
			assert.Equal(t, 1, ad.Sequence)

			require.NotNil(t, ad.InLine)
			assert.Equal(t, "Ad", ad.InLine.AdTitle)
			require.NotNil(t, ad.InLine.AdSystem)
			assert.Equal(t, "PBS-CTV", ad.InLine.AdSystem.Value)

			require.NotNil(t, ad.InLine.Creatives)
			require.Len(t, ad.InLine.Creatives.Creative, 1)
			creative := ad.InLine.Creatives.Creative[0]
			assert.Equal(t, "1", creative.ID)
			assert.Equal(t, 1, creative.Sequence)

			require.NotNil(t, creative.Linear)
			assert.Equal(t, "00:00:00", creative.Linear.Duration)
		})
	}
}

func TestBuildSkeletonInlineVastWithDuration(t *testing.T) {
	vast := BuildSkeletonInlineVastWithDuration("4.0", 30)
	require.NotNil(t, vast)
	assert.Equal(t, "4.0", vast.Version)

	require.Len(t, vast.Ads, 1)
	require.NotNil(t, vast.Ads[0].InLine)
	require.NotNil(t, vast.Ads[0].InLine.Creatives)
	require.Len(t, vast.Ads[0].InLine.Creatives.Creative, 1)
	require.NotNil(t, vast.Ads[0].InLine.Creatives.Creative[0].Linear)
	assert.Equal(t, "00:00:30", vast.Ads[0].InLine.Creatives.Creative[0].Linear.Duration)
}

func TestVast_Marshal(t *testing.T) {
	vast := &Vast{
		Version: "3.0",
		Ads: []Ad{
			{
				ID:       "ad1",
				Sequence: 1,
				InLine: &InLine{
					AdSystem: &AdSystem{
						Version: "1.0",
						Value:   "TestSystem",
					},
					AdTitle:    "Test Ad",
					Advertiser: "Test Advertiser",
					Pricing: &Pricing{
						Model:    "cpm",
						Currency: "USD",
						Value:    "5.00",
					},
					Creatives: &Creatives{
						Creative: []Creative{
							{
								ID:       "creative1",
								Sequence: 1,
								Linear: &Linear{
									Duration: "00:00:30",
									MediaFiles: &MediaFiles{
										MediaFile: []MediaFile{
											{
												Delivery: "progressive",
												Type:     "video/mp4",
												Width:    1920,
												Height:   1080,
												Bitrate:  5000,
												Value:    "https://example.com/video.mp4",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	output, err := vast.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, output)

	xmlStr := string(output)
	assert.Contains(t, xmlStr, `<?xml version="1.0" encoding="UTF-8"?>`)
	assert.Contains(t, xmlStr, `<VAST version="3.0">`)
	assert.Contains(t, xmlStr, `<Ad id="ad1" sequence="1">`)
	assert.Contains(t, xmlStr, `<InLine>`)
	assert.Contains(t, xmlStr, `<AdSystem version="1.0">TestSystem</AdSystem>`)
	assert.Contains(t, xmlStr, `<AdTitle>Test Ad</AdTitle>`)
	assert.Contains(t, xmlStr, `<Advertiser>Test Advertiser</Advertiser>`)
	assert.Contains(t, xmlStr, `<Pricing model="cpm" currency="USD">5.00</Pricing>`)
	assert.Contains(t, xmlStr, `<Duration>00:00:30</Duration>`)
	assert.Contains(t, xmlStr, `</VAST>`)
}

func TestVast_MarshalCompact(t *testing.T) {
	vast := BuildSkeletonInlineVast("3.0")
	output, err := vast.MarshalCompact()
	require.NoError(t, err)
	require.NotEmpty(t, output)

	xmlStr := string(output)
	// Compact should not have newlines in the body
	assert.Contains(t, xmlStr, `<VAST version="3.0"><Ad`)
}

func TestUnmarshal(t *testing.T) {
	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<VAST version="3.0">
  <Ad id="test-ad" sequence="1">
    <InLine>
      <AdSystem version="2.0">TestAdServer</AdSystem>
      <AdTitle>Sample Ad</AdTitle>
      <Advertiser>Sample Inc</Advertiser>
      <Pricing model="cpm" currency="EUR">10.50</Pricing>
      <Creatives>
        <Creative id="c1" sequence="1">
          <Linear>
            <Duration>00:00:15</Duration>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`)

	vast, err := Unmarshal(xmlData)
	require.NoError(t, err)
	require.NotNil(t, vast)

	assert.Equal(t, "3.0", vast.Version)
	require.Len(t, vast.Ads, 1)

	ad := vast.Ads[0]
	assert.Equal(t, "test-ad", ad.ID)
	assert.Equal(t, 1, ad.Sequence)

	require.NotNil(t, ad.InLine)
	assert.Equal(t, "Sample Ad", ad.InLine.AdTitle)
	assert.Equal(t, "Sample Inc", ad.InLine.Advertiser)

	require.NotNil(t, ad.InLine.AdSystem)
	assert.Equal(t, "2.0", ad.InLine.AdSystem.Version)
	assert.Equal(t, "TestAdServer", ad.InLine.AdSystem.Value)

	require.NotNil(t, ad.InLine.Pricing)
	assert.Equal(t, "cpm", ad.InLine.Pricing.Model)
	assert.Equal(t, "EUR", ad.InLine.Pricing.Currency)
	assert.Equal(t, "10.50", ad.InLine.Pricing.Value)

	require.NotNil(t, ad.InLine.Creatives)
	require.Len(t, ad.InLine.Creatives.Creative, 1)
	creative := ad.InLine.Creatives.Creative[0]
	assert.Equal(t, "c1", creative.ID)

	require.NotNil(t, creative.Linear)
	assert.Equal(t, "00:00:15", creative.Linear.Duration)
}

func TestUnmarshal_WithExtensions(t *testing.T) {
	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.0">
  <Ad id="ad1">
    <InLine>
      <AdTitle>Ad with Extensions</AdTitle>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:30</Duration>
          </Linear>
        </Creative>
      </Creatives>
      <Extensions>
        <Extension type="waterfall">
          <CustomData>some value</CustomData>
        </Extension>
        <Extension type="prebid">
          <BidInfo>test</BidInfo>
        </Extension>
      </Extensions>
    </InLine>
  </Ad>
</VAST>`)

	vast, err := Unmarshal(xmlData)
	require.NoError(t, err)
	require.NotNil(t, vast)
	require.Len(t, vast.Ads, 1)
	require.NotNil(t, vast.Ads[0].InLine)
	require.NotNil(t, vast.Ads[0].InLine.Extensions)
	require.Len(t, vast.Ads[0].InLine.Extensions.Extension, 2)

	ext1 := vast.Ads[0].InLine.Extensions.Extension[0]
	assert.Equal(t, "waterfall", ext1.Type)
	assert.Contains(t, ext1.InnerXML, "CustomData")

	ext2 := vast.Ads[0].InLine.Extensions.Extension[1]
	assert.Equal(t, "prebid", ext2.Type)
	assert.Contains(t, ext2.InnerXML, "BidInfo")
}

func TestUnmarshal_WrapperAd(t *testing.T) {
	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<VAST version="3.0">
  <Ad id="wrapper-ad">
    <Wrapper>
      <AdSystem>Wrapper System</AdSystem>
      <VASTAdTagURI><![CDATA[https://example.com/vast.xml]]></VASTAdTagURI>
      <Impression><![CDATA[https://example.com/track]]></Impression>
    </Wrapper>
  </Ad>
</VAST>`)

	vast, err := Unmarshal(xmlData)
	require.NoError(t, err)
	require.NotNil(t, vast)
	require.Len(t, vast.Ads, 1)

	ad := vast.Ads[0]
	assert.Equal(t, "wrapper-ad", ad.ID)
	assert.Nil(t, ad.InLine)
	require.NotNil(t, ad.Wrapper)
	assert.Equal(t, "Wrapper System", ad.Wrapper.AdSystem.Value)
}

func TestRoundTrip(t *testing.T) {
	original := &Vast{
		Version: "4.0",
		Ads: []Ad{
			{
				ID:       "roundtrip-test",
				Sequence: 1,
				InLine: &InLine{
					AdSystem: &AdSystem{Value: "PBS"},
					AdTitle:  "Round Trip Test",
					Creatives: &Creatives{
						Creative: []Creative{
							{
								ID: "c1",
								Linear: &Linear{
									Duration: "00:00:15",
								},
							},
						},
					},
				},
			},
		},
	}

	// Marshal
	xmlBytes, err := original.Marshal()
	require.NoError(t, err)

	// Unmarshal
	parsed, err := Unmarshal(xmlBytes)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, original.Version, parsed.Version)
	require.Len(t, parsed.Ads, 1)
	assert.Equal(t, original.Ads[0].ID, parsed.Ads[0].ID)
	assert.Equal(t, original.Ads[0].InLine.AdTitle, parsed.Ads[0].InLine.AdTitle)
}

func TestMediaFileWithCDATA(t *testing.T) {
	vast := &Vast{
		Version: "3.0",
		Ads: []Ad{
			{
				ID: "media-test",
				InLine: &InLine{
					AdTitle: "Media Test",
					Creatives: &Creatives{
						Creative: []Creative{
							{
								Linear: &Linear{
									Duration: "00:00:30",
									MediaFiles: &MediaFiles{
										MediaFile: []MediaFile{
											{
												Delivery: "progressive",
												Type:     "video/mp4",
												Width:    1280,
												Height:   720,
												Value:    "https://example.com/video.mp4?param=value&other=123",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	output, err := vast.Marshal()
	require.NoError(t, err)

	// MediaFile URL should be in CDATA
	xmlStr := string(output)
	assert.Contains(t, xmlStr, "<![CDATA[https://example.com/video.mp4?param=value&other=123]]>")
}

func TestTrackingEvents(t *testing.T) {
	vast := &Vast{
		Version: "3.0",
		Ads: []Ad{
			{
				ID: "tracking-test",
				InLine: &InLine{
					AdTitle: "Tracking Test",
					Creatives: &Creatives{
						Creative: []Creative{
							{
								Linear: &Linear{
									Duration: "00:00:30",
									TrackingEvents: &TrackingEvents{
										Tracking: []Tracking{
											{Event: "start", Value: "https://example.com/start"},
											{Event: "firstQuartile", Value: "https://example.com/q1"},
											{Event: "midpoint", Value: "https://example.com/mid"},
											{Event: "thirdQuartile", Value: "https://example.com/q3"},
											{Event: "complete", Value: "https://example.com/complete"},
											{Event: "progress", Offset: "00:00:05", Value: "https://example.com/5sec"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	output, err := vast.Marshal()
	require.NoError(t, err)

	xmlStr := string(output)
	assert.Contains(t, xmlStr, `event="start"`)
	assert.Contains(t, xmlStr, `event="complete"`)
	assert.Contains(t, xmlStr, `event="progress"`)
	assert.Contains(t, xmlStr, `offset="00:00:05"`)
}
