package privacy

import (
	"github.com/prebid/prebid-server/config"
)

type ActivityResult int

const (
	ActivityAbstain ActivityResult = iota
	ActivityAllow
	ActivityDeny
)

const defaultActivityResult = true

type ActivityControl struct {
	plans map[Activity]ActivityPlan
}

func NewActivityControl(privacyConf *config.AccountPrivacy) (ActivityControl, error) {
	ac := ActivityControl{}
	var err error

	if privacyConf == nil || privacyConf.AllowActivities == nil {
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

func activityRulesToEnforcementRules(rules []config.ActivityRule) ([]Rule, error) {
	var enfRules []Rule

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

func conditionToRuleComponentNames(conditions []string) ([]Component, error) {
	sn := make([]Component, 0)
	for _, condition := range conditions {
		scope, err := ParseComponent(condition)
		if err != nil {
			return nil, err
		}
		sn = append(sn, scope)
	}
	return sn, nil
}

func activityDefaultToDefaultResult(activityDefault *bool) bool {
	if activityDefault == nil {
		return defaultActivityResult
	}
	return *activityDefault
}

func (e ActivityControl) Allow(activity Activity, target Component) bool {
	plan, planDefined := e.plans[activity]

	if !planDefined {
		return defaultActivityResult
	}

	return plan.Evaluate(target)
}

type ActivityPlan struct {
	defaultResult bool
	rules         []Rule
}

func (p ActivityPlan) Evaluate(target Component) bool {
	for _, rule := range p.rules {
		result := rule.Evaluate(target)
		if result == ActivityDeny || result == ActivityAllow {
			return result == ActivityAllow
		}
	}
	return p.defaultResult
}
