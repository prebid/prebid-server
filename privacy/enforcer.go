package privacy

import (
	"fmt"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"strings"
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
		return ac, err
	} else {
		//temporarily disable Activities if they are specified at the account level
		return ac, &errortypes.Warning{Message: "account.Privacy has no effect as the feature is under development."}
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
	enfRules := make([]ActivityRule, 0)
	for _, r := range rules {
		cmpName, err := conditionToRuleComponentName(r.Condition.ComponentName)
		if err != nil {
			return nil, err
		}
		er := ComponentEnforcementRule{
			allowed:       r.Allow,
			componentName: cmpName,
			componentType: r.Condition.ComponentType,
		}
		enfRules = append(enfRules, er)
	}
	return enfRules, nil
}

func conditionToRuleComponentName(conditions []string) ([]ScopedName, error) {
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

func (e ActivityControl) Allow(activity Activity, target ScopedName) ActivityResult {
	plan, planDefined := e.plans[activity]

	if !planDefined {
		return ActivityAbstain
	}

	return plan.Allow(target)
}

type ActivityPlan struct {
	defaultResult ActivityResult
	rules         []ActivityRule
}

func (p ActivityPlan) Allow(target ScopedName) ActivityResult {
	for _, rule := range p.rules {
		result := rule.Allow(target)
		if result == ActivityDeny || result == ActivityAllow {
			return result
		}
	}
	return p.defaultResult
}

type ActivityRule interface {
	Allow(target ScopedName) ActivityResult
}

type ComponentEnforcementRule struct {
	componentName []ScopedName
	componentType []string
	// include gppSectionId from 3.5
	// include geo from 3.5
	allowed bool
}

func (r ComponentEnforcementRule) Allow(target ScopedName) ActivityResult {
	if len(r.componentName) == 0 && len(r.componentType) == 0 {
		return ActivityAbstain
	}

	nameClauseExists := len(r.componentName) > 0
	typeClauseExists := len(r.componentType) > 0

	componentNameFound := false
	for _, scope := range r.componentName {
		if strings.EqualFold(scope.Scope, target.Scope) &&
			(scope.Name == "*" || strings.EqualFold(scope.Name, target.Name)) {
			componentNameFound = true
			break
		}
	}

	componentTypeFound := false
	for _, componentType := range r.componentType {
		if strings.EqualFold(componentType, target.Scope) {
			componentTypeFound = true
			break
		}
	}
	// behavior if rule matches: can be either true=allow or false=deny. result is abstain if the rule doesn't match
	matchFound := (componentNameFound || !nameClauseExists) && (componentTypeFound || !typeClauseExists)
	if matchFound {
		if r.allowed {
			return ActivityAllow
		} else {
			return ActivityDeny
		}
	}
	return ActivityAbstain
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
