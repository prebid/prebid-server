package validation_test

// Tests that drive the module through the real Prebid Server plumbing: the
// module builder, the hook repository and execution plan, the hook executor
// with its change set, and the host configuration loader. They guard the
// README as much as the code.

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/hooks"
	"github.com/prebid/prebid-server/v4/hooks/hookexecution"
	metricsConfig "github.com/prebid/prebid-server/v4/metrics/config"
	"github.com/prebid/prebid-server/v4/modules"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const moduleCode = "vastlint.validation"

const planJSON = `{"endpoints":{"/openrtb2/auction":{"stages":{"raw_bidder_response":{"groups":[{"timeout":200,"hook_sequence":[{"module_code":"vastlint.validation","hook_impl_code":"vastlint-raw-bidder-response"}]}]}}}}}`

func build(t *testing.T, moduleCfg map[string]interface{}) (hooks.HookRepository, map[string][]string, *modules.ShutdownModules) {
	t.Helper()
	repo, stages, shutdown, err := modules.NewBuilder().Build(config.Modules{"vastlint": {"validation": moduleCfg}}, moduledeps.ModuleDeps{})
	require.NoError(t, err)
	return repo, stages, shutdown
}

func executor(t *testing.T, repo hooks.HookRepository) hookexecution.HookStageExecutor {
	t.Helper()
	var plan config.HookExecutionPlan
	require.NoError(t, json.Unmarshal([]byte(planJSON), &plan))
	builder := hooks.NewExecutionPlanBuilder(config.Hooks{Enabled: true, HostExecutionPlan: plan}, repo)
	return hookexecution.NewHookExecutor(builder, hookexecution.EndpointAuction, &metricsConfig.NilMetricsEngine{})
}

func video(id, adm string) *adapters.TypedBid {
	return &adapters.TypedBid{Bid: &openrtb2.Bid{ID: id, ImpID: "imp-1", AdM: adm, MType: openrtb2.MarkupVideo}, BidType: openrtb_ext.BidTypeVideo}
}

// fixture reads a creative from testdata: missing_impression.xml fails an
// error-level rule, valid_with_warnings.xml (a fixture from the engine's golden set)
// has no error-level finding.
func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	return string(b)
}

func TestFrameworkBuildsAndRegistersTheModule(t *testing.T) {
	repo, stages, shutdown := build(t, map[string]interface{}{"enabled": true, "cache_size": 1 << 20})
	_, ok := repo.GetRawBidderResponseHook(moduleCode)
	assert.True(t, ok, "hook registered under the module code")
	assert.Contains(t, stages["vastlint_validation"], hooks.StageRawBidderResponse.String(), "the builder keys stages by vendor_module")
	require.NotNil(t, shutdown)
	shutdown.Shutdown()

	repo, _, _ = build(t, map[string]interface{}{"enabled": false})
	_, ok = repo.GetRawBidderResponseHook(moduleCode)
	assert.False(t, ok, "a disabled module is never built")

	_, _, _, err := modules.NewBuilder().Build(config.Modules{"vastlint": {"validation": map[string]interface{}{"enabled": true, "reject_on": []string{"no-such-rule"}}}}, moduledeps.ModuleDeps{})
	assert.Error(t, err, "a bad host config fails startup")
}

func TestExecutorAppliesAnnotationsAndReportsTags(t *testing.T) {
	repo, _, shutdown := build(t, map[string]interface{}{"enabled": true})
	defer shutdown.Shutdown()
	exec := executor(t, repo)
	resp := adapters.BidderResponse{Bids: []*adapters.TypedBid{video("broken", fixture(t, "missing_impression.xml")), video("clean", fixture(t, "valid_with_warnings.xml"))}}
	reject := exec.ExecuteRawBidderResponseStage(&resp, "adprime")
	require.Nil(t, reject)
	require.Len(t, resp.Bids, 2, "monitor mode removes nothing")
	for _, tb := range resp.Bids {
		assert.Contains(t, string(tb.Bid.Ext), `"vastlint":{`, "the executor applied the annotation to the caller's response")
	}
	assert.Contains(t, string(resp.Bids[0].Bid.Ext), `"valid":false`)
	assert.Contains(t, string(resp.Bids[1].Bid.Ext), `"valid":true`)

	outcomes := exec.GetOutcomes()
	require.Len(t, outcomes, 1)
	require.Len(t, outcomes[0].Groups, 1)
	require.Len(t, outcomes[0].Groups[0].InvocationResults, 1)
	inv := outcomes[0].Groups[0].InvocationResults[0]
	assert.Equal(t, moduleCode, inv.HookID.ModuleCode)
	assert.Equal(t, hookexecution.StatusSuccess, inv.Status)
	assert.Equal(t, hookexecution.ActionUpdate, inv.Action)
	require.Len(t, inv.AnalyticsTags.Activities, 1)
	act := inv.AnalyticsTags.Activities[0]
	assert.Equal(t, "vast_validation", act.Name)
	require.Len(t, act.Results, 2)
	for _, r := range act.Results {
		assert.Equal(t, "success-modify", string(r.Status))
		assert.Equal(t, "checked", r.Values["action"])
		assert.Equal(t, "adprime", r.AppliedTo.Bidder)
		assert.Equal(t, []string{"imp-1"}, r.AppliedTo.ImpIds)
	}
}

