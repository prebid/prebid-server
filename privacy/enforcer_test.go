package privacy

import (
	"errors"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/util/ptrutil"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TemporarilyDisabledTestNewActivityControl(t *testing.T) {

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
			err:             errors.New("unable to parse condition: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy_config_is_specified_and_FetchBids_is_incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					FetchBids: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse condition: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy_config_is_specified_and_EnrichUserFPD_is_incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					EnrichUserFPD: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse condition: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy_config_is_specified_and_ReportAnalytics_is_incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					ReportAnalytics: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse condition: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy_config_is_specified_and_TransmitUserFPD_is_incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					TransmitUserFPD: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse condition: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy_config_is_specified_and_TransmitPreciseGeo_is_incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					TransmitPreciseGeo: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse condition: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy_config_is_specified_and_TransmitUniqueRequestIds_is_incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					TransmitUniqueRequestIds: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse condition: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy_config_is_specified_and_TransmitTids_is_incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					TransmitTids: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse condition: bidder.bidderA.bidderB"),
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
		expectedResult  ActivityResult
	}{
		{
			name:            "activityDefault_is_nil",
			activityDefault: nil,
			expectedResult:  ActivityAllow,
		},
		{
			name:            "activityDefault_is_true",
			activityDefault: ptrutil.ToPtr(true),
			expectedResult:  ActivityAllow,
		},
		{
			name:            "activityDefault_is_false",
			activityDefault: ptrutil.ToPtr(false),
			expectedResult:  ActivityDeny,
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
		target          ScopedName
		activityResult  ActivityResult
	}{
		{
			name:            "plans_is_nil",
			activityControl: ActivityControl{plans: nil},
			activity:        ActivityFetchBids,
			target:          ScopedName{Scope: "bidder", Name: "bidderA"},
			activityResult:  ActivityAbstain,
		},
		{
			name: "activity_not_defined",
			activityControl: ActivityControl{plans: map[Activity]ActivityPlan{
				ActivitySyncUser: getDefaultActivityPlan()}},
			activity:       ActivityFetchBids,
			target:         ScopedName{Scope: "bidder", Name: "bidderA"},
			activityResult: ActivityAbstain,
		},
		{
			name: "activity_defined_but_not_found_default_returned",
			activityControl: ActivityControl{plans: map[Activity]ActivityPlan{
				ActivityFetchBids: getDefaultActivityPlan()}},
			activity:       ActivityFetchBids,
			target:         ScopedName{Scope: "bidder", Name: "bidderB"},
			activityResult: ActivityAllow,
		},
		{
			name: "activity_defined_and_allowed",
			activityControl: ActivityControl{plans: map[Activity]ActivityPlan{
				ActivityFetchBids: getDefaultActivityPlan()}},
			activity:       ActivityFetchBids,
			target:         ScopedName{Scope: "bidder", Name: "bidderA"},
			activityResult: ActivityAllow,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualResult := test.activityControl.Allow(test.activity, test.target)
			assert.Equal(t, test.activityResult, actualResult)

		})
	}
}

func TestAllowComponentEnforcementRule(t *testing.T) {

	testCases := []struct {
		name           string
		componentRule  ComponentEnforcementRule
		target         ScopedName
		activityResult ActivityResult
	}{
		{
			name: "activity_is_allowed",
			componentRule: ComponentEnforcementRule{
				allowed: true,
				componentName: []ScopedName{
					{Scope: "bidder", Name: "bidderA"},
				},
				componentType: []string{"bidder"},
			},
			target:         ScopedName{Scope: "bidder", Name: "bidderA"},
			activityResult: ActivityAllow,
		},
		{
			name: "activity_is_not_allowed",
			componentRule: ComponentEnforcementRule{
				allowed: false,
				componentName: []ScopedName{
					{Scope: "bidder", Name: "bidderA"},
				},
				componentType: []string{"bidder"},
			},
			target:         ScopedName{Scope: "bidder", Name: "bidderA"},
			activityResult: ActivityDeny,
		},
		{
			name: "abstain_both_clauses_do_not_match",
			componentRule: ComponentEnforcementRule{
				allowed: true,
				componentName: []ScopedName{
					{Scope: "bidder", Name: "bidderA"},
				},
				componentType: []string{"bidder"},
			},
			target:         ScopedName{Scope: "bidder", Name: "bidderB"},
			activityResult: ActivityAbstain,
		},
		{
			name: "activity_is_not_allowed_componentName_only",
			componentRule: ComponentEnforcementRule{
				allowed: true,
				componentName: []ScopedName{
					{Scope: "bidder", Name: "bidderA"},
				},
			},
			target:         ScopedName{Scope: "bidder", Name: "bidderA"},
			activityResult: ActivityAllow,
		},
		{
			name: "activity_is_allowed_componentType_only",
			componentRule: ComponentEnforcementRule{
				allowed:       true,
				componentType: []string{"bidder"},
			},
			target:         ScopedName{Scope: "bidder", Name: "bidderB"},
			activityResult: ActivityAllow,
		},
		{
			name: "abstain_activity_no_componentType_and_no_componentName",
			componentRule: ComponentEnforcementRule{
				allowed: true,
			},
			target:         ScopedName{Scope: "bidder", Name: "bidderB"},
			activityResult: ActivityAbstain,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualResult := test.componentRule.Allow(test.target)
			assert.Equal(t, test.activityResult, actualResult)

		})
	}
}

func TestNewScopedName(t *testing.T) {

	testCases := []struct {
		name              string
		condition         string
		expectedScopeName ScopedName
		err               error
	}{
		{
			name:              "condition_is_empty",
			condition:         "",
			expectedScopeName: ScopedName{},
			err:               errors.New("unable to parse empty condition"),
		},
		{
			name:              "condition_is_incorrect",
			condition:         "bidder.bidderA.bidderB",
			expectedScopeName: ScopedName{},
			err:               errors.New("unable to parse condition: bidder.bidderA.bidderB"),
		},
		{
			name:              "condition_is_scoped_to_bidder",
			condition:         "bidder.bidderA",
			expectedScopeName: ScopedName{Scope: "bidder", Name: "bidderA"},
			err:               nil,
		},
		{
			name:              "condition_is_scoped_to_analytics",
			condition:         "analytics.bidderA",
			expectedScopeName: ScopedName{Scope: "analytics", Name: "bidderA"},
			err:               nil,
		},
		{
			name:              "condition_is_scoped_to_userid",
			condition:         "userid.bidderA",
			expectedScopeName: ScopedName{Scope: "userid", Name: "bidderA"},
			err:               nil,
		},
		{
			name:              "condition_is_bidder_name",
			condition:         "bidderA",
			expectedScopeName: ScopedName{Scope: "bidder", Name: "bidderA"},
			err:               nil,
		},
		{
			name:              "condition_is_module_tag_rtd",
			condition:         "rtd.test",
			expectedScopeName: ScopedName{Scope: "rtd", Name: "test"},
			err:               nil,
		},
		{
			name:              "condition_scope_defaults_to_genera",
			condition:         "test.test",
			expectedScopeName: ScopedName{Scope: "general", Name: "test"},
			err:               nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualSN, actualErr := NewScopedName(test.condition)
			if test.err == nil {
				assert.Equal(t, test.expectedScopeName, actualSN)
				assert.NoError(t, actualErr)
			} else {
				assert.EqualError(t, actualErr, test.err.Error())
			}
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
		defaultResult: ActivityAllow,
		rules: []ActivityRule{
			ComponentEnforcementRule{
				allowed: true,
				componentName: []ScopedName{
					{Scope: "bidder", Name: "bidderA"},
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
