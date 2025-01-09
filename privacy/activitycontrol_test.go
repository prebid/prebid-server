package privacy

import (
	"testing"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestNewActivityControl(t *testing.T) {
	testCases := []struct {
		name            string
		privacyConf     config.AccountPrivacy
		activityControl ActivityControl
	}{
		{
			name:            "empty",
			privacyConf:     config.AccountPrivacy{},
			activityControl: ActivityControl{plans: nil},
		},
		{
			name: "specified_and_correct",
			privacyConf: config.AccountPrivacy{
				AllowActivities: &config.AllowActivities{
					SyncUser:                 getTestActivityConfig(true),
					FetchBids:                getTestActivityConfig(true),
					EnrichUserFPD:            getTestActivityConfig(true),
					ReportAnalytics:          getTestActivityConfig(true),
					TransmitUserFPD:          getTestActivityConfig(true),
					TransmitPreciseGeo:       getTestActivityConfig(false),
					TransmitUniqueRequestIds: getTestActivityConfig(true),
					TransmitTids:             getTestActivityConfig(true),
				},
				IPv6Config: config.IPv6{AnonKeepBits: 32},
				IPv4Config: config.IPv4{AnonKeepBits: 16},
			},
			activityControl: ActivityControl{
				plans: map[Activity]ActivityPlan{
					ActivitySyncUser:                 getTestActivityPlan(ActivityAllow),
					ActivityFetchBids:                getTestActivityPlan(ActivityAllow),
					ActivityEnrichUserFPD:            getTestActivityPlan(ActivityAllow),
					ActivityReportAnalytics:          getTestActivityPlan(ActivityAllow),
					ActivityTransmitUserFPD:          getTestActivityPlan(ActivityAllow),
					ActivityTransmitPreciseGeo:       getTestActivityPlan(ActivityDeny),
					ActivityTransmitUniqueRequestIDs: getTestActivityPlan(ActivityAllow),
					ActivityTransmitTIDs:             getTestActivityPlan(ActivityAllow),
				},
				IPv6Config: config.IPv6{AnonKeepBits: 32},
				IPv4Config: config.IPv4{AnonKeepBits: 16},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualAC := NewActivityControl(&test.privacyConf)
			assert.Equal(t, test.activityControl, actualAC)
		})
	}
}

func TestCfgToDefaultResult(t *testing.T) {
	testCases := []struct {
		name            string
		activityDefault *bool
		expectedResult  bool
	}{
		{
			name:            "nil",
			activityDefault: nil,
			expectedResult:  true,
		},
		{
			name:            "true",
			activityDefault: ptrutil.ToPtr(true),
			expectedResult:  true,
		},
		{
			name:            "false",
			activityDefault: ptrutil.ToPtr(false),
			expectedResult:  false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualResult := cfgToDefaultResult(test.activityDefault)
			assert.Equal(t, test.expectedResult, actualResult)
		})
	}
}

func TestActivityControlAllow(t *testing.T) {
	testCases := []struct {
		name            string
		activityControl ActivityControl
		activity        Activity
		target          Component
		activityResult  bool
	}{
		{
			name:            "plans_is_nil",
			activityControl: ActivityControl{plans: nil},
			activity:        ActivityFetchBids,
			target:          Component{Type: "bidder", Name: "bidderA"},
			activityResult:  true,
		},
		{
			name: "activity_not_defined",
			activityControl: ActivityControl{plans: map[Activity]ActivityPlan{
				ActivitySyncUser: getTestActivityPlan(ActivityAllow)}},
			activity:       ActivityFetchBids,
			target:         Component{Type: "bidder", Name: "bidderA"},
			activityResult: true,
		},
		{
			name: "activity_defined_but_not_found_default_returned",
			activityControl: ActivityControl{plans: map[Activity]ActivityPlan{
				ActivityFetchBids: getTestActivityPlan(ActivityAllow)}},
			activity:       ActivityFetchBids,
			target:         Component{Type: "bidder", Name: "bidderB"},
			activityResult: true,
		},
		{
			name: "activity_defined_and_allowed",
			activityControl: ActivityControl{plans: map[Activity]ActivityPlan{
				ActivityFetchBids: getTestActivityPlan(ActivityAllow)}},
			activity:       ActivityFetchBids,
			target:         Component{Type: "bidder", Name: "bidderA"},
			activityResult: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualResult := test.activityControl.Allow(test.activity, test.target, ActivityRequest{})
			assert.Equal(t, test.activityResult, actualResult)

		})
	}
}

func TestActivityRequest(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		r := ActivityRequest{}
		assert.False(t, r.IsPolicies())
		assert.False(t, r.IsBidRequest())
	})

	t.Run("policies", func(t *testing.T) {
		r := NewRequestFromPolicies(Policies{})
		assert.True(t, r.IsPolicies())
		assert.False(t, r.IsBidRequest())
	})

	t.Run("request", func(t *testing.T) {
		r := NewRequestFromBidRequest(openrtb_ext.RequestWrapper{})
		assert.False(t, r.IsPolicies())
		assert.True(t, r.IsBidRequest())
	})
}

func getTestActivityConfig(allow bool) config.Activity {
	return config.Activity{
		Default: ptrutil.ToPtr(true),
		Rules: []config.ActivityRule{
			{
				Allow: allow,
				Condition: config.ActivityCondition{
					ComponentName: []string{"bidderA"},
					ComponentType: []string{"bidder"},
				},
			},
		},
	}
}

func getTestActivityPlan(result ActivityResult) ActivityPlan {
	return ActivityPlan{
		defaultResult: true,
		rules: []Rule{
			ConditionRule{
				result:        result,
				componentName: []string{"bidderA"},
				componentType: []string{"bidder"},
			},
		},
	}
}
