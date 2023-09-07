package privacy

import (
	"strings"
)

type ConditionRule struct {
	result        ActivityResult
	componentName []string
	componentType []string
}

func (r ConditionRule) Evaluate(target Component) ActivityResult {
	if matched := evaluateComponentName(target, r.componentName); !matched {
		return ActivityAbstain
	}

	if matched := evaluateComponentType(target, r.componentType); !matched {
		return ActivityAbstain
	}

	return r.result
}

func evaluateComponentName(target Component, componentNames []string) bool {
	// no clauses are considered a match
	if len(componentNames) == 0 {
		return true
	}

	// if there are clauses, at least one needs to match
	for _, n := range componentNames {
		if target.MatchesName(n) {
			return true
		}
	}

	return false
}

func evaluateComponentType(target Component, componentTypes []string) bool {
	// no clauses are considered a match
	if len(componentTypes) == 0 {
		return true
	}

	// if there are clauses, at least one needs to match
	for _, s := range componentTypes {
		if strings.EqualFold(s, target.Type) {
			return true
		}
	}

	return false
}
