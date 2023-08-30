package privacy

import (
	"strings"
)

type Rule interface {
	Evaluate(target Component, request ActivityRequest) ActivityResult
}

type ComponentEnforcementRule struct {
	result        ActivityResult
	componentName []Component
	componentType []string
	gppSID        []int8
}

// noClausesDefinedResult represents the default return when there is no matching criteria specified.
const noClausesDefinedResult = true

func (r ComponentEnforcementRule) Evaluate(target Component, request ActivityRequest) ActivityResult {
	if matched := evaluateComponentName(target, r.componentName); !matched {
		return ActivityAbstain
	}

	if matched := evaluateComponentType(target, r.componentType); !matched {
		return ActivityAbstain
	}

	if matched := evaluateGPPSID(r.gppSID, request); !matched {
		return ActivityAbstain
	}

	return r.result
}

func evaluateComponentName(target Component, componentNames []Component) bool {
	if len(componentNames) == 0 {
		return noClausesDefinedResult
	}

	for _, n := range componentNames {
		if n.Matches(target) {
			return true
		}
	}

	return false
}

func evaluateComponentType(target Component, componentTypes []string) bool {
	if len(componentTypes) == 0 {
		return noClausesDefinedResult
	}

	for _, s := range componentTypes {
		if strings.EqualFold(s, target.Type) {
			return true
		}
	}

	return false
}

func evaluateGPPSID(sid []int8, request ActivityRequest) bool {
	if len(sid) == 0 {
		return noClausesDefinedResult
	}

	for _, x := range getGPPSID(request) {
		for _, y := range sid {
			if x == y {
				return true
			}
		}
	}
	return false
}

func getGPPSID(request ActivityRequest) []int8 {
	if request.IsPolicies() {
		return request.policies.GPPSID
	}

	if request.IsBidRequest() && request.bidRequest.Regs != nil {
		return request.bidRequest.Regs.GPPSID
	}

	return nil
}
