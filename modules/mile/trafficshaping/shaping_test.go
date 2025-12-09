package trafficshaping

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestFilterBannerSizes_EdgeCases(t *testing.T) {
	t.Run("convert_w_h_to_format_when_not_allowed", func(t *testing.T) {
		allowedSizes := map[BannerSize]struct{}{
			{W: 300, H: 250}: {},
			{W: 728, H: 90}:  {},
		}

		imp := &openrtb2.Imp{
			Banner: &openrtb2.Banner{
				W: ptrutil.ToPtr[int64](160),
				H: ptrutil.ToPtr[int64](600),
			},
		}

		filterBannerSizes(imp, allowedSizes)

		// Should convert to format array with allowed sizes
		assert.Nil(t, imp.Banner.W)
		assert.Nil(t, imp.Banner.H)
		assert.Greater(t, len(imp.Banner.Format), 0)
		// Verify formats contain allowed sizes
		hasAllowedSize := false
		for _, format := range imp.Banner.Format {
			if format.W == 300 && format.H == 250 {
				hasAllowedSize = true
				break
			}
		}
		assert.True(t, hasAllowedSize)
	})

	t.Run("keep_w_h_when_allowed", func(t *testing.T) {
		allowedSizes := map[BannerSize]struct{}{
			{W: 300, H: 250}: {},
		}

		imp := &openrtb2.Imp{
			Banner: &openrtb2.Banner{
				W: ptrutil.ToPtr[int64](300),
				H: ptrutil.ToPtr[int64](250),
			},
		}

		filterBannerSizes(imp, allowedSizes)

		// Should keep w/h when size is allowed
		assert.NotNil(t, imp.Banner.W)
		assert.NotNil(t, imp.Banner.H)
		assert.Equal(t, int64(300), *imp.Banner.W)
		assert.Equal(t, int64(250), *imp.Banner.H)
	})

	t.Run("empty_allowed_sizes", func(t *testing.T) {
		allowedSizes := map[BannerSize]struct{}{}

		imp := &openrtb2.Imp{
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					{W: 300, H: 250},
				},
			},
		}

		originalLen := len(imp.Banner.Format)
		filterBannerSizes(imp, allowedSizes)

		// Should not modify when allowedSizes is empty
		assert.Equal(t, originalLen, len(imp.Banner.Format))
	})

	t.Run("nil_banner", func(t *testing.T) {
		allowedSizes := map[BannerSize]struct{}{
			{W: 300, H: 250}: {},
		}

		imp := &openrtb2.Imp{
			Banner: nil,
		}

		// Should not panic
		filterBannerSizes(imp, allowedSizes)
		assert.Nil(t, imp.Banner)
	})

	t.Run("nil_imp", func(t *testing.T) {
		allowedSizes := map[BannerSize]struct{}{
			{W: 300, H: 250}: {},
		}

		var imp *openrtb2.Imp

		// Should not panic
		filterBannerSizes(imp, allowedSizes)
	})

	t.Run("w_h_to_format_empty_allowed", func(t *testing.T) {
		allowedSizes := map[BannerSize]struct{}{}

		imp := &openrtb2.Imp{
			Banner: &openrtb2.Banner{
				W: ptrutil.ToPtr[int64](160),
				H: ptrutil.ToPtr[int64](600),
			},
		}

		expectedW := imp.Banner.W
		expectedH := imp.Banner.H

		filterBannerSizes(imp, allowedSizes)

		// Should not modify when allowedSizes is empty
		assert.Equal(t, expectedW, imp.Banner.W)
		assert.Equal(t, expectedH, imp.Banner.H)
	})
}

