package privacy

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestComponentEnforcementRuleEvaluate(t *testing.T) {
	var (
		target  = Component{Type: "bidder", Name: "bidderA"}
		request = NewRequestFromPolicies(Policies{GPPSID: []int8{1}, GPC: "1"})
	)

	testCases := []struct {
		name           string
		rule           ConditionRule
		activityResult ActivityResult
	}{
		{
			name: "all-match-allow",
			rule: ConditionRule{
				result:        ActivityAllow,
				componentName: []string{"bidderA"},
				componentType: []string{"bidder"},
				gppSID:        []int8{1},
				gpc:           "1",
			},
			activityResult: ActivityAllow,
		},
		{
			name: "all-match-deny",
			rule: ConditionRule{
				result:        ActivityDeny,
				componentName: []string{"bidderA"},
				componentType: []string{"bidder"},
				gppSID:        []int8{1},
				gpc:           "1",
			},
			activityResult: ActivityDeny,
		},
		{
			name: "no-conditions-allow",
			rule: ConditionRule{
				result: ActivityAllow,
			},
			activityResult: ActivityAllow,
		},
		{
			name: "no-conditions-deny",
			rule: ConditionRule{
				result: ActivityDeny,
			},
			activityResult: ActivityDeny,
		},
		{
			name: "mismatch-componentname",
			rule: ConditionRule{
				result:        ActivityAllow,
				componentName: []string{"mismatch"},
			},
			activityResult: ActivityAbstain,
		},
		{
			name: "mismatch-componenttype",
			rule: ConditionRule{
				result:        ActivityAllow,
				componentType: []string{"mismatch"},
			},
			activityResult: ActivityAbstain,
		},
		{
			name: "mismatch-gppsid",
			rule: ConditionRule{
				result: ActivityAllow,
				gppSID: []int8{2},
			},
			activityResult: ActivityAbstain,
		},
		{
			name: "mismatch-gpc",
			rule: ConditionRule{
				result: ActivityAllow,
				gpc:    "mismatch",
			},
			activityResult: ActivityAbstain,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualResult := test.rule.Evaluate(target, request)
			assert.Equal(t, test.activityResult, actualResult)
		})
	}
}

func TestEvaluateComponentName(t *testing.T) {
	target := Component{Type: "bidder", Name: "bidderA"}

	testCases := []struct {
		name           string
		componentNames []string
		expected       bool
	}{
		{
			name:           "nil",
			componentNames: nil,
			expected:       true,
		},
		{
			name:           "none",
			componentNames: []string{},
			expected:       true,
		},
		{
			name:           "one-match-same-case",
			componentNames: []string{"bidderA"},
			expected:       true,
		},
		{
			name:           "one-different-case",
			componentNames: []string{"BIDDERA"},
			expected:       true,
		},
		{
			name:           "one-no-match",
			componentNames: []string{"nomatch"},
			expected:       false,
		},
		{
			name:           "many",
			componentNames: []string{"nomatch1", "bidderA", "nomatch2"},
			expected:       true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualResult := evaluateComponentName(target, test.componentNames)
			assert.Equal(t, test.expected, actualResult)
		})
	}
}

func TestEvaluateComponentType(t *testing.T) {
	target := Component{Type: "bidder", Name: "bidderA"}

	testCases := []struct {
		name           string
		componentTypes []string
		expected       bool
	}{
		{
			name:           "nil",
			componentTypes: nil,
			expected:       true,
		},
		{
			name:           "none",
			componentTypes: []string{},
			expected:       true,
		},
		{
			name:           "one-match-same-case",
			componentTypes: []string{"bidder"},
			expected:       true,
		},
		{
			name:           "one-different-case",
			componentTypes: []string{"BIDDER"},
			expected:       true,
		},
		{
			name:           "one-no-match",
			componentTypes: []string{"nomatch"},
			expected:       false,
		},
		{
			name:           "many",
			componentTypes: []string{"nomatch1", "bidder", "nomatch2"},
			expected:       true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualResult := evaluateComponentType(target, test.componentTypes)
			assert.Equal(t, test.expected, actualResult)
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

func TestEvaluateGPC(t *testing.T) {
	testCases := []struct {
		name      string
		condition string
		request   string
		expected  bool
	}{
		{
			name:      "empty",
			condition: "",
			request:   "",
			expected:  true,
		},
		{
			name:      "condition-empty",
			condition: "",
			request:   "1",
			expected:  true,
		},
		{
			name:      "request-empty",
			condition: "1",
			request:   "",
			expected:  false,
		},
		{
			name:      "match",
			condition: "1",
			request:   "1",
			expected:  true,
		},
		{
			name:      "no-match",
			condition: "1",
			request:   "2",
			expected:  false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualResult := evaluateGPC(test.condition, NewRequestFromPolicies(Policies{GPC: test.request}))
			assert.Equal(t, test.expected, actualResult)
		})
	}
}

func TestGetGPC(t *testing.T) {
	testCases := []struct {
		name     string
		request  ActivityRequest
		expected string
	}{
		{
			name:     "empty",
			request:  ActivityRequest{},
			expected: "",
		},
		{
			name:     "policies",
			request:  ActivityRequest{policies: &Policies{GPC: "1"}},
			expected: "1",
		},
		{
			name:     "request-regs",
			request:  ActivityRequest{bidRequest: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"gpc":"1"}`)}}}},
			expected: "1",
		},
		{
			name:     "request-regs-nil",
			request:  ActivityRequest{bidRequest: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Regs: nil}}},
			expected: "",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualResult := getGPC(test.request)
			assert.Equal(t, test.expected, actualResult)
		})
	}
}
