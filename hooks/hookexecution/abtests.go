package hookexecution

import (
	"math/rand"
	"slices"
	"sync"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/hooks/hookanalytics"
	"github.com/tidwall/gjson"
)

type (
	ABTests struct {
		config       *config.Configuration
		account      *config.Account
		accountID    string
		runMap       map[string]bool
		logMap       map[string]bool
		loggedMap    map[string]bool
		targetingMap map[string]string
		planLoaded   bool
		mu           sync.Mutex
	}
)

func NewABTests(cfg *config.Configuration) *ABTests {
	abTester := ABTests{
		config:       cfg,
		runMap:       make(map[string]bool),
		logMap:       make(map[string]bool),
		loggedMap:    make(map[string]bool),
		targetingMap: make(map[string]string),
	}

	return &abTester
}

func (t *ABTests) SetAccount(account *config.Account) {
	t.account = account
}

func (t *ABTests) SetAccountID(body []byte) {
	t.accountID = gjson.GetBytes(body, "site.publisher.id").String()
}

func (t *ABTests) Run(module string) bool {
	if !t.planLoaded {
		t.planHost()
		t.planAccount()
		t.planLoaded = true
	}

	val, ok := t.runMap[module]
	if !ok {
		return true
	}
	return val
}

func (t *ABTests) WriteOutcome(outcome *StageOutcome) {
	for module, logged := range t.logMap {
		if !logged {
			continue
		}

		if t.getLogged(module) {
			continue
		}

		var a hookanalytics.Activity
		resultStatus := hookanalytics.ResultStatusSkip
		if t.runMap[module] {
			resultStatus = hookanalytics.ResultStatusRun
		}
		a.Name = "core-module-abtests"
		a.Status = hookanalytics.ActivityStatusSuccess
		a.Results = append(a.Results, hookanalytics.Result{
			Status: resultStatus,
			Values: map[string]interface{}{
				"module": module,
			},
		})

		for groupKey, group := range outcome.Groups {
			for invocationResultKey, invocationResult := range group.InvocationResults {
				if invocationResult.HookID.ModuleCode == module {
					outcome.Groups[groupKey].InvocationResults[invocationResultKey].AnalyticsTags.Activities =
						append(outcome.Groups[groupKey].InvocationResults[invocationResultKey].AnalyticsTags.Activities, a)
					t.setLogged(module, true)
				}
			}
		}

		if t.runMap[module] || t.getLogged(module) {
			continue
		}

		var group GroupOutcome
		var invocationResult HookOutcome
		invocationResult.AnalyticsTags.Activities = append(invocationResult.AnalyticsTags.Activities, a)
		invocationResult.Status = StatusSuccess
		invocationResult.HookID.ModuleCode = module
		group.InvocationResults = append(group.InvocationResults, invocationResult)
		outcome.Groups = append(outcome.Groups, group)
		t.setLogged(module, true)
	}
}

func (t *ABTests) planHost() {
	for _, abtest := range t.config.Hooks.HostExecutionPlan.ABTests {
		module := abtest.ModuleCode
		if module == "" {
			glog.Warning("hooks.execution_plan.[]abtests.module_code is required")
			continue
		}

		if abtest.Enabled == nil || !*abtest.Enabled {
			continue
		}

		lat := true
		if abtest.LogAnalyticsTag != nil {
			lat = *abtest.LogAnalyticsTag
		}
		t.logMap[module] = lat

		if lat {
			t.setLogged(module, false)
		}

		if abtest.AdServerTargeting != "" {
			t.targetingMap[module] = abtest.AdServerTargeting
		}

		if !t.containsAccount(abtest.Accounts) {
			t.runMap[abtest.ModuleCode] = false
			continue
		}

		pa := uint16(100)
		if abtest.PercentActive != nil && *abtest.PercentActive < uint16(100) {
			pa = *abtest.PercentActive
		}
		t.runMap[module] = uint16(rand.Intn(100)) < pa
	}
}

func (t *ABTests) planAccount() {
	cfg := t.config.Hooks.DefaultAccountExecutionPlan.ABTests
	if t.account != nil {
		cfg = t.account.Hooks.ExecutionPlan.ABTests
	}

	for _, abtest := range cfg {
		module := abtest.ModuleCode
		if module == "" {
			glog.Warning("hooks.execution_plan.[]abtests.module_code is required")
			continue
		}

		if abtest.Enabled == nil || !*abtest.Enabled {
			delete(t.runMap, abtest.ModuleCode)
			delete(t.targetingMap, abtest.ModuleCode)
			continue
		}

		lat := true
		if abtest.LogAnalyticsTag != nil {
			lat = *abtest.LogAnalyticsTag
		}
		t.logMap[module] = lat

		if lat {
			t.setLogged(module, false)
		}

		if abtest.AdServerTargeting != "" {
			t.targetingMap[module] = abtest.AdServerTargeting
		}

		pa := uint16(100)
		if abtest.PercentActive != nil && *abtest.PercentActive < uint16(100) {
			pa = *abtest.PercentActive
		}
		t.runMap[module] = uint16(rand.Intn(100)) < pa
	}
}

func (t *ABTests) containsAccount(accounts []string) bool {
	if len(accounts) == 0 {
		return true
	}

	accountID := t.accountID
	if t.account != nil {
		accountID = t.account.ID
	}

	return slices.Contains(accounts, accountID)
}

func (t *ABTests) getLogged(module string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.loggedMap[module]
}

func (t *ABTests) setLogged(module string, val bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.loggedMap[module] = val
}

func (t *ABTests) GetTargetingKeywords() map[string]string {
	if !t.planLoaded {
		t.planHost()
		t.planAccount()
		t.planLoaded = true
	}

	result := make(map[string]string)
	for module, keyword := range t.targetingMap {
		if keyword != "" {
			value := "0"
			if t.runMap[module] {
				value = "1"
			}
			result[keyword] = value
		}
	}
	return result
}
