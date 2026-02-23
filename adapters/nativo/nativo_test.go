package nativo

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestBidderNativo(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderNativo, config.Adapter{
		Endpoint: "https://foo.io/?src=prebid"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "nativotest", bidder)
}

func TestGetRendererMeta(t *testing.T) {
	tests := []struct {
		description string
		ext         *openrtb_ext.ExtRequest
		expected    *openrtb_ext.ExtBidPrebidMeta
	}{
		{
			description: "nil ext returns nil",
			ext:         nil,
			expected:    nil,
		},
		{
			description: "nil sdk returns nil",
			ext:         &openrtb_ext.ExtRequest{},
			expected:    nil,
		},
		{
			description: "empty renderers returns nil",
			ext: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Sdk: &openrtb_ext.ExtRequestSdk{},
				},
			},
			expected: nil,
		},
		{
			description: "NativoRenderer found returns meta with name and version",
			ext: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Sdk: &openrtb_ext.ExtRequestSdk{
						Renderers: []openrtb_ext.ExtRequestSdkRenderer{
							{Name: "NativoRenderer", Version: "1.0"},
						},
					},
				},
			},
			expected: &openrtb_ext.ExtBidPrebidMeta{
				RendererName:    "NativoRenderer",
				RendererVersion: "1.0",
			},
		},
		{
			description: "NativoRenderer matched case-insensitively",
			ext: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Sdk: &openrtb_ext.ExtRequestSdk{
						Renderers: []openrtb_ext.ExtRequestSdkRenderer{
							{Name: "nativorenderer", Version: "2.0"},
						},
					},
				},
			},
			expected: &openrtb_ext.ExtBidPrebidMeta{
				RendererName:    "nativorenderer",
				RendererVersion: "2.0",
			},
		},
		{
			description: "non-NativoRenderer returns nil",
			ext: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Sdk: &openrtb_ext.ExtRequestSdk{
						Renderers: []openrtb_ext.ExtRequestSdkRenderer{
							{Name: "OtherRenderer", Version: "1.0"},
						},
					},
				},
			},
			expected: nil,
		},
		{
			description: "NativoRenderer found among multiple renderers",
			ext: &openrtb_ext.ExtRequest{
				Prebid: openrtb_ext.ExtRequestPrebid{
					Sdk: &openrtb_ext.ExtRequestSdk{
						Renderers: []openrtb_ext.ExtRequestSdkRenderer{
							{Name: "OtherRenderer", Version: "1.0"},
							{Name: "NativoRenderer", Version: "3.0"},
						},
					},
				},
			},
			expected: &openrtb_ext.ExtBidPrebidMeta{
				RendererName:    "NativoRenderer",
				RendererVersion: "3.0",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			result := getRendererMeta(test.ext)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestGetMediaTypeForBid(t *testing.T) {
	tests := []struct {
		description  string
		bid          openrtb2.Bid
		expectedType openrtb_ext.BidType
		hasError     bool
	}{
		{
			description: "nil ext returns error",
			bid:         openrtb2.Bid{ImpID: "test-imp"},
			hasError:    true,
		},
		{
			description: "invalid ext JSON returns error",
			bid:         openrtb2.Bid{ImpID: "test-imp", Ext: json.RawMessage(`invalid`)},
			hasError:    true,
		},
		{
			description: "ext without prebid field returns error",
			bid:         openrtb2.Bid{ImpID: "test-imp", Ext: json.RawMessage(`{"other":"data"}`)},
			hasError:    true,
		},
		{
			description:  "valid banner type",
			bid:          openrtb2.Bid{ImpID: "test-imp", Ext: json.RawMessage(`{"prebid":{"type":"banner"}}`)},
			expectedType: openrtb_ext.BidTypeBanner,
		},
		{
			description:  "valid native type",
			bid:          openrtb2.Bid{ImpID: "test-imp", Ext: json.RawMessage(`{"prebid":{"type":"native"}}`)},
			expectedType: openrtb_ext.BidTypeNative,
		},
		{
			description:  "valid video type",
			bid:          openrtb2.Bid{ImpID: "test-imp", Ext: json.RawMessage(`{"prebid":{"type":"video"}}`)},
			expectedType: openrtb_ext.BidTypeVideo,
		},
		{
			description: "unknown bid type string returns error",
			bid:         openrtb2.Bid{ImpID: "test-imp", Ext: json.RawMessage(`{"prebid":{"type":"unknown"}}`)},
			hasError:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			result, err := getMediaTypeForBid(test.bid)
			if test.hasError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedType, result)
			}
		})
	}
}

func TestGetMediaTypeForImp(t *testing.T) {
	tests := []struct {
		description  string
		impID        string
		imps         []openrtb2.Imp
		expectedType openrtb_ext.BidType
		hasError     bool
	}{
		{
			description:  "banner imp",
			impID:        "test-imp",
			imps:         []openrtb2.Imp{{ID: "test-imp", Banner: &openrtb2.Banner{}}},
			expectedType: openrtb_ext.BidTypeBanner,
		},
		{
			description:  "video imp",
			impID:        "test-imp",
			imps:         []openrtb2.Imp{{ID: "test-imp", Video: &openrtb2.Video{}}},
			expectedType: openrtb_ext.BidTypeVideo,
		},
		{
			description:  "native imp",
			impID:        "test-imp",
			imps:         []openrtb2.Imp{{ID: "test-imp", Native: &openrtb2.Native{}}},
			expectedType: openrtb_ext.BidTypeNative,
		},
		{
			description: "imp not found returns error",
			impID:       "unknown-imp",
			imps:        []openrtb2.Imp{{ID: "test-imp", Banner: &openrtb2.Banner{}}},
			hasError:    true,
		},
		{
			description: "imp found but no media type set returns error",
			impID:       "test-imp",
			imps:        []openrtb2.Imp{{ID: "test-imp"}},
			hasError:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			result, err := getMediaTypeForImp(test.impID, test.imps)
			if test.hasError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedType, result)
			}
		})
	}
}
