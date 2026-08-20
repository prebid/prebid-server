package openwrap

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/adunitconfig"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestGetSupportedAdAttributeWireIDs(t *testing.T) {
	tests := []struct {
		name       string
		os         OS
		sdkVersion string
		adFormat   AdFormat
		expected   []int
	}{
		{
			name:       "Android_4.0.0_-_below_minimum",
			os:         OSAndroid,
			sdkVersion: "4.0.0",
			adFormat:   AdFormatInterstitialDisplay,
			expected:   nil,
		},
		{
			name:       "Android_4.1.0_-_interstitial_display",
			os:         OSAndroid,
			sdkVersion: "4.1.0",
			adFormat:   AdFormatInterstitialDisplay,
			expected:   []int{AdAttrWireEngageToClose},
		},
		{
			name:       "iOS_4.1.0_-_interstitial_display",
			os:         OSiOS,
			sdkVersion: "4.1.0",
			adFormat:   AdFormatInterstitialDisplay,
			expected:   []int{AdAttrWireEngageToClose},
		},
		{
			name:       "iOS_4.5.0_-_interstitial_display",
			os:         OSiOS,
			sdkVersion: "4.5.0",
			adFormat:   AdFormatInterstitialDisplay,
			expected:   []int{AdAttrWireEngageToClose},
		},
		{
			name:       "iOS_4.5.0_-_banner_display",
			os:         OSiOS,
			sdkVersion: "4.5.0",
			adFormat:   AdFormatBannerDisplay,
			expected:   []int{AdAttrWireEngageToClose},
		},
		{
			name:       "iOS_4.5.0_-_MREC_display",
			os:         OSiOS,
			sdkVersion: "4.5.0",
			adFormat:   AdFormatMRECDisplay,
			expected:   []int{AdAttrWireEngageToClose},
		},
		{
			name:       "Android_4.5.0_-_MREC_display",
			os:         OSAndroid,
			sdkVersion: "4.5.0",
			adFormat:   AdFormatMRECDisplay,
			expected:   []int{AdAttrWireEngageToClose},
		},
		{
			name:       "Android_4.5.0_-_MREC_video_no_row_below_4.9",
			os:         OSAndroid,
			sdkVersion: "4.5.0",
			adFormat:   AdFormatMRECVideo,
			expected:   nil,
		},
		{
			name:       "iOS_4.1.0_-_interstitial_video_no_row_in_4.1_4.2_band",
			os:         OSiOS,
			sdkVersion: "4.1.0",
			adFormat:   AdFormatInterstitialVideo,
			expected:   nil,
		},
		{
			name:       "iOS_4.3.0_-_interstitial_video",
			os:         OSiOS,
			sdkVersion: "4.3.0",
			adFormat:   AdFormatInterstitialVideo,
			expected:   []int{AdAttrWireEngageToClose},
		},
		{
			name:       "iOS_4.3.0_-_rewarded_video",
			os:         OSiOS,
			sdkVersion: "4.3.0",
			adFormat:   AdFormatRewardedVideo,
			expected:   []int{AdAttrWireEngageToClose},
		},
		{
			name:       "Android_4.3.0_-_interstitial_display",
			os:         OSAndroid,
			sdkVersion: "4.3.0",
			adFormat:   AdFormatInterstitialDisplay,
			expected:   []int{AdAttrWireEngageToClose},
		},
		{
			name:       "Android_4.3.0_-_interstitial_video",
			os:         OSAndroid,
			sdkVersion: "4.3.0",
			adFormat:   AdFormatInterstitialVideo,
			expected:   []int{AdAttrWireEngageToClose},
		},
		{
			name:       "Android_4.3.0_-_rewarded_video",
			os:         OSAndroid,
			sdkVersion: "4.3.0",
			adFormat:   AdFormatRewardedVideo,
			expected:   []int{AdAttrWireEngageToClose},
		},
		{
			name:       "Android_4.5.0_-_banner_display",
			os:         OSAndroid,
			sdkVersion: "4.5.0",
			adFormat:   AdFormatBannerDisplay,
			expected:   []int{AdAttrWireEngageToClose},
		},
		{
			name:       "Android_4.5.0_-_interstitial_video",
			os:         OSAndroid,
			sdkVersion: "4.5.0",
			adFormat:   AdFormatInterstitialVideo,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireTrueDoubleEndCard},
		},
		{
			name:       "Android_4.4.0_-_interstitial_display",
			os:         OSAndroid,
			sdkVersion: "4.4.0",
			adFormat:   AdFormatInterstitialDisplay,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireTrueDoubleEndCard},
		},
		{
			name:       "Android_4.4.0_-_interstitial_video",
			os:         OSAndroid,
			sdkVersion: "4.4.0",
			adFormat:   AdFormatInterstitialVideo,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireTrueDoubleEndCard},
		},
		{
			name:       "Android_4.4.0_-_rewarded_video",
			os:         OSAndroid,
			sdkVersion: "4.4.0",
			adFormat:   AdFormatRewardedVideo,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireTrueDoubleEndCard},
		},
		{
			name:       "Android_4.9.0_-_MREC_display",
			os:         OSAndroid,
			sdkVersion: "4.9.0",
			adFormat:   AdFormatMRECDisplay,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay},
		},
		{
			name:       "Android_4.9.0_-_MREC_video",
			os:         OSAndroid,
			sdkVersion: "4.9.0",
			adFormat:   AdFormatMRECVideo,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay},
		},
		{
			name:       "Android_4.9.0_-_interstitial_video",
			os:         OSAndroid,
			sdkVersion: "4.9.0",
			adFormat:   AdFormatInterstitialVideo,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireTrueDoubleEndCard, AdAttrWireCTAOverlay},
		},
		{
			name:       "Android_4.9.0_-_interstitial_display",
			os:         OSAndroid,
			sdkVersion: "4.9.0",
			adFormat:   AdFormatInterstitialDisplay,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay},
		},
		{
			name:       "Android_4.9.0_-_rewarded_video",
			os:         OSAndroid,
			sdkVersion: "4.9.0",
			adFormat:   AdFormatRewardedVideo,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireTrueDoubleEndCard, AdAttrWireCTAOverlay},
		},
		{
			name:       "iOS_4.9.0_-_interstitial_display",
			os:         OSiOS,
			sdkVersion: "4.9.0",
			adFormat:   AdFormatInterstitialDisplay,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay},
		},
		{
			name:       "iOS_4.9.0_-_interstitial_video",
			os:         OSiOS,
			sdkVersion: "4.9.0",
			adFormat:   AdFormatInterstitialVideo,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay},
		},
		{
			name:       "iOS_4.9.0_-_rewarded_video",
			os:         OSiOS,
			sdkVersion: "4.9.0",
			adFormat:   AdFormatRewardedVideo,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay},
		},
		{
			name:       "iOS_4.9.0_-_MREC_display",
			os:         OSiOS,
			sdkVersion: "4.9.0",
			adFormat:   AdFormatMRECDisplay,
			expected:   []int{AdAttrWireCTAOverlay},
		},
		{
			name:       "iOS_4.9.0_-_banner_display_no_row_in_4.9_band",
			os:         OSiOS,
			sdkVersion: "4.9.0",
			adFormat:   AdFormatBannerDisplay,
			expected:   nil,
		},
		{
			name:       "Android_5.1.0_-_interstitial_display",
			os:         OSAndroid,
			sdkVersion: "5.1.0",
			adFormat:   AdFormatInterstitialDisplay,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus},
		},
		{
			name:       "Android_5.3.0_-_interstitial_display",
			os:         OSAndroid,
			sdkVersion: "5.3.0",
			adFormat:   AdFormatInterstitialDisplay,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus},
		},
		{
			name:       "Android_5.1.0_-_interstitial_video",
			os:         OSAndroid,
			sdkVersion: "5.1.0",
			adFormat:   AdFormatInterstitialVideo,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireTrueDoubleEndCard, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus},
		},
		{
			name:       "Android_5.1.0_-_rewarded_video",
			os:         OSAndroid,
			sdkVersion: "5.1.0",
			adFormat:   AdFormatRewardedVideo,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireTrueDoubleEndCard, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus},
		},
		{
			name:       "Android_5.1.0_-_MREC_display",
			os:         OSAndroid,
			sdkVersion: "5.1.0",
			adFormat:   AdFormatMRECDisplay,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus},
		},
		{
			name:       "Android_5.1.0_-_banner_display",
			os:         OSAndroid,
			sdkVersion: "5.1.0",
			adFormat:   AdFormatBannerDisplay,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay},
		},
		{
			name:       "iOS_5.1.0_-_interstitial_display",
			os:         OSiOS,
			sdkVersion: "5.1.0",
			adFormat:   AdFormatInterstitialDisplay,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus},
		},
		{
			name:       "iOS_5.1.0_-_interstitial_video",
			os:         OSiOS,
			sdkVersion: "5.1.0",
			adFormat:   AdFormatInterstitialVideo,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus},
		},
		{
			name:       "iOS_5.1.0_-_MREC_display",
			os:         OSiOS,
			sdkVersion: "5.1.0",
			adFormat:   AdFormatMRECDisplay,
			expected:   []int{AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus},
		},
		{
			name:       "iOS_5.1.0_-_MREC_video",
			os:         OSiOS,
			sdkVersion: "5.1.0",
			adFormat:   AdFormatMRECVideo,
			expected:   []int{AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus},
		},
		{
			name:       "iOS_5.1.0_-_banner_display",
			os:         OSiOS,
			sdkVersion: "5.1.0",
			adFormat:   AdFormatBannerDisplay,
			expected:   []int{AdAttrWireMRAIDAppStatus},
		},
		{
			name:       "Android_5.2.0_-_interstitial_video",
			os:         OSAndroid,
			sdkVersion: "5.2.0",
			adFormat:   AdFormatInterstitialVideo,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireTrueDoubleEndCard, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus},
		},
		{
			name:       "Android_5.2.0_-_rewarded_video",
			os:         OSAndroid,
			sdkVersion: "5.2.0",
			adFormat:   AdFormatRewardedVideo,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireTrueDoubleEndCard, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus},
		},
		{
			name:       "Android_5.2.0_-_MREC_display",
			os:         OSAndroid,
			sdkVersion: "5.2.0",
			adFormat:   AdFormatMRECDisplay,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus},
		},
		{
			name:       "Android_5.2.0_-_MREC_video",
			os:         OSAndroid,
			sdkVersion: "5.2.0",
			adFormat:   AdFormatMRECVideo,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus},
		},
		{
			name:       "Android_5.2.0_-_banner_display",
			os:         OSAndroid,
			sdkVersion: "5.2.0",
			adFormat:   AdFormatBannerDisplay,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay},
		},
		{
			name:       "iOS_5.2.0_-_interstitial_display",
			os:         OSiOS,
			sdkVersion: "5.2.0",
			adFormat:   AdFormatInterstitialDisplay,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus},
		},
		{
			name:       "iOS_5.2.0_-_interstitial_video",
			os:         OSiOS,
			sdkVersion: "5.2.0",
			adFormat:   AdFormatInterstitialVideo,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus},
		},
		{
			name:       "iOS_5.2.0_-_rewarded_video",
			os:         OSiOS,
			sdkVersion: "5.2.0",
			adFormat:   AdFormatRewardedVideo,
			expected:   []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus},
		},
		{
			name:       "iOS_5.2.0_-_MREC_display",
			os:         OSiOS,
			sdkVersion: "5.2.0",
			adFormat:   AdFormatMRECDisplay,
			expected:   []int{AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus},
		},
		{
			name:       "iOS_5.2.0_-_MREC_video",
			os:         OSiOS,
			sdkVersion: "5.2.0",
			adFormat:   AdFormatMRECVideo,
			expected:   []int{AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus},
		},
		{
			name:       "iOS_5.2.0_-_banner_display",
			os:         OSiOS,
			sdkVersion: "5.2.0",
			adFormat:   AdFormatBannerDisplay,
			expected:   []int{AdAttrWireMRAIDAppStatus},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetSupportedAdAttributeWireIDs(tt.os, tt.sdkVersion, tt.adFormat)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name string
		v1   string
		v2   string
		want int
	}{
		{name: "less_than", v1: "5.1.0", v2: "5.2.0", want: -1},
		{name: "equal", v1: "5.2.0", v2: "5.2.0", want: 0},
		{name: "greater_than", v1: "5.3.0", v2: "5.2.0", want: 1},
		{name: "short_form_equal", v1: "5.1", v2: "5.1.0", want: 0},
		{name: "trims_whitespace", v1: " 5.1.0 ", v2: "5.1.0", want: 0},
		{name: "above_minimum_gate", v1: "4.1.0", v2: "4.0.9", want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, compareVersions(tt.v1, tt.v2))
		})
	}
}

