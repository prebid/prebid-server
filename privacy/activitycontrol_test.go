package privacy

import (
	"errors"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestNewActivityControl(t *testing.T) {
	testCases := []struct {
		name            string
		privacyConf     *config.AccountPrivacy
		activityControl ActivityControl
		err             error
	}{
		{
			name:            "privacy_config_is_nil",
			privacyConf:     nil,
			activityControl: ActivityControl{plans: nil},
			err:             nil,
		},
		{
			name: "privacy_config_is_specified_and_correct",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					SyncUser:                 getDefaultActivityConfig(),
					FetchBids:                getDefaultActivityConfig(),
					EnrichUserFPD:            getDefaultActivityConfig(),
					ReportAnalytics:          getDefaultActivityConfig(),
					TransmitUserFPD:          getDefaultActivityConfig(),
					TransmitPreciseGeo:       getDefaultActivityConfig(),
					TransmitUniqueRequestIds: getDefaultActivityConfig(),
					TransmitTids:             getDefaultActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: map[Activity]ActivityPlan{
				ActivitySyncUser:                 getDefaultActivityPlan(),
				ActivityFetchBids:                getDefaultActivityPlan(),
				ActivityEnrichUserFPD:            getDefaultActivityPlan(),
				ActivityReportAnalytics:          getDefaultActivityPlan(),
				ActivityTransmitUserFPD:          getDefaultActivityPlan(),
				ActivityTransmitPreciseGeo:       getDefaultActivityPlan(),
				ActivityTransmitUniqueRequestIds: getDefaultActivityPlan(),
				ActivityTransmitTids:             getDefaultActivityPlan(),
			}},
			err: nil,
		},
		{
			name: "privacy_config_is_specified_and_SyncUser_is_incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					SyncUser: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse component: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy_config_is_specified_and_FetchBids_is_incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					FetchBids: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse component: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy_config_is_specified_and_EnrichUserFPD_is_incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					EnrichUserFPD: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse component: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy_config_is_specified_and_ReportAnalytics_is_incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					ReportAnalytics: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse component: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy_config_is_specified_and_TransmitUserFPD_is_incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					TransmitUserFPD: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse component: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy_config_is_specified_and_TransmitPreciseGeo_is_incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					TransmitPreciseGeo: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse component: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy_config_is_specified_and_TransmitUniqueRequestIds_is_incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					TransmitUniqueRequestIds: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse component: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy_config_is_specified_and_TransmitTids_is_incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					TransmitTids: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse component: bidder.bidderA.bidderB"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualAC, actualErr := NewActivityControl(test.privacyConf)
			if test.err == nil {
				assert.Equal(t, test.activityControl, actualAC)
				assert.NoError(t, actualErr)
			} else {
				assert.EqualError(t, actualErr, test.err.Error())
			}
		})
	}
}

func TestActivityDefaultToDefaultResult(t *testing.T) {
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
			actualResult := activityDefaultToDefaultResult(test.activityDefault)
			assert.Equal(t, test.expectedResult, actualResult)
		})
	}
}

func TestAllowActivityControl(t *testing.T) {

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
				ActivitySyncUser: getDefaultActivityPlan()}},
			activity:       ActivityFetchBids,
			target:         Component{Type: "bidder", Name: "bidderA"},
			activityResult: true,
		},
		{
			name: "activity_defined_but_not_found_default_returned",
			activityControl: ActivityControl{plans: map[Activity]ActivityPlan{
				ActivityFetchBids: getDefaultActivityPlan()}},
			activity:       ActivityFetchBids,
			target:         Component{Type: "bidder", Name: "bidderB"},
			activityResult: true,
		},
		{
			name: "activity_defined_and_allowed",
			activityControl: ActivityControl{plans: map[Activity]ActivityPlan{
				ActivityFetchBids: getDefaultActivityPlan()}},
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

// constants
func getDefaultActivityConfig() config.Activity {
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

func getDefaultActivityPlan() ActivityPlan {
	return ActivityPlan{
		defaultResult: true,
		rules: []Rule{
			ComponentEnforcementRule{
				result: ActivityAllow,
				componentName: []Component{
					{Type: "bidder", Name: "bidderA"},
				},
				componentType: []string{"bidder"},
			},
		},
	}
}

func getIncorrectActivityConfig() config.Activity {
	return config.Activity{
		Default: ptrutil.ToPtr(true),
		Rules: []config.ActivityRule{
			{
				Allow: true,
				Condition: config.ActivityCondition{
					ComponentName: []string{"bidder.bidderA.bidderB"},
					ComponentType: []string{"bidder"},
				},
			},
		},
	}
}
