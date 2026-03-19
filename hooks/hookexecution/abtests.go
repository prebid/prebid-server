package hookexecution

import (
	"math/rand"
	"slices"
	"sync"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/hooks/hookanalytics"
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
		initOnce     sync.Once
		mu           sync.RWMutex
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
	if id := gjson.GetBytes(body, "site.publisher.id").String(); id != "" {
		t.accountID = id
		return
	}
	t.accountID = gjson.GetBytes(body, "app.publisher.id").String()
}

func (t *ABTests) init() {
	t.initOnce.Do(func() {
		t.planHost()
		t.planAccount()
	})
}

func (t *ABTests) Run(module string) bool {
	t.init()
	t.mu.RLock()
	defer t.mu.RUnlock()
	val, ok := t.runMap[module]
	if !ok {
		return true
	}
	return val
}

func (t *ABTests) WriteOutcome(outcome *StageOutcome) {
	t.init()
	t.mu.RLock()
	logMap := t.logMap
	runMap := t.runMap
	t.mu.RUnlock()

	for module, logged := range logMap {
		if !logged {
			continue
		}

		if t.checkAndSetLogged(module) {
			continue
		}

		var a hookanalytics.Activity
		resultStatus := hookanalytics.ResultStatusSkip
		if runMap[module] {
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

		placed := false
		for groupKey, group := range outcome.Groups {
			for invocationResultKey, invocationResult := range group.InvocationResults {
				if invocationResult.HookID.ModuleCode == module {
					outcome.Groups[groupKey].InvocationResults[invocationResultKey].AnalyticsTags.Activities =
						append(outcome.Groups[groupKey].InvocationResults[invocationResultKey].AnalyticsTags.Activities, a)
					placed = true
				}
			}
		}

		if runMap[module] || placed {
			continue
		}

		var group GroupOutcome
		var invocationResult HookOutcome
		invocationResult.AnalyticsTags.Activities = append(invocationResult.AnalyticsTags.Activities, a)
		invocationResult.Status = StatusSuccess
		invocationResult.HookID.ModuleCode = module
		group.InvocationResults = append(group.InvocationResults, invocationResult)
		outcome.Groups = append(outcome.Groups, group)
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
			t.loggedMap[module] = false
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

// Account-level AB test entries are already scoped to the specific account
// by the configuration hierarchy, thus no account filtering is needed.
// The "accounts" field is only meaningful in the host-level plan (planHost),
// where it scopes a global entry to a subset of accounts.
// This matches the PBS-Java reference implementation.
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
			delete(t.logMap, abtest.ModuleCode)
			delete(t.loggedMap, abtest.ModuleCode)
			continue
		}

		lat := true
		if abtest.LogAnalyticsTag != nil {
			lat = *abtest.LogAnalyticsTag
		}
		t.logMap[module] = lat

		if lat {
			t.loggedMap[module] = false
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

// checkAndSetLogged atomically checks whether the module outcome has already been
// logged and, if not, marks it as logged. Returns true if already logged (skip),
// false if not yet logged (caller should proceed to build analytics entry).
func (t *ABTests) checkAndSetLogged(module string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.loggedMap[module] {
		return true
	}
	t.loggedMap[module] = true
	return false
}

func (t *ABTests) GetTargetingKeywords() map[string]string {
	t.init()
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make(map[string]string)
	for module, keyword := range t.targetingMap {
		if keyword != "" {
			value := string(hookanalytics.ResultStatusSkip)
			if t.runMap[module] {
				value = string(hookanalytics.ResultStatusRun)
			}
			result[keyword] = value
		}
	}
	return result
}
