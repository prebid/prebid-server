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
		target          ScopedName
		activityResult  bool
	}{
		{
			name:            "plans_is_nil",
			activityControl: ActivityControl{plans: nil},
			activity:        ActivityFetchBids,
			target:          ScopedName{Scope: "bidder", Name: "bidderA"},
			activityResult:  true,
		},
		{
			name: "activity_not_defined",
			activityControl: ActivityControl{plans: map[Activity]ActivityPlan{
				ActivitySyncUser: getDefaultActivityPlan()}},
			activity:       ActivityFetchBids,
			target:         ScopedName{Scope: "bidder", Name: "bidderA"},
			activityResult: true,
		},
		{
			name: "activity_defined_but_not_found_default_returned",
			activityControl: ActivityControl{plans: map[Activity]ActivityPlan{
				ActivityFetchBids: getDefaultActivityPlan()}},
			activity:       ActivityFetchBids,
			target:         ScopedName{Scope: "bidder", Name: "bidderB"},
			activityResult: true,
		},
		{
			name: "activity_defined_and_allowed",
			activityControl: ActivityControl{plans: map[Activity]ActivityPlan{
				ActivityFetchBids: getDefaultActivityPlan()}},
			activity:       ActivityFetchBids,
			target:         ScopedName{Scope: "bidder", Name: "bidderA"},
			activityResult: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualResult := test.activityControl.Evaluate(test.activity, test.target)
			assert.Equal(t, test.activityResult, actualResult)

		})
	}
}

func TestComponentEnforcementRuleEvaluate(t *testing.T) {
	testCases := []struct {
		name           string
		componentRule  ComponentEnforcementRule
		target         ScopedName
		activityResult ActivityResult
	}{
		{
			name: "activity_is_allowed",
			componentRule: ComponentEnforcementRule{
				result: ActivityAllow,
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
				result: ActivityDeny,
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
				result: ActivityAllow,
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
				result: ActivityAllow,
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
				result:        ActivityAllow,
				componentType: []string{"bidder"},
			},
			target:         ScopedName{Scope: "bidder", Name: "bidderB"},
			activityResult: ActivityAllow,
		},
		{
			name: "no-conditions-allow",
			componentRule: ComponentEnforcementRule{
				result: ActivityAllow,
			},
			target:         ScopedName{Scope: "bidder", Name: "bidderA"},
			activityResult: ActivityAllow,
		},
		{
			name: "no-conditions-deny",
			componentRule: ComponentEnforcementRule{
				result: ActivityDeny,
			},
			target:         ScopedName{Scope: "bidder", Name: "bidderA"},
			activityResult: ActivityDeny,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualResult := test.componentRule.Evaluate(test.target)
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
			name:              "empty",
			condition:         "",
			expectedScopeName: ScopedName{},
			err:               errors.New("unable to parse empty condition"),
		},
		{
			name:              "incorrect",
			condition:         "bidder.bidderA.bidderB",
			expectedScopeName: ScopedName{},
			err:               errors.New("unable to parse condition: bidder.bidderA.bidderB"),
		},
		{
			name:              "scope-bidder",
			condition:         "bidder.bidderA",
			expectedScopeName: ScopedName{Scope: "bidder", Name: "bidderA"},
			err:               nil,
		},
		{
			name:              "scope-analytics",
			condition:         "analytics.bidderA",
			expectedScopeName: ScopedName{Scope: "analytics", Name: "bidderA"},
			err:               nil,
		},
		{
			name:              "scope-userid",
			condition:         "userid.bidderA",
			expectedScopeName: ScopedName{Scope: "userid", Name: "bidderA"},
			err:               nil,
		},
		{
			name:              "scope-default",
			condition:         "bidderA",
			expectedScopeName: ScopedName{Scope: "bidder", Name: "bidderA"},
			err:               nil,
		},
		{
			name:              "scope-rtf",
			condition:         "rtd.test",
			expectedScopeName: ScopedName{Scope: "rtd", Name: "test"},
			err:               nil,
		},
		{
			name:              "scope-general",
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
		defaultResult: true,
		rules: []ActivityRule{
			ComponentEnforcementRule{
				result: ActivityAllow,
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
