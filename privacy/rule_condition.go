package privacy

// noClausesDefinedResult represents the default return when there is no matching criteria specified.
const noClausesDefinedResult = true

type ConditionRule struct {
	result        ActivityResult
	componentName []string
	componentType []string
	gppSID        []int8
}

func (r ConditionRule) Evaluate(target Component, request ActivityRequest) ActivityResult {
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

func evaluateComponentName(target Component, componentNames []string) bool {
	// no clauses are considered a match
	if len(componentNames) == 0 {
		return noClausesDefinedResult
	}

	for _, n := range componentNames {
		if target.MatchesName(n) {
			return true
		}
	}

	return false
}

func evaluateComponentType(target Component, componentTypes []string) bool {
	if len(componentTypes) == 0 {
		return noClausesDefinedResult
	}

	// if there are clauses, at least one needs to match
	for _, t := range componentTypes {
		if target.MatchesType(t) {
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