func TestDetermineAdFormatForVideo(t *testing.T) {
	tests := []struct {
		name     string
		impCtx   models.ImpCtx
		expected AdFormat
	}{
		{
			name: "rewarded_video",
			impCtx: models.ImpCtx{
				IsRewardInventory: ptrutil.ToPtr(int8(1)),
				Instl:             1,
				Video:             &openrtb2.Video{},
			},
			expected: AdFormatRewardedVideo,
		},
		{
			name: "rewarded_flag_without_video",
			impCtx: models.ImpCtx{
				IsRewardInventory: ptrutil.ToPtr(int8(1)),
				Instl:             1,
			},
			expected: "",
		},
		{
			name: "interstitial_video",
			impCtx: models.ImpCtx{
				Instl: 1,
				Video: &openrtb2.Video{},
			},
			expected: AdFormatInterstitialVideo,
		},
		{
			name: "interstitial_video_with_banner_present",
			impCtx: models.ImpCtx{
				Instl:  1,
				Video:  &openrtb2.Video{},
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr(int64(320)), H: ptrutil.ToPtr(int64(50))},
			},
			expected: AdFormatInterstitialVideo,
		},
		{
			name: "MREC_video",
			impCtx: models.ImpCtx{
				Instl: 0,
				Video: &openrtb2.Video{W: ptrutil.ToPtr(int64(300)), H: ptrutil.ToPtr(int64(250))},
			},
			expected: AdFormatMRECVideo,
		},
		{
			name: "MREC_video_with_MREC_banner_but_non_MREC_video_size",
			impCtx: models.ImpCtx{
				Instl:  0,
				Video:  &openrtb2.Video{W: ptrutil.ToPtr(int64(320)), H: ptrutil.ToPtr(int64(50))},
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr(int64(300)), H: ptrutil.ToPtr(int64(250))},
			},
			expected: "",
		},
		{
			name: "non_interstitial_non_MREC_video",
			impCtx: models.ImpCtx{
				Instl: 0,
				Video: &openrtb2.Video{W: ptrutil.ToPtr(int64(320)), H: ptrutil.ToPtr(int64(50))},
			},
			expected: "",
		},
		{
			name: "MREC_video_with_banner_disabled",
			impCtx: models.ImpCtx{
				Instl:  0,
				Video:  &openrtb2.Video{W: ptrutil.ToPtr(int64(300)), H: ptrutil.ToPtr(int64(250))},
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr(int64(300)), H: ptrutil.ToPtr(int64(250))},
				BannerAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Banner: &adunitconfig.Banner{Enabled: ptrutil.ToPtr(false)},
					},
				},
			},
			expected: AdFormatMRECVideo,
		},
		{
			name: "MREC_video_with_video_disabled",
			impCtx: models.ImpCtx{
				Instl: 0,
				Video: &openrtb2.Video{W: ptrutil.ToPtr(int64(300)), H: ptrutil.ToPtr(int64(250))},
				VideoAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Video: &adunitconfig.Video{Enabled: ptrutil.ToPtr(false)},
					},
				},
			},
			expected: "",
		},
		{
			name: "video_disabled_by_ad_unit_config",
			impCtx: models.ImpCtx{
				Instl: 1,
				Video: &openrtb2.Video{},
				VideoAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Video: &adunitconfig.Video{Enabled: ptrutil.ToPtr(false)},
					},
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineAdFormatForVideo(tt.impCtx)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDetermineAdFormatForBanner(t *testing.T) {
	tests := []struct {
		name     string
		impCtx   models.ImpCtx
		expected AdFormat
	}{
		{
			name: "interstitial_display",
			impCtx: models.ImpCtx{
				Instl:  1,
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr(int64(320)), H: ptrutil.ToPtr(int64(50))},
			},
			expected: AdFormatInterstitialDisplay,
		},
		{
			name: "interstitial_display_with_video_present",
			impCtx: models.ImpCtx{
				Instl:  1,
				Video:  &openrtb2.Video{},
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr(int64(320)), H: ptrutil.ToPtr(int64(50))},
			},
			expected: AdFormatInterstitialDisplay,
		},
		{
			name: "MREC_display",
			impCtx: models.ImpCtx{
				Instl:  0,
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr(int64(300)), H: ptrutil.ToPtr(int64(250))},
			},
			expected: AdFormatMRECDisplay,
		},
		{
			name: "banner_display",
			impCtx: models.ImpCtx{
				Instl:  0,
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr(int64(320)), H: ptrutil.ToPtr(int64(50))},
			},
			expected: AdFormatBannerDisplay,
		},
		{
			name: "no_banner_object",
			impCtx: models.ImpCtx{
				Instl: 1,
				Video: &openrtb2.Video{},
			},
			expected: "",
		},
		{
			name: "banner_disabled_by_ad_unit_config",
			impCtx: models.ImpCtx{
				Instl:  1,
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr(int64(320)), H: ptrutil.ToPtr(int64(50))},
				BannerAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Banner: &adunitconfig.Banner{Enabled: ptrutil.ToPtr(false)},
					},
				},
			},
			expected: "",
		},
		{
			name: "MREC_display_when_video_disabled",
			impCtx: models.ImpCtx{
				Instl:  0,
				Video:  &openrtb2.Video{},
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr(int64(300)), H: ptrutil.ToPtr(int64(250))},
				VideoAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Video: &adunitconfig.Video{Enabled: ptrutil.ToPtr(false)},
					},
				},
			},
			expected: AdFormatMRECDisplay,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineAdFormatForBanner(tt.impCtx)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDetermineOS(t *testing.T) {
	tests := []struct {
		name     string
		deviceOS string
		expected OS
	}{
		{
			name:     "Android",
			deviceOS: "Android",
			expected: OSAndroid,
		},
		{
			name:     "android_lowercase",
			deviceOS: "android",
			expected: OSAndroid,
		},
		{
			name:     "iOS",
			deviceOS: "iOS",
			expected: OSiOS,
		},
		{
			name:     "iPhone",
			deviceOS: "iPhone",
			expected: OSiOS,
		},
		{
			name:     "iPad",
			deviceOS: "iPad",
			expected: OSiOS,
		},
		{
			name:     "unknown_OS",
			deviceOS: "Windows",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineOS(tt.deviceOS)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCreateOWSDKExtension(t *testing.T) {
	tests := []struct {
		name     string
		wireIDs  []int
		expected map[string]any
	}{
		{
			name:     "no_ids",
			wireIDs:  []int{},
			expected: nil,
		},
		{
			name:     "all_non_positive_ids",
			wireIDs:  []int{0, -1},
			expected: nil,
		},
		{
			name:    "single_id",
			wireIDs: []int{AdAttrWireCTAOverlay},
			expected: map[string]any{
				"adattributes": []int{AdAttrWireCTAOverlay},
			},
		},
		{
			name:    "dedupe_and_sort",
			wireIDs: []int{3, 1, 1, 3},
			expected: map[string]any{
				"adattributes": []int{1, 3},
			},
		},
		{
			name:    "skips_non_positive_ids",
			wireIDs: []int{0, -1, 4, 3},
			expected: map[string]any{
				"adattributes": []int{3, 4},
			},
		},
		{
			name:    "multiple_ids_sorted",
			wireIDs: []int{4, 1, 3},
			expected: map[string]any{
				"adattributes": []int{1, 3, 4},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateOWSDKExtension(tt.wireIDs)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestMergeOWSDKServerFieldsIntoExtJSON(t *testing.T) {
	marshalServerOWSDK := func(serverOWSDK map[string]any) ([]byte, []byte) {
		if len(serverOWSDK) == 0 {
			return nil, nil
		}
		serverOWSDKJSON, err := json.Marshal(serverOWSDK)
		assert.NoError(t, err)
		adAttributesJSON, err := json.Marshal(serverOWSDK[adAttributesKey])
		assert.NoError(t, err)
		return serverOWSDKJSON, adAttributesJSON
	}

	tests := []struct {
		name        string
		extJSON     json.RawMessage
		serverOWSDK map[string]any
		wantJSON    string
	}{
		{
			name:        "empty_server_owsdk_returns_ext_unchanged",
			extJSON:     json.RawMessage(`{"foo":"bar"}`),
			serverOWSDK: map[string]any{},
			wantJSON:    `{"foo":"bar"}`,
		},
		{
			name:        "valid_ext_JSON_unmarshals_and_preserves_sibling_keys",
			extJSON:     json.RawMessage(`{"prebid":{"test":1}}`),
			serverOWSDK: map[string]any{"adattributes": []int{1}},
			wantJSON:    `{"prebid":{"test":1},"owsdk":{"adattributes":[1]}}`,
		},
		{
			name:        "invalid_ext_JSON_resets_ext_map_and_still_merges_owsdk",
			extJSON:     json.RawMessage(`{not-valid-json`),
			serverOWSDK: map[string]any{"adattributes": []int{4}},
			wantJSON:    `{"owsdk":{"adattributes":[4]}}`,
		},
		{
			name:        "existing_owsdk_in_ext_is_merged_with_server_fields",
			extJSON:     json.RawMessage(`{"owsdk":{"ctaoverlay":1}}`),
			serverOWSDK: map[string]any{"adattributes": []int{1, 3}},
			wantJSON:    `{"owsdk":{"adattributes":[1,3],"ctaoverlay":1}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverOWSDKJSON, adAttributesJSON := marshalServerOWSDK(tt.serverOWSDK)
			got, err := mergeOWSDKServerFieldsIntoExtJSON(tt.extJSON, serverOWSDKJSON, adAttributesJSON)
			assert.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(got))
		})
	}
}

func TestShouldServerInjectFormatLevelAdAttributes(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"", false},
		{"4.0.9", false},
		{"4.1.0", true},
		{"5.0.0", true},
		{"5.1.0", true},
		{"5.1.1", true},
		{"5.2.0", true},
		{"5.2.1", true},
		{"5.3.0", true},
		{"5.3.1", false},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldServerInjectFormatLevelAdAttributes(tt.version))
		})
	}
}

var (
	adAttrInterstitialDisplay = []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus}
	adAttrInterstitialVideo   = []int{AdAttrWireEngageToClose, AdAttrWireTrueDoubleEndCard, AdAttrWireCTAOverlay, AdAttrWireMRAIDAppStatus}
	adAttrBannerLeaderboard   = []int{AdAttrWireEngageToClose, AdAttrWireCTAOverlay}
	adAttrMRECDisplayIOS49    = []int{AdAttrWireCTAOverlay}
)

type formatExtExpect struct {
	inject     bool
	adAttrs    []int
	keepPrebid bool
}

func assertFormatExtOWSDK(t *testing.T, label string, ext json.RawMessage, expect formatExtExpect) {
	t.Helper()
	if !expect.inject {
		assert.Empty(t, ext, label)
		return
	}

	var got struct {
		Prebid json.RawMessage `json:"prebid"`
		OWSDK  struct {
			Adattributes []int `json:"adattributes"`
		} `json:"owsdk"`
	}
	assert.NoError(t, json.Unmarshal(ext, &got))
	assert.Equal(t, expect.adAttrs, got.OWSDK.Adattributes, "%s adattributes", label)
	if expect.keepPrebid {
		assert.NotEmpty(t, got.Prebid, "%s should preserve prebid", label)
	}
}

func TestApplyOWSDKFormatLevelAdAttributes(t *testing.T) {
	tests := []struct {
		name     string
		deviceOS string
		imp      *openrtb2.Imp
		impCtx   models.ImpCtx
		banner   formatExtExpect
		video    formatExtExpect
	}{
		{
			name:     "Android_5.2.0_-_interstitial_video_plus_display_banner_and_video_ext",
			deviceOS: "Android",
			imp: &openrtb2.Imp{
				ID:     "test_imp_1",
				Instl:  1,
				Video:  &openrtb2.Video{MinDuration: 10, MaxDuration: 10},
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr(int64(300)), H: ptrutil.ToPtr(int64(250))},
			},
			impCtx: models.ImpCtx{
				DisplayManagerVer: "5.2.0",
				Instl:             1,
				Video:             &openrtb2.Video{MinDuration: 10, MaxDuration: 10},
				Banner:            &openrtb2.Banner{W: ptrutil.ToPtr(int64(300)), H: ptrutil.ToPtr(int64(250))},
			},
			banner: formatExtExpect{inject: true, adAttrs: adAttrInterstitialDisplay},
			video:  formatExtExpect{inject: true, adAttrs: adAttrInterstitialVideo},
		},
		{
			name:     "Android_5.1.0_-_interstitial_display_plus_video",
			deviceOS: "Android",
			imp: &openrtb2.Imp{
				ID:     "test_imp_1b",
				Instl:  1,
				Video:  &openrtb2.Video{MinDuration: 10, MaxDuration: 10},
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr(int64(300)), H: ptrutil.ToPtr(int64(250))},
			},
			impCtx: models.ImpCtx{
				DisplayManagerVer: "5.1.0",
				Instl:             1,
				Video:             &openrtb2.Video{MinDuration: 10, MaxDuration: 10},
				Banner:            &openrtb2.Banner{W: ptrutil.ToPtr(int64(300)), H: ptrutil.ToPtr(int64(250))},
			},
			banner: formatExtExpect{inject: true, adAttrs: adAttrInterstitialDisplay},
			video:  formatExtExpect{inject: true, adAttrs: adAttrInterstitialVideo},
		},
		{
			name:     "iOS_4.9.0_-_MREC_display_plus_video",
			deviceOS: "iOS",
			imp: &openrtb2.Imp{
				ID:     "test_imp_2",
				Instl:  0,
				Video:  &openrtb2.Video{W: ptrutil.ToPtr(int64(300)), H: ptrutil.ToPtr(int64(250)), MinDuration: 10, MaxDuration: 10},
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr(int64(300)), H: ptrutil.ToPtr(int64(250))},
			},
			impCtx: models.ImpCtx{
				DisplayManagerVer: "4.9.0",
				Instl:             0,
				Video:             &openrtb2.Video{W: ptrutil.ToPtr(int64(300)), H: ptrutil.ToPtr(int64(250)), MinDuration: 10, MaxDuration: 10},
				Banner:            &openrtb2.Banner{W: ptrutil.ToPtr(int64(300)), H: ptrutil.ToPtr(int64(250))},
			},
			banner: formatExtExpect{inject: true, adAttrs: adAttrMRECDisplayIOS49},
			video:  formatExtExpect{inject: true, adAttrs: adAttrMRECDisplayIOS49},
		},
		{
			name:     "Android_4.0.0_-_below_minimum_version",
			deviceOS: "Android",
			imp: &openrtb2.Imp{
				ID:    "test_imp_3",
				Instl: 1,
				Video: &openrtb2.Video{},
			},
			impCtx: models.ImpCtx{
				DisplayManagerVer: "4.0.0",
				Instl:             1,
				Video:             &openrtb2.Video{},
			},
		},
		{
			name:     "Android_5.3.1_-_SDK_sends_adattributes_server_skips_injection",
			deviceOS: "Android",
			imp: &openrtb2.Imp{
				ID:     "test_imp_5",
				Instl:  1,
				Video:  &openrtb2.Video{MinDuration: 10, MaxDuration: 10},
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr(int64(300)), H: ptrutil.ToPtr(int64(250))},
			},
			impCtx: models.ImpCtx{
				DisplayManagerVer: "5.3.1",
				Instl:             1,
				Video:             &openrtb2.Video{MinDuration: 10, MaxDuration: 10},
				Banner:            &openrtb2.Banner{W: ptrutil.ToPtr(int64(300)), H: ptrutil.ToPtr(int64(250))},
			},
		},
		{
			name:     "Unknown_OS",
			deviceOS: "Windows",
			imp: &openrtb2.Imp{
				ID:    "test_imp_4",
				Instl: 1,
				Video: &openrtb2.Video{},
			},
			impCtx: models.ImpCtx{
				DisplayManagerVer: "5.1.0",
				Instl:             1,
				Video:             &openrtb2.Video{},
			},
		},
		{
			name:     "empty_deviceOS_skips_injection",
			deviceOS: "",
			imp: &openrtb2.Imp{
				ID:     "test_imp_empty_os",
				Instl:  1,
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr(int64(320)), H: ptrutil.ToPtr(int64(50))},
			},
			impCtx: models.ImpCtx{
				DisplayManagerVer: "5.2.0",
				Instl:             1,
				Banner:            &openrtb2.Banner{W: ptrutil.ToPtr(int64(320)), H: ptrutil.ToPtr(int64(50))},
			},
		},
		{
			name:     "Android_5.2.0_-_rewarded_video_only",
			deviceOS: "Android",
			imp: &openrtb2.Imp{
				ID:    "test_imp_rewarded",
				Video: &openrtb2.Video{MinDuration: 5, MaxDuration: 30},
			},
			impCtx: models.ImpCtx{
				DisplayManagerVer: "5.2.0",
				IsRewardInventory: ptrutil.ToPtr(int8(1)),
				Video:             &openrtb2.Video{MinDuration: 5, MaxDuration: 30},
			},
			video: formatExtExpect{inject: true, adAttrs: adAttrInterstitialVideo},
		},
		{
			name:     "Android_5.2.0_-_interstitial_display_banner_only",
			deviceOS: "Android",
			imp: &openrtb2.Imp{
				ID:     "test_imp_isd_banner",
				Instl:  1,
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr(int64(320)), H: ptrutil.ToPtr(int64(50))},
			},
			impCtx: models.ImpCtx{
				DisplayManagerVer: "5.2.0",
				Instl:             1,
				Banner:            &openrtb2.Banner{W: ptrutil.ToPtr(int64(320)), H: ptrutil.ToPtr(int64(50))},
			},
			banner: formatExtExpect{inject: true, adAttrs: adAttrInterstitialDisplay},
		},
		{
			name:     "Android_5.2.0_-_MREC_display_banner_only",
			deviceOS: "Android",
			imp: &openrtb2.Imp{
				ID:     "test_imp_mrec_banner",
				Instl:  0,
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr(int64(300)), H: ptrutil.ToPtr(int64(250))},
			},
			impCtx: models.ImpCtx{
				DisplayManagerVer: "5.2.0",
				Instl:             0,
				Banner:            &openrtb2.Banner{W: ptrutil.ToPtr(int64(300)), H: ptrutil.ToPtr(int64(250))},
			},
			banner: formatExtExpect{inject: true, adAttrs: adAttrInterstitialDisplay},
		},
		{
			name:     "Android_5.2.0_-_banner_display_leaderboard_only",
			deviceOS: "Android",
			imp: &openrtb2.Imp{
				ID:     "test_imp_lb",
				Instl:  0,
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr(int64(320)), H: ptrutil.ToPtr(int64(50))},
			},
			impCtx: models.ImpCtx{
				DisplayManagerVer: "5.2.0",
				Instl:             0,
				Banner:            &openrtb2.Banner{W: ptrutil.ToPtr(int64(320)), H: ptrutil.ToPtr(int64(50))},
			},
			banner: formatExtExpect{inject: true, adAttrs: adAttrBannerLeaderboard},
		},
		{
			name:     "Android_5.2.0_-_video_disabled_banner_injection_only_on_interstitial",
			deviceOS: "Android",
			imp: &openrtb2.Imp{
				ID:     "test_imp_vid_disabled",
				Instl:  1,
				Video:  &openrtb2.Video{MinDuration: 10, MaxDuration: 10},
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr(int64(320)), H: ptrutil.ToPtr(int64(50))},
			},
			impCtx: models.ImpCtx{
				DisplayManagerVer: "5.2.0",
				Instl:             1,
				Video:             &openrtb2.Video{MinDuration: 10, MaxDuration: 10},
				Banner:            &openrtb2.Banner{W: ptrutil.ToPtr(int64(320)), H: ptrutil.ToPtr(int64(50))},
				VideoAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Video: &adunitconfig.Video{Enabled: ptrutil.ToPtr(false)},
					},
				},
			},
			banner: formatExtExpect{inject: true, adAttrs: adAttrInterstitialDisplay},
		},
		{
			name:     "Android_5.2.0_-_banner_disabled_video_injection_only_on_interstitial",
			deviceOS: "Android",
			imp: &openrtb2.Imp{
				ID:     "test_imp_banner_disabled",
				Instl:  1,
				Video:  &openrtb2.Video{MinDuration: 10, MaxDuration: 10},
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr(int64(320)), H: ptrutil.ToPtr(int64(50))},
			},
			impCtx: models.ImpCtx{
				DisplayManagerVer: "5.2.0",
				Instl:             1,
				Video:             &openrtb2.Video{MinDuration: 10, MaxDuration: 10},
				Banner:            &openrtb2.Banner{W: ptrutil.ToPtr(int64(320)), H: ptrutil.ToPtr(int64(50))},
				BannerAdUnitCtx: models.AdUnitCtx{
					AppliedSlotAdUnitConfig: &adunitconfig.AdConfig{
						Banner: &adunitconfig.Banner{Enabled: ptrutil.ToPtr(false)},
					},
				},
			},
			video: formatExtExpect{inject: true, adAttrs: adAttrInterstitialVideo},
		},
		{
			name:     "Android_5.2.0_-_merges_into_existing_banner.ext_and_video.ext",
			deviceOS: "Android",
			imp: &openrtb2.Imp{
				ID:    "test_imp_merge",
				Instl: 1,
				Video: &openrtb2.Video{
					MinDuration: 10,
					MaxDuration: 10,
					Ext:         json.RawMessage(`{"prebid":{"video":1}}`),
				},
				Banner: &openrtb2.Banner{
					W:   ptrutil.ToPtr(int64(320)),
					H:   ptrutil.ToPtr(int64(50)),
					Ext: json.RawMessage(`{"prebid":{"banner":1}}`),
				},
			},
			impCtx: models.ImpCtx{
				DisplayManagerVer: "5.2.0",
				Instl:             1,
				Video:             &openrtb2.Video{MinDuration: 10, MaxDuration: 10},
				Banner:            &openrtb2.Banner{W: ptrutil.ToPtr(int64(320)), H: ptrutil.ToPtr(int64(50))},
			},
			banner: formatExtExpect{inject: true, adAttrs: adAttrInterstitialDisplay, keepPrebid: true},
			video:  formatExtExpect{inject: true, adAttrs: adAttrInterstitialVideo, keepPrebid: true},
		},
		{
			name:     "nil_imp_no_op",
			deviceOS: "Android",
			imp:      nil,
			impCtx: models.ImpCtx{
				DisplayManagerVer: "5.2.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NoError(t, ApplyOWSDKFormatLevelAdAttributes(tt.imp, tt.impCtx, tt.deviceOS))
			if tt.imp == nil {
				return
			}
			if tt.imp.Banner != nil {
				assertFormatExtOWSDK(t, "banner", tt.imp.Banner.Ext, tt.banner)
			}
			if tt.imp.Video != nil {
				assertFormatExtOWSDK(t, "video", tt.imp.Video.Ext, tt.video)
			}
		})
	}
}
