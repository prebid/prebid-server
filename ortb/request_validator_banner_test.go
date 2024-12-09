package ortb

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestValidateBanner(t *testing.T) {
	tests := []struct {
		name         string
		banner       *openrtb2.Banner
		interstitial bool
		wantError    bool
	}{
		{
			name:         "nil",
			banner:       nil,
			interstitial: false,
			wantError:    false,
		},
		{
			name:         "no_root_or_format_and_interstitial",
			banner:       &openrtb2.Banner{},
			interstitial: true,
			wantError:    false,
		},
		{
			name:         "no_root_or_format_and_not_interstitial",
			banner:       &openrtb2.Banner{},
			interstitial: false,
			wantError:    true,
		},
		{
			name: "well_formed_root",
			banner: &openrtb2.Banner{
				W: ptrutil.ToPtr[int64](11),
				H: ptrutil.ToPtr[int64](1),
			},
			interstitial: false,
			wantError:    false,
		},
		{
			name: "well_formed_format",
			banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					{
						W: 1,
						H: 1,
					},
				},
			},
			interstitial: false,
			wantError:    false,
		},
		{
			name: "invalid_format",
			banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					{
						W:      1,
						H:      1,
						HRatio: -1,
					},
				},
			},
			interstitial: false,
			wantError:    true,
		},
		{
			name: "negative_width",
			banner: &openrtb2.Banner{
				W: ptrutil.ToPtr[int64](-1),
				H: ptrutil.ToPtr[int64](1),
			},
			interstitial: false,
			wantError:    true,
		},
		{
			name: "negative_height",
			banner: &openrtb2.Banner{
				W: ptrutil.ToPtr[int64](1),
				H: ptrutil.ToPtr[int64](-1),
			},
			interstitial: false,
			wantError:    true,
		},
		{
			name: "nonzero_wmin",
			banner: &openrtb2.Banner{
				W:    ptrutil.ToPtr[int64](1),
				H:    ptrutil.ToPtr[int64](1),
				WMin: 1,
			},
			interstitial: false,
			wantError:    true,
		},
		{
			name: "nonzero_wmax",
			banner: &openrtb2.Banner{
				W:    ptrutil.ToPtr[int64](1),
				H:    ptrutil.ToPtr[int64](1),
				WMax: 1,
			},
			interstitial: false,
			wantError:    true,
		},
		{
			name: "nonzero_hmin",
			banner: &openrtb2.Banner{
				W:    ptrutil.ToPtr[int64](1),
				H:    ptrutil.ToPtr[int64](1),
				HMin: 1,
			},
			interstitial: false,
			wantError:    true,
		},
		{
			name: "nonzero_hmax",
			banner: &openrtb2.Banner{
				W:    ptrutil.ToPtr[int64](1),
				H:    ptrutil.ToPtr[int64](1),
				HMax: 1,
			},
			interstitial: false,
			wantError:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validateBanner(test.banner, 1, test.interstitial)
			if test.wantError {
				assert.Error(t, result)
			} else {
				assert.NoError(t, result)
			}
		})
	}
}

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name      string
		format    *openrtb2.Format
		wantError bool
	}{
		{
			name:      "nil",
			format:    nil,
			wantError: false,
		},
		{
			name: "well_formed",
			format: &openrtb2.Format{
				W:      1,
				H:      1,
				WRatio: 0,
				HRatio: 0,
				WMin:   0,
			},
			wantError: false,
		},
		{
			name: "well_formed_using_ratios",
			format: &openrtb2.Format{
				W:      0,
				H:      0,
				WRatio: 1,
				HRatio: 1,
				WMin:   1,
			},
			wantError: false,
		},
		{
			name: "negative_width",
			format: &openrtb2.Format{
				W:      -1,
				H:      1,
				WRatio: 0,
				HRatio: 0,
				WMin:   0,
			},
			wantError: true,
		},
		{
			name: "negative_height",
			format: &openrtb2.Format{
				W:      1,
				H:      -1,
				WRatio: 0,
				HRatio: 0,
				WMin:   0,
			},
			wantError: true,
		},
		{
			name: "negative_width_ratio",
			format: &openrtb2.Format{
				W:      0,
				H:      0,
				WRatio: -1,
				HRatio: 1,
				WMin:   1,
			},
			wantError: true,
		},
		{
			name: "negative_height_ratio",
			format: &openrtb2.Format{
				W:      0,
				H:      0,
				WRatio: 1,
				HRatio: -1,
				WMin:   1,
			},
			wantError: true,
		},
		{
			name: "negative_height_wmin",
			format: &openrtb2.Format{
				W:      1,
				H:      1,
				WRatio: 1,
				HRatio: 1,
				WMin:   -1,
			},
			wantError: true,
		},
		{
			name: "using_both_formats",
			format: &openrtb2.Format{
				W:      1,
				H:      1,
				WRatio: 1,
				HRatio: 1,
				WMin:   1,
			},
			wantError: true,
		},
		{
			name: "using_neither_format",
			format: &openrtb2.Format{
				W:      0,
				H:      0,
				WRatio: 0,
				HRatio: 0,
				WMin:   0,
			},
			wantError: true,
		},
		{
			name: "using_non_ratios_but_zeros",
			format: &openrtb2.Format{
				W:      1,
				H:      0,
				WRatio: 0,
				HRatio: 0,
				WMin:   0,
			},
			wantError: true,
		},
		{
			name: "using_ratios_but_zeros",
			format: &openrtb2.Format{
				W:      0,
				H:      0,
				WRatio: 1,
				HRatio: 0,
				WMin:   0,
			},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validateFormat(test.format, 1, 1)
			if test.wantError {
				assert.Error(t, result)
			} else {
				assert.NoError(t, result)
			}
		})
	}
}
