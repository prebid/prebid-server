package privacy

import (
	"errors"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/util/ptrutil"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewActivityControl(t *testing.T) {

	testCases := []struct {
		name            string
		privacyConf     *config.AccountPrivacy
		activityControl ActivityControl
		err             error
	}{
		{
			name:            "privacy config is nil",
			privacyConf:     nil,
			activityControl: ActivityControl{plans: nil},
			err:             nil,
		},
		{
			name: "privacy config is specified and correct",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					SyncUser:                 getDefaultActivityConfig(),
					FetchBids:                getDefaultActivityConfig(),
					EnrichUserFPD:            getDefaultActivityConfig(),
					ReportAnalytics:          getDefaultActivityConfig(),
					TransmitUserFPD:          getDefaultActivityConfig(),
					TransmitPreciseGeo:       getDefaultActivityConfig(),
					TransmitUniqueRequestIds: getDefaultActivityConfig(),
					TransmitTIds:             getDefaultActivityConfig(),
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
				ActivityTransmitTIds:             getDefaultActivityPlan(),
			}},
			err: nil,
		},
		{
			name: "privacy config is specified and SyncUser is incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					SyncUser: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse condition: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy config is specified and FetchBids is incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					FetchBids: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse condition: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy config is specified and EnrichUserFPD is incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					EnrichUserFPD: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse condition: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy config is specified and ReportAnalytics is incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					ReportAnalytics: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse condition: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy config is specified and TransmitUserFPD is incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					TransmitUserFPD: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse condition: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy config is specified and TransmitPreciseGeo is incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					TransmitPreciseGeo: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse condition: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy config is specified and TransmitUniqueRequestIds is incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					TransmitUniqueRequestIds: getIncorrectActivityConfig(),
				},
			},
			activityControl: ActivityControl{plans: nil},
			err:             errors.New("unable to parse condition: bidder.bidderA.bidderB"),
		},
		{
			name: "privacy config is specified and TransmitTIds is incorrect",
			privacyConf: &config.AccountPrivacy{
				AllowActivities: config.AllowActivities{
					TransmitTIds: getIncorrectActivityConfig(),
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
				assert.Equal(t, test.activityControl, actualAC, "incorrect activity control")
				assert.NoError(t, actualErr, "error should be nil")
			} else {
				assert.EqualError(t, actualErr, test.err.Error(), "error is incorrect")
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
			name:            "activityDefault is nil",
			activityDefault: nil,
			expectedResult:  ActivityAllow,
		},
		{
			name:            "activityDefault is nil",
			activityDefault: ptrutil.ToPtr(true),
			expectedResult:  ActivityAllow,
		},
		{
			name:            "activityDefault is nil",
			activityDefault: ptrutil.ToPtr(false),
			expectedResult:  ActivityDeny,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualResult := activityDefaultToDefaultResult(test.activityDefault)
			assert.Equal(t, actualResult, test.expectedResult, "result is incorrect")

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
			name:            "plans is nil",
			activityControl: ActivityControl{plans: nil},
			activity:        ActivityFetchBids,
			target:          ScopedName{Scope: "bidder", Name: "bidderA"},
			activityResult:  ActivityAbstain,
		},
		{
			name: "activity not defined",
			activityControl: ActivityControl{plans: map[Activity]ActivityPlan{
				ActivitySyncUser: getDefaultActivityPlan()}},
			activity:       ActivityFetchBids,
			target:         ScopedName{Scope: "bidder", Name: "bidderA"},
			activityResult: ActivityAbstain,
		},
		{
			name: "activity defined but not allowed",
			activityControl: ActivityControl{plans: map[Activity]ActivityPlan{
				ActivityFetchBids: getDefaultActivityPlan()}},
			activity:       ActivityFetchBids,
			target:         ScopedName{Scope: "bidder", Name: "bidderB"},
			activityResult: ActivityAllow,
		},
		{
			name: "activity defined and allowed",
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
			assert.Equal(t, test.activityResult, actualResult, "incorrect allow activity result")

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
			name: "activity is allowed",
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
			name: "activity is not allowed",
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
			name: "activity is not found",
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
			name: "activity is not allowed, componentName only",
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
			name: "activity is allowed, componentType only",
			componentRule: ComponentEnforcementRule{
				allowed:       true,
				componentType: []string{"bidder"},
			},
			target:         ScopedName{Scope: "bidder", Name: "bidderB"},
			activityResult: ActivityAllow,
		},
		{
			name: "activity is allowed, no componentType and no componentName",
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
			assert.Equal(t, test.activityResult, actualResult, "incorrect allow activity result")

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
			name:              "condition is empty",
			condition:         "",
			expectedScopeName: ScopedName{},
			err:               errors.New("unable to parse empty condition"),
		},
		{
			name:              "condition is incorrect",
			condition:         "bidder.bidderA.bidderB",
			expectedScopeName: ScopedName{},
			err:               errors.New("unable to parse condition: bidder.bidderA.bidderB"),
		},
		{
			name:              "condition is correct with separator",
			condition:         "bidder.bidderA",
			expectedScopeName: ScopedName{Scope: "bidder", Name: "bidderA"},
			err:               nil,
		},
		{
			name:              "condition is bidder name",
			condition:         "bidderA",
			expectedScopeName: ScopedName{Scope: "bidder", Name: "bidderA"},
			err:               nil,
		},
		{
			name:              "condition is bidder name",
			condition:         "rtd.test",
			expectedScopeName: ScopedName{Scope: "rtd", Name: "test"},
			err:               nil,
		},
		{
			name:              "condition is bidder name",
			condition:         "test.test",
			expectedScopeName: ScopedName{Scope: "general", Name: "test"},
			err:               nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualSN, actualErr := NewScopedName(test.condition)
			if test.err == nil {
				assert.Equal(t, test.expectedScopeName, actualSN, "incorrect activity control")
				assert.NoError(t, actualErr, "error should be nil")
			} else {
				assert.EqualError(t, actualErr, test.err.Error(), "error is incorrect")
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
