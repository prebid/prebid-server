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

func NewActivityControl(cfg *config.AccountPrivacy) ActivityControl {
	ac := ActivityControl{}

	if cfg == nil || cfg.AllowActivities == nil {
		return ac
	}

	plans := make(map[Activity]ActivityPlan, 8)
	plans[ActivitySyncUser] = buildPlan(cfg.AllowActivities.SyncUser)
	plans[ActivityFetchBids] = buildPlan(cfg.AllowActivities.FetchBids)
	plans[ActivityEnrichUserFPD] = buildPlan(cfg.AllowActivities.EnrichUserFPD)
	plans[ActivityReportAnalytics] = buildPlan(cfg.AllowActivities.ReportAnalytics)
	plans[ActivityTransmitUserFPD] = buildPlan(cfg.AllowActivities.TransmitUserFPD)
	plans[ActivityTransmitPreciseGeo] = buildPlan(cfg.AllowActivities.TransmitPreciseGeo)
	plans[ActivityTransmitUniqueRequestIds] = buildPlan(cfg.AllowActivities.TransmitUniqueRequestIds)
	plans[ActivityTransmitTids] = buildPlan(cfg.AllowActivities.TransmitTids)
	ac.plans = plans

	return ac
}

func buildPlan(activity config.Activity) ActivityPlan {
	return ActivityPlan{
		rules:         cfgToRules(activity.Rules),
		defaultResult: cfgToDefaultResult(activity.Default),
	}
}

func cfgToRules(rules []config.ActivityRule) []Rule {
	var enfRules []Rule

	for _, r := range rules {
		var result ActivityResult
		if r.Allow {
			result = ActivityAllow
		} else {
			result = ActivityDeny
		}

		er := ConditionRule{
			result:        result,
			componentName: r.Condition.ComponentName,
			componentType: r.Condition.ComponentType,
		}
		enfRules = append(enfRules, er)
	}
	return enfRules
}

func cfgToDefaultResult(activityDefault *bool) bool {
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
