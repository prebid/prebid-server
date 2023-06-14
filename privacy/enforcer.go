package privacy

import (
	"fmt"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"strings"
)

type EnforceResult int // or maybe ActivityResult?

const (
	EnforceAbstain EnforceResult = iota
	EnforceAllow
	EnforceDeny
)

type ActivityControl struct {
	plans map[Activity]EnforcementPlan
}

func NewActivityControl(hostConf config.AccountPrivacy, accConf config.AccountPrivacy) (ActivityControl, error) {
	//!!how to merge host config with acc configs?
	ac := ActivityControl{plans: nil}
	var err error

	plans := make(map[Activity]EnforcementPlan)

	plans[ActivitySyncUser], err = buildEnforcementPlan(hostConf.AllowActivities.SyncUser)
	if err != nil {
		return ac, err
	}
	plans[ActivityFetchBids], err = buildEnforcementPlan(hostConf.AllowActivities.FetchBids)
	if err != nil {
		return ac, err
	}
	plans[ActivityEnrichUserFPD], err = buildEnforcementPlan(hostConf.AllowActivities.EnrichUserFPD)
	if err != nil {
		return ac, err
	}
	plans[ActivityReportAnalytics], err = buildEnforcementPlan(hostConf.AllowActivities.ReportAnalytics)
	if err != nil {
		return ac, err
	}
	plans[ActivityTransmitUserFPD], err = buildEnforcementPlan(hostConf.AllowActivities.TransmitUserFPD)
	if err != nil {
		return ac, err
	}
	plans[ActivityTransmitPreciseGeo], err = buildEnforcementPlan(hostConf.AllowActivities.TransmitPreciseGeo)
	if err != nil {
		return ac, err
	}
	plans[ActivityTransmitUniqueRequestIds], err = buildEnforcementPlan(hostConf.AllowActivities.TransmitUniqueRequestIds)
	if err != nil {
		return ac, err
	}
	plans[ActivityTransmitTIds], err = buildEnforcementPlan(hostConf.AllowActivities.TransmitTIds)
	if err != nil {
		return ac, err
	}

	ac.plans = plans

	return ac, nil
}

func buildEnforcementPlan(activity config.Activity) (EnforcementPlan, error) {
	ef := EnforcementPlan{}
	rules, err := activityRulesToEnforcementRules(activity.Rules)
	if err != nil {
		return ef, err
	}
	ef.defaultResult = activityDefaultToDefaultResult(activity.Default)
	ef.rules = rules
	return ef, nil
}

func activityRulesToEnforcementRules(rules []config.ActivityRule) ([]EnforcementRule, error) {
	enfRules := make([]EnforcementRule, 0)
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

func activityDefaultToDefaultResult(activityDefault *bool) EnforceResult {
	if activityDefault == nil {
		return EnforceAbstain
	} else if *activityDefault {
		return EnforceDeny
	}
	return EnforceAllow
}

func (e ActivityControl) Allow(activity Activity, request openrtb_ext.RequestWrapper, target ScopedName) EnforceResult {
	plan, planDefined := e.plans[activity]

	if !planDefined {
		return EnforceAbstain
	}

	return plan.Allow(request, target)
}

// allow this to be created from acitivty config, which veronika will get from the account config root object
// maybe call this ActivityPlan?
type EnforcementPlan struct {
	defaultResult EnforceResult
	rules         []EnforcementRule
}

func (p EnforcementPlan) Allow(request openrtb_ext.RequestWrapper, target ScopedName) EnforceResult {
	for _, rule := range p.rules {
		result := rule.Allow(request, target) // exit on first non-abstain response
		if result == EnforceAllow || result == EnforceDeny {
			return result
		}
	}
	return p.defaultResult
}

// maybe call this ActivityRule?
type EnforcementRule interface {
	Allow(request openrtb_ext.RequestWrapper, target ScopedName) EnforceResult
}

type ComponentEnforcementRule struct {
	componentName []ScopedName
	componentType []string
	// include gppSectionId from 3.5
	// include geo from 3.5
	allowed bool // behavior if rule matches. can be either true=allow or false=deny. result is abstain if the rule doesn't match
}

func (r ComponentEnforcementRule) Allow(request openrtb_ext.RequestWrapper, target ScopedName) EnforceResult {
	// all string comparisons in this section are case sensitive
	// doc: https://docs.google.com/document/d/1dRxFUFmhh2jGanzGZvfkK_6jtHPpHXWD7Qsi6KEugeE/edit
	// the doc details the boolean operations.
	//  - "or" within each field (componentName, componentType
	//  - "and" between the rules present. empty fields are ignored (refer to doc for details)

	// componentName
	// check for matching scoped name. a wildcard is allowed for the name in which any target with the same scope is matched

	// componentType
	// can either act as a scope wildcard or meta targeting. can be scope "bidder", "analytics", maybe others.
	// may also be "rtd" meta. you need to pass that through somehow, perhaps as targetMeta? targetMeta can be a slice. should be small enough that search speed isn't a concern.

	// gppSectionId
	// check if id is present in the gppsid slice. no parsing of gpp should happen here.

	// geo
	// simple filter on the req.user section

	scopeFound := false
	for _, scope := range r.componentName {
		if strings.EqualFold(scope.Scope, target.Scope) && strings.EqualFold(scope.Name, target.Name) {
			scopeFound = true
			break
		}
	}

	typeFound := false
	for _, componentType := range r.componentType {
		if strings.EqualFold(componentType, target.Scope) {
			typeFound = true
			break
		}
	}

	matchFound := scopeFound || typeFound

	if matchFound {
		if r.allowed {
			return EnforceDeny
		} else {
			return EnforceAllow
		}
	}
	return EnforceAbstain
}

// the default scope should be hardcoded as bidder
// ex: "bidder.appnexus", "bidder.*", "appnexus", "analytics.pubmatic"
// TODO: add parsing helpers
type ScopedName struct {
	Scope string
	Name  string
}

const (
	ScopeTypeBidder    = "bidder"
	ScopeTypeAnalytics = "analytics"
	ScopeTypeRTD       = "rtd" // real time data
	ScopeTypeUserId    = "userid"
	ScopeTypeGeneral   = "general"
)

func NewScopedName(condition string) (ScopedName, error) {

	var scope, name string
	split := strings.Split(condition, ".")
	if len(split) == 2 {
		s := strings.ToLower(split[0])
		if s == ScopeTypeBidder || s == ScopeTypeAnalytics || s == ScopeTypeUserId {
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

// ex: "USA.VA", "USA". see all comments in https://github.com/prebid/prebid-server/issues/2622
// TODO: add parsing helpers
type Geo struct {
	Country string
	Region  string
}
