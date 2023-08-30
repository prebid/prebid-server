package privacy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComponentEnforcementRuleEvaluate(t *testing.T) {
	testCases := []struct {
		name           string
		componentRule  ComponentEnforcementRule
		target         Component
		activityResult ActivityResult
	}{
		{
			name: "activity_is_allowed",
			componentRule: ComponentEnforcementRule{
				result: ActivityAllow,
				componentName: []Component{
					{Type: "bidder", Name: "bidderA"},
				},
				componentType: []string{"bidder"},
			},
			target:         Component{Type: "bidder", Name: "bidderA"},
			activityResult: ActivityAllow,
		},
		{
			name: "activity_is_not_allowed",
			componentRule: ComponentEnforcementRule{
				result: ActivityDeny,
				componentName: []Component{
					{Type: "bidder", Name: "bidderA"},
				},
				componentType: []string{"bidder"},
			},
			target:         Component{Type: "bidder", Name: "bidderA"},
			activityResult: ActivityDeny,
		},
		{
			name: "abstain_both_clauses_do_not_match",
			componentRule: ComponentEnforcementRule{
				result: ActivityAllow,
				componentName: []Component{
					{Type: "bidder", Name: "bidderA"},
				},
				componentType: []string{"bidder"},
			},
			target:         Component{Type: "bidder", Name: "bidderB"},
			activityResult: ActivityAbstain,
		},
		{
			name: "activity_is_not_allowed_componentName_only",
			componentRule: ComponentEnforcementRule{
				result: ActivityAllow,
				componentName: []Component{
					{Type: "bidder", Name: "bidderA"},
				},
			},
			target:         Component{Type: "bidder", Name: "bidderA"},
			activityResult: ActivityAllow,
		},
		{
			name: "activity_is_allowed_componentType_only",
			componentRule: ComponentEnforcementRule{
				result:        ActivityAllow,
				componentType: []string{"bidder"},
			},
			target:         Component{Type: "bidder", Name: "bidderB"},
			activityResult: ActivityAllow,
		},
		{
			name: "no-conditions-allow",
			componentRule: ComponentEnforcementRule{
				result: ActivityAllow,
			},
			target:         Component{Type: "bidder", Name: "bidderA"},
			activityResult: ActivityAllow,
		},
		{
			name: "no-conditions-deny",
			componentRule: ComponentEnforcementRule{
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
