package privacy

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/util/ptrutil"
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
					SyncUser:                 getTestActivityConfig(),
					FetchBids:                getTestActivityConfig(),
					EnrichUserFPD:            getTestActivityConfig(),
					ReportAnalytics:          getTestActivityConfig(),
					TransmitUserFPD:          getTestActivityConfig(),
					TransmitPreciseGeo:       getTestActivityConfig(),
					TransmitUniqueRequestIds: getTestActivityConfig(),
					TransmitTids:             getTestActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: map[Activity]ActivityPlan{
				ActivitySyncUser:                 getTestActivityPlan(),
				ActivityFetchBids:                getTestActivityPlan(),
				ActivityEnrichUserFPD:            getTestActivityPlan(),
				ActivityReportAnalytics:          getTestActivityPlan(),
				ActivityTransmitUserFPD:          getTestActivityPlan(),
				ActivityTransmitPreciseGeo:       getTestActivityPlan(),
				ActivityTransmitUniqueRequestIds: getTestActivityPlan(),
				ActivityTransmitTids:             getTestActivityPlan(),
			}},
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
				ActivitySyncUser: getTestActivityPlan()}},
			activity:       ActivityFetchBids,
			target:         Component{Type: "bidder", Name: "bidderA"},
			activityResult: true,
		},
		{
			name: "activity_defined_but_not_found_default_returned",
			activityControl: ActivityControl{plans: map[Activity]ActivityPlan{
				ActivityFetchBids: getTestActivityPlan()}},
			activity:       ActivityFetchBids,
			target:         Component{Type: "bidder", Name: "bidderB"},
			activityResult: true,
		},
		{
			name: "activity_defined_and_allowed",
			activityControl: ActivityControl{plans: map[Activity]ActivityPlan{
				ActivityFetchBids: getTestActivityPlan()}},
			activity:       ActivityFetchBids,
			target:         Component{Type: "bidder", Name: "bidderA"},
			activityResult: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualResult := test.activityControl.Allow(test.activity, test.target)
			assert.Equal(t, test.activityResult, actualResult)

		})
	}
}

func getTestActivityConfig() config.Activity {
	return config.Activity{
		Default: ptrutil.ToPtr(true),
		Rules: []config.ActivityRule{
			{
				Allow: true,
				Condition: config.ActivityCondition{
					ComponentName: []string{"bidderA"},
					ComponentType: []string{"bidder"},
				},
			},
		},
	}
}

func getTestActivityPlan() ActivityPlan {
	return ActivityPlan{
		defaultResult: true,
		rules: []Rule{
			ConditionRule{
				result:        ActivityAllow,
				componentName: []string{"bidderA"},
				componentType: []string{"bidder"},
			},
		},
	}
}
