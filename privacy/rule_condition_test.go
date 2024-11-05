package privacy

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestComponentEnforcementRuleEvaluate(t *testing.T) {
	testCases := []struct {
		name           string
		componentRule  ConditionRule
		target         Component
		activityResult ActivityResult
	}{
		{
			name: "activity_is_allowed",
			componentRule: ConditionRule{
				result:        ActivityAllow,
				componentName: []string{"bidderA"},
				componentType: []string{"bidder"},
			},
			target:         Component{Type: "bidder", Name: "bidderA"},
			activityResult: ActivityAllow,
		},
		{
			name: "activity_is_not_allowed",
			componentRule: ConditionRule{
				result:        ActivityDeny,
				componentName: []string{"bidderA"},
				componentType: []string{"bidder"},
			},
			target:         Component{Type: "bidder", Name: "bidderA"},
			activityResult: ActivityDeny,
		},
		{
			name: "abstain_both_clauses_do_not_match",
			componentRule: ConditionRule{
				result:        ActivityAllow,
				componentName: []string{"bidderA"},
				componentType: []string{"bidder"},
			},
			target:         Component{Type: "bidder", Name: "bidderB"},
			activityResult: ActivityAbstain,
		},
		{
			name: "abstain_gppsid",
			componentRule: ConditionRule{
				result: ActivityAllow,
				gppSID: []int8{1},
			},
			target:         Component{Type: "bidder", Name: "bidderB"},
			activityResult: ActivityAbstain,
		},
		{
			name: "activity_is_not_allowed_componentName_only",
			componentRule: ConditionRule{
				result:        ActivityAllow,
				componentName: []string{"bidderA"},
			},
			target:         Component{Type: "bidder", Name: "bidderA"},
			activityResult: ActivityAllow,
		},
		{
			name: "activity_is_allowed_componentType_only",
			componentRule: ConditionRule{
				result:        ActivityAllow,
				componentType: []string{"bidder"},
			},
			target:         Component{Type: "bidder", Name: "bidderB"},
			activityResult: ActivityAllow,
		},
		{
			name: "no-conditions-allow",
			componentRule: ConditionRule{
				result: ActivityAllow,
			},
			target:         Component{Type: "bidder", Name: "bidderA"},
			activityResult: ActivityAllow,
		},
		{
			name: "no-conditions-deny",
			componentRule: ConditionRule{
				result: ActivityDeny,
			},
			target:         Component{Type: "bidder", Name: "bidderA"},
			activityResult: ActivityDeny,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualResult := test.componentRule.Evaluate(test.target, ActivityRequest{})
			assert.Equal(t, test.activityResult, actualResult)
		})
	}
}

func TestEvaluateGPPSID(t *testing.T) {
	testCases := []struct {
		name         string
		sidCondition []int8
		sidRequest   []int8
		expected     bool
	}{
		{
			name:         "condition-nil-request-nil",
			sidCondition: nil,
			sidRequest:   nil,
			expected:     true,
		},
		{
			name:         "condition-empty-request-nil",
			sidCondition: []int8{},
			sidRequest:   nil,
			expected:     true,
		},
		{
			name:         "condition-nil-request-empty",
			sidCondition: nil,
			sidRequest:   []int8{},
			expected:     true,
		},
		{
			name:         "condition-empty-request-empty",
			sidCondition: []int8{},
			sidRequest:   []int8{},
			expected:     true,
		},
		{
			name:         "condition-one-request-nil",
			sidCondition: []int8{1},
			sidRequest:   nil,
			expected:     false,
		},
		{
			name:         "condition-many-request-nil",
			sidCondition: []int8{1, 2},
			sidRequest:   nil,
			expected:     false,
		},
		{
			name:         "condition-one-request-empty",
			sidCondition: []int8{1},
			sidRequest:   []int8{},
			expected:     false,
		},
		{
			name:         "condition-many-request-empty",
			sidCondition: []int8{1, 2},
			sidRequest:   []int8{},
			expected:     false,
		},
		{
			name:         "condition-one-request-one-match",
			sidCondition: []int8{1},
			sidRequest:   []int8{1},
			expected:     true,
		},
		{
			name:         "condition-one-request-one-nomatch",
			sidCondition: []int8{1},
			sidRequest:   []int8{2},
			expected:     false,
		},
		{
			name:         "condition-one-request-many-match",
			sidCondition: []int8{1},
			sidRequest:   []int8{1, 2},
			expected:     true,
		},
		{
			name:         "condition-one-request-many-nomatch",
			sidCondition: []int8{3},
			sidRequest:   []int8{1, 2},
			expected:     false,
		},
		{
			name:         "condition-nil-request-one",
			sidCondition: nil,
			sidRequest:   []int8{1},
			expected:     true,
		},
		{
			name:         "condition-nil-request-many",
			sidCondition: nil,
			sidRequest:   []int8{1, 2},
			expected:     true,
		},
		{
			name:         "condition-empty-request-one",
			sidCondition: []int8{},
			sidRequest:   []int8{1},
			expected:     true,
		},
		{
			name:         "condition-empty-request-many",
			sidCondition: []int8{},
			sidRequest:   []int8{1, 2},
			expected:     true,
		},
		{
			name:         "condition-many-request-one-match",
			sidCondition: []int8{1, 2},
			sidRequest:   []int8{1},
			expected:     true,
		},
		{
			name:         "condition-many-request-one-nomatch",
			sidCondition: []int8{1, 2},
			sidRequest:   []int8{3},
			expected:     false,
		},
		{
			name:         "condition-many-request-many-match",
			sidCondition: []int8{1, 2},
			sidRequest:   []int8{1, 2},
			expected:     true,
		},
		{
			name:         "condition-many-request-many-nomatch",
			sidCondition: []int8{1, 2},
			sidRequest:   []int8{3, 4},
			expected:     false,
		},
		{
			name:         "condition-many-request-many-mixed",
			sidCondition: []int8{1, 2},
			sidRequest:   []int8{2, 3},
			expected:     true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualResult := evaluateGPPSID(test.sidCondition, NewRequestFromPolicies(Policies{GPPSID: test.sidRequest}))
			assert.Equal(t, test.expected, actualResult)
		})
	}
}

func TestGetGPPSID(t *testing.T) {
	testCases := []struct {
		name     string
		request  ActivityRequest
		expected []int8
	}{
		{
			name:     "empty",
			request:  ActivityRequest{},
			expected: nil,
		},
		{
			name:     "policies",
			request:  ActivityRequest{policies: &Policies{GPPSID: []int8{1}}},
			expected: []int8{1},
		},
		{
			name:     "request-regs",
			request:  ActivityRequest{bidRequest: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Regs: &openrtb2.Regs{GPPSID: []int8{1}}}}},
			expected: []int8{1},
		},
		{
			name:     "request-regs-nil",
			request:  ActivityRequest{bidRequest: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Regs: nil}}},
			expected: nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualResult := getGPPSID(test.request)
			assert.Equal(t, test.expected, actualResult)
		})
	}
}