func TestExecutorHonorsAccountLevelConfig(t *testing.T) {
	repo, _, shutdown := build(t, map[string]interface{}{"enabled": true, "cache_size": 1 << 20})
	defer shutdown.Shutdown()
	exec := executor(t, repo)
	exec.SetAccount(&config.Account{ID: "acct-1", Hooks: config.AccountHooks{Modules: config.AccountModules{
		"vastlint": {"validation": json.RawMessage(`{"mode":"reject","reject_on":["VAST-2.0-inline-impression"]}`)},
	}}})
	resp := adapters.BidderResponse{Bids: []*adapters.TypedBid{video("broken", fixture(t, "missing_impression.xml")), video("clean", fixture(t, "valid_with_warnings.xml"))}}
	require.Nil(t, exec.ExecuteRawBidderResponseStage(&resp, "adprime"))
	require.Len(t, resp.Bids, 1, "the account's reject override removed the broken bid")
	assert.Equal(t, "clean", resp.Bids[0].Bid.ID)
	inv := exec.GetOutcomes()[0].Groups[0].InvocationResults[0]
	actions := map[string]int{}
	for _, r := range inv.AnalyticsTags.Activities[0].Results {
		actions[r.Values["action"].(string)]++
	}
	assert.Equal(t, map[string]int{"dropped": 1, "checked": 1}, actions)

	// A second request for an account without overrides sees the host mode
	// (monitor) and the cached verdicts.
	exec2 := executor(t, repo)
	exec2.SetAccount(&config.Account{ID: "acct-2"})
	resp2 := adapters.BidderResponse{Bids: []*adapters.TypedBid{video("broken", fixture(t, "missing_impression.xml"))}}
	require.Nil(t, exec2.ExecuteRawBidderResponseStage(&resp2, "adprime"))
	require.Len(t, resp2.Bids, 1, "host config applies to an account without overrides")
	r := exec2.GetOutcomes()[0].Groups[0].InvocationResults[0].AnalyticsTags.Activities[0].Results[0]
	assert.Equal(t, "checked", r.Values["action"])
	assert.Equal(t, "hit", r.Values["cache"], "the verdict cache is shared across accounts and keyed by rules, not by mode")
}

func TestExecutorAsyncModeLeavesResponseUntouched(t *testing.T) {
	repo, _, shutdown := build(t, map[string]interface{}{"enabled": true, "mode": "async"})
	defer shutdown.Shutdown()
	exec := executor(t, repo)
	resp := adapters.BidderResponse{Bids: []*adapters.TypedBid{video("broken", fixture(t, "missing_impression.xml"))}}
	require.Nil(t, exec.ExecuteRawBidderResponseStage(&resp, "adprime"))
	assert.Empty(t, resp.Bids[0].Bid.Ext, "async mode never touches the bid")
	inv := exec.GetOutcomes()[0].Groups[0].InvocationResults[0]
	assert.Equal(t, hookexecution.ActionNone, inv.Action)
	assert.Equal(t, "queued", inv.AnalyticsTags.Activities[0].Results[0].Values["action"])
}

// TestREADMEConfigurationLoads feeds the README's YAML and JSON blocks through
// Prebid Server's own configuration loader, module builder and plan builder,
// so the documented examples can never drift from what the server accepts
// and actually enable.
func TestREADMEConfigurationLoads(t *testing.T) {
	readme, err := os.ReadFile("README.md")
	require.NoError(t, err)
	for _, block := range []struct{ lang, fence string }{{"yaml", "```yaml\n(.*?)```"}, {"json", "```json\n(.*?)```"}} {
		m := regexp.MustCompile("(?s)" + block.fence).FindStringSubmatch(string(readme))
		require.NotNil(t, m, "README has a %s block", block.lang)
		v := viper.New()
		config.SetupViper(v, "", config.BidderInfos{})
		v.SetConfigType(block.lang)
		require.NoError(t, v.ReadConfig(strings.NewReader(m[1])), block.lang)
		v.Set("gdpr.default_value", "0") // unrelated to the module; the loader requires a host to choose it
		cfg, err := config.New(v, config.BidderInfos{}, openrtb_ext.NormalizeBidderName)
		require.NoError(t, err, block.lang)
		require.True(t, cfg.Hooks.Enabled, block.lang)

		repo, stages, shutdown, err := modules.NewBuilder().Build(cfg.Hooks.Modules, moduledeps.ModuleDeps{})
		require.NoError(t, err, "%s: the documented module config builds", block.lang)
		require.NotNil(t, shutdown)
		defer shutdown.Shutdown()
		assert.Contains(t, stages["vastlint_validation"], hooks.StageRawBidderResponse.String(), "%s: the documented config enables the module", block.lang)
		hook, ok := repo.GetRawBidderResponseHook(moduleCode)
		require.True(t, ok, block.lang)
		if block.lang == "yaml" {
			// The YAML example sets a rule override; a YAML loader lowercases keys and
			// splits them on dots, which the module must survive.
			mod := hook.(interface{ Rules() map[string]string })
			assert.Equal(t, map[string]string{"VAST-2.0-adsystem-no-version": "off"}, mod.Rules(), "the rule override reaches the module with its canonical id")
		}
		plan := hooks.NewExecutionPlanBuilder(cfg.Hooks, repo).PlanForRawBidderResponseStage(hookexecution.EndpointAuction, nil)
		require.Len(t, plan, 1, "%s: one group in the documented plan", block.lang)
		require.Len(t, plan[0].Hooks, 1)
		assert.Equal(t, moduleCode, plan[0].Hooks[0].Module, block.lang)
		assert.Equal(t, "vastlint-raw-bidder-response", plan[0].Hooks[0].Code, block.lang)
	}
}
