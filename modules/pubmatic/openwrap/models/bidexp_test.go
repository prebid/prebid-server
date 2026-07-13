package models

import (
	"testing"

	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestOmitTrackerBidExp(t *testing.T) {
	tests := []struct {
		name      string
		rctx      RequestCtx
		bidExpEnf int
		want      bool
	}{
		{
			name:      "omit_when_bidexp_enf_absent_and_AppSubIntegrationPath_is_AdMob_SDK_bidding",
			rctx:      RequestCtx{AppSubIntegrationPath: ptrutil.ToPtr(AppSubIntegrationPathIDAdMobSDKBidding)},
			bidExpEnf: 0,
			want:      true,
		},
		{
			name:      "omit_when_bidexp_enf_absent_and_AppSubIntegrationPath_is_Google_Ad_Manager_SDK_bidding",
			rctx:      RequestCtx{AppSubIntegrationPath: ptrutil.ToPtr(AppSubIntegrationPathIDGoogleAdManagerSDKBidding)},
			bidExpEnf: 0,
			want:      true,
		},
		{
			name:      "no_omit_when_bidexp_enf_is_1_and_AppSubIntegrationPath_is_AdMob_SDK_bidding",
			rctx:      RequestCtx{AppSubIntegrationPath: ptrutil.ToPtr(AppSubIntegrationPathIDAdMobSDKBidding)},
			bidExpEnf: 1,
			want:      false,
		},
		{
			name:      "no_omit_when_bidexp_enf_is_1_and_AppSubIntegrationPath_is_Google_Ad_Manager_SDK_bidding",
			rctx:      RequestCtx{AppSubIntegrationPath: ptrutil.ToPtr(AppSubIntegrationPathIDGoogleAdManagerSDKBidding)},
			bidExpEnf: 1,
			want:      false,
		},
		{
			name:      "omit_when_other_value_and_AppSubIntegrationPath_is_AdMob_SDK_bidding",
			rctx:      RequestCtx{AppSubIntegrationPath: ptrutil.ToPtr(AppSubIntegrationPathIDAdMobSDKBidding)},
			bidExpEnf: 2,
			want:      true,
		},
		{
			name:      "no_omit_when_AppSubIntegrationPath_is_not_AdMob_or_GAM_even_if_bidexp_enf_absent",
			rctx:      RequestCtx{AppSubIntegrationPath: ptrutil.ToPtr(99)},
			bidExpEnf: 0,
			want:      false,
		},
		{
			name:      "no_omit_when_AppSubIntegrationPath_is_nil",
			rctx:      RequestCtx{},
			bidExpEnf: 0,
			want:      false,
		},
		{
			name:      "no_omit_when_AppSubIntegrationPath_is_unset_sentinel_minus_1",
			rctx:      RequestCtx{AppSubIntegrationPath: ptrutil.ToPtr(-1)},
			bidExpEnf: 0,
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, OmitTrackerBidExp(tt.rctx, tt.bidExpEnf))
		})
	}
}
