package privacy

import (
	"fmt"
	"strings"

	"github.com/prebid/prebid-server/config"
)

type ActivityResult int

const (
	ActivityAbstain ActivityResult = iota
	ActivityAllow
	ActivityDeny
)

const (
	ScopeTypeBidder    = "bidder"
	ScopeTypeAnalytics = "analytics"
	ScopeTypeRTD       = "rtd" // real time data
	ScopeTypeUserID    = "userid"
	ScopeTypeGeneral   = "general"
)

type ActivityControl struct {
	plans map[Activity]ActivityPlan
}

func NewActivityControl(privacyConf *config.AccountPrivacy) (ActivityControl, error) {
	ac := ActivityControl{}
	var err error

	if privacyConf == nil {
		return ac, nil
	}

	plans := make(map[Activity]ActivityPlan)

	plans[ActivitySyncUser], err = buildEnforcementPlan(privacyConf.AllowActivities.SyncUser)
	if err != nil {
		return ac, err
	}
	plans[ActivityFetchBids], err = buildEnforcementPlan(privacyConf.AllowActivities.FetchBids)
	if err != nil {
		return ac, err
	}
	plans[ActivityEnrichUserFPD], err = buildEnforcementPlan(privacyConf.AllowActivities.EnrichUserFPD)
	if err != nil {
		return ac, err
	}
	plans[ActivityReportAnalytics], err = buildEnforcementPlan(privacyConf.AllowActivities.ReportAnalytics)
	if err != nil {
		return ac, err
	}
	plans[ActivityTransmitUserFPD], err = buildEnforcementPlan(privacyConf.AllowActivities.TransmitUserFPD)
	if err != nil {
		return ac, err
	}
	plans[ActivityTransmitPreciseGeo], err = buildEnforcementPlan(privacyConf.AllowActivities.TransmitPreciseGeo)
	if err != nil {
		return ac, err
	}
	plans[ActivityTransmitUniqueRequestIds], err = buildEnforcementPlan(privacyConf.AllowActivities.TransmitUniqueRequestIds)
	if err != nil {
		return ac, err
	}
	plans[ActivityTransmitTids], err = buildEnforcementPlan(privacyConf.AllowActivities.TransmitTids)
	if err != nil {
		return ac, err
	}

	ac.plans = plans

	return ac, nil
}

func buildEnforcementPlan(activity config.Activity) (ActivityPlan, error) {
	ef := ActivityPlan{}
	rules, err := activityRulesToEnforcementRules(activity.Rules)
	if err != nil {
		return ef, err
	}
	ef.defaultResult = activityDefaultToDefaultResult(activity.Default)
	ef.rules = rules
	return ef, nil
}

func activityRulesToEnforcementRules(rules []config.ActivityRule) ([]ActivityRule, error) {
	var enfRules []ActivityRule

	for _, r := range rules {
		var result ActivityResult
		if r.Allow {
			result = ActivityAllow
		} else {
			result = ActivityDeny
		}

		componentName, err := conditionToRuleComponentNames(r.Condition.ComponentName)
		if err != nil {
			return nil, err
		}

		er := ComponentEnforcementRule{
			result:        result,
			componentName: componentName,
			componentType: r.Condition.ComponentType,
		}
		enfRules = append(enfRules, er)
	}
	return enfRules, nil
}

func conditionToRuleComponentNames(conditions []string) ([]ScopedName, error) {
	sn := make([]ScopedName, 0)
	for _, condition := range conditions {
		scope, err := NewScopedName(condition)
		if err != nil {
			return sn, err
		}
		sn = append(sn, scope)
	}
	return sn, nil
}

func activityDefaultToDefaultResult(activityDefault *bool) ActivityResult {
	if activityDefault == nil {
		// if default is unspecified, the hardcoded default-default is true.
		return ActivityAllow
	} else if *activityDefault {
		return ActivityAllow
	}
	return ActivityDeny
}

func (e ActivityControl) Evaluate(activity Activity, target ScopedName) ActivityResult {
	plan, planDefined := e.plans[activity]

	if !planDefined {
		return ActivityAbstain
	}

	return plan.Evaluate(target)
}

type ActivityPlan struct {
	defaultResult ActivityResult
	rules         []ActivityRule
}

func (p ActivityPlan) Evaluate(target ScopedName) ActivityResult {
	for _, rule := range p.rules {
		result := rule.Evaluate(target)
		if result == ActivityDeny || result == ActivityAllow {
			return result
		}
	}
	return p.defaultResult
}

type ActivityRule interface {
	Evaluate(target ScopedName) ActivityResult
}

type ComponentEnforcementRule struct {
	result        ActivityResult
	componentName []ScopedName
	componentType []string
}

func (r ComponentEnforcementRule) Evaluate(target ScopedName) ActivityResult {
	if matched := evaluateComponentName(target, r.componentName); !matched {
		return ActivityAbstain
	}

	if matched := evaluateComponentType(target, r.componentType); !matched {
		return ActivityAbstain
	}

	return r.result
}

func evaluateComponentName(target ScopedName, componentNames []ScopedName) bool {
	// no clauses are considered a match
	if len(componentNames) == 0 {
		return true
	}

	// if there are clauses, at least one needs to match
	for _, n := range componentNames {
		if strings.EqualFold(n.Scope, target.Scope) && (n.Name == "*" || strings.EqualFold(n.Name, target.Name)) {
			return true
		}
	}

	return false
}

func evaluateComponentType(target ScopedName, componentTypes []string) bool {
	// no clauses are considered a match
	if len(componentTypes) == 0 {
		return true
	}

	// if there are clauses, at least one needs to match
	for _, s := range componentTypes {
		if strings.EqualFold(s, target.Scope) {
			return true
		}
	}

	return false
}

type ScopedName struct {
	Scope string
	Name  string
}

func NewScopedName(condition string) (ScopedName, error) {
	if condition == "" {
		return ScopedName{}, fmt.Errorf("unable to parse empty condition")
	}
	var scope, name string
	split := strings.Split(condition, ".")
	if len(split) == 2 {
		s := strings.ToLower(split[0])
		if s == ScopeTypeBidder || s == ScopeTypeAnalytics || s == ScopeTypeUserID {
			scope = s
		} else if strings.Contains(s, ScopeTypeRTD) {
			scope = ScopeTypeRTD
		} else {
			scope = ScopeTypeGeneral
		}
		name = split[1]
	} else if len(split) == 1 {
		scope = ScopeTypeBidder
		name = split[0]
	} else {
		return ScopedName{}, fmt.Errorf("unable to parse condition: %s", condition)
	}

	return ScopedName{
		Scope: scope,
		Name:  name,
	}, nil
}
