package privacy

import (
	"strings"
)

type Rule interface {
	Evaluate(target Component) ActivityResult
}

type ComponentEnforcementRule struct {
	result        ActivityResult
	componentName []Component
	componentType []string
}

func (r ComponentEnforcementRule) Evaluate(target Component) ActivityResult {
	if matched := evaluateComponentName(target, r.componentName); !matched {
		return ActivityAbstain
	}

	if matched := evaluateComponentType(target, r.componentType); !matched {
		return ActivityAbstain
	}

	return r.result
}

func evaluateComponentName(target Component, componentNames []Component) bool {
	// no clauses are considered a match
	if len(componentNames) == 0 {
		return true
	}

	// if there are clauses, at least one needs to match
	for _, n := range componentNames {
		if n.Matches(target) {
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
