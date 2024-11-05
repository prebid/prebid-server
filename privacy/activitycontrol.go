package privacy

import (
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type ActivityResult int

const (
	ActivityAbstain ActivityResult = iota
	ActivityAllow
	ActivityDeny
)

const defaultActivityResult = true

func NewRequestFromPolicies(p Policies) ActivityRequest {
	return ActivityRequest{policies: &p}
}

func NewRequestFromBidRequest(r openrtb_ext.RequestWrapper) ActivityRequest {
	return ActivityRequest{bidRequest: &r}
}

type ActivityRequest struct {
	policies   *Policies
	bidRequest *openrtb_ext.RequestWrapper
}

func (r ActivityRequest) IsPolicies() bool {
	return r.policies != nil
}

func (r ActivityRequest) IsBidRequest() bool {
	return r.bidRequest != nil
}

type ActivityControl struct {
	plans      map[Activity]ActivityPlan
	IPv6Config config.IPv6
	IPv4Config config.IPv4
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
	plans[ActivityTransmitUniqueRequestIDs] = buildPlan(cfg.AllowActivities.TransmitUniqueRequestIds)
	plans[ActivityTransmitTIDs] = buildPlan(cfg.AllowActivities.TransmitTids)
	ac.plans = plans

	ac.IPv4Config = cfg.IPv4Config
	ac.IPv6Config = cfg.IPv6Config

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
		result := ActivityDeny
		if r.Allow {
			result = ActivityAllow
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

func (e ActivityControl) Allow(activity Activity, target Component, request ActivityRequest) bool {
	plan, planDefined := e.plans[activity]

	if !planDefined {
		return defaultActivityResult
	}

	return plan.Evaluate(target, request)
}

type ActivityPlan struct {
	defaultResult bool
	rules         []Rule
}

func (p ActivityPlan) Evaluate(target Component, request ActivityRequest) bool {
	for _, rule := range p.rules {
		result := rule.Evaluate(target, request)
		if result == ActivityDeny || result == ActivityAllow {
			return result == ActivityAllow
		}
	}
	return p.defaultResult
}