func TestGetAllowedBidders(t *testing.T) {
	t.Run("nil_config", func(t *testing.T) {
		result := getAllowedBidders("test-gpid", nil)
		assert.Nil(t, result)
	})

	t.Run("missing_gpid", func(t *testing.T) {
		config := &ShapingConfig{
			GPIDRules: map[string]*GPIDRule{
				"other-gpid": {
					AllowedBidders: map[string]struct{}{
						"rubicon": {},
					},
				},
			},
		}

		result := getAllowedBidders("test-gpid", config)
		assert.Nil(t, result)
	})

	t.Run("valid_gpid", func(t *testing.T) {
		config := &ShapingConfig{
			GPIDRules: map[string]*GPIDRule{
				"test-gpid": {
					AllowedBidders: map[string]struct{}{
						"rubicon":  {},
						"appnexus": {},
					},
				},
			},
		}

		result := getAllowedBidders("test-gpid", config)
		assert.NotNil(t, result)
		assert.Equal(t, 2, len(result))
		_, hasRubicon := result["rubicon"]
		_, hasAppnexus := result["appnexus"]
		assert.True(t, hasRubicon)
		assert.True(t, hasAppnexus)
	})

	t.Run("empty_allowed_bidders", func(t *testing.T) {
		config := &ShapingConfig{
			GPIDRules: map[string]*GPIDRule{
				"test-gpid": {
					AllowedBidders: map[string]struct{}{},
				},
			},
		}

		result := getAllowedBidders("test-gpid", config)
		assert.NotNil(t, result)
		assert.Equal(t, 0, len(result))
	})
}

func TestPruneEIDs_GetUserExtError(t *testing.T) {
	allowedVendors := map[string]struct{}{
		"uid2": {},
	}

	// Create wrapper with invalid JSON in user.ext to trigger GetUserExt error
	wrapper := &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			User: &openrtb2.User{
				Ext: json.RawMessage(`{"invalid": json}`), // Invalid JSON
			},
		},
	}

	err := pruneEIDs(wrapper, allowedVendors)
	assert.Error(t, err)
}

func TestShouldKeepEID_TDIDNoMatch(t *testing.T) {
	allowedVendors := map[string]struct{}{
		"tdid": {},
	}

	// EID with adserver.org source but no matching rtiPartner
	eid := openrtb2.EID{
		Source: "adserver.org",
		UIDs: []openrtb2.UID{
			{
				ID:  "test-id",
				Ext: json.RawMessage(`{"rtiPartner":"OTHER"}`), // Not TDID
			},
		},
	}

	result := shouldKeepEID(eid, allowedVendors)
	assert.False(t, result)
}

func TestShouldKeepEID_MoreVendorPatterns(t *testing.T) {
	tests := []struct {
		name          string
		eidSource     string
		allowedVendor string
		expected      bool
	}{
		{
			name:          "33acrossId",
			eidSource:     "https://33across.com/api/v1/uid",
			allowedVendor: "33acrossId",
			expected:      true,
		},
		{
			name:          "hadronId",
			eidSource:     "https://audigent.com/hadron",
			allowedVendor: "hadronId",
			expected:      true,
		},
		{
			name:          "idl_env",
			eidSource:     "https://liveramp.com/identitylink",
			allowedVendor: "idl_env",
			expected:      true,
		},
		{
			name:          "index",
			eidSource:     "https://casalemedia.com/index",
			allowedVendor: "index",
			expected:      true,
		},
		{
			name:          "magnite",
			eidSource:     "https://rubiconproject.com/magnite",
			allowedVendor: "magnite",
			expected:      true,
		},
		{
			name:          "medianet",
			eidSource:     "https://media.net/uid",
			allowedVendor: "medianet",
			expected:      true,
		},
		{
			name:          "openx",
			eidSource:     "https://openx.net/uid",
			allowedVendor: "openx",
			expected:      true,
		},
		{
			name:          "pubmatic",
			eidSource:     "https://pubmatic.com/uid",
			allowedVendor: "pubmatic",
			expected:      true,
		},
		{
			name:          "sovrn",
			eidSource:     "https://liveintent.com/sovrn",
			allowedVendor: "sovrn",
			expected:      true,
		},
		{
			name:          "uid2",
			eidSource:     "https://uidapi.com/uid",
			allowedVendor: "uid2",
			expected:      true,
		},
		{
			name:          "direct_match",
			eidSource:     "customsource.com",
			allowedVendor: "customsource.com",
			expected:      true,
		},
		{
			name:          "ambiguous_fail_open",
			eidSource:     "unknown.source.com",
			allowedVendor: "othervendor",
			expected:      true, // Fail-open behavior
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowedVendors := map[string]struct{}{
				tt.allowedVendor: {},
			}
			eid := openrtb2.EID{Source: tt.eidSource}
			result := shouldKeepEID(eid, allowedVendors)
			assert.Equal(t, tt.expected, result)
		})
	}
}
