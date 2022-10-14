package plans

import (
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks/hep"
	"github.com/prebid/prebid-server/hooks/stages"
)

func NewHookExecutionPlanBuilder(hooks config.Hooks, repo hep.HookRepository) hep.HookExecutionPlanBuilder {
	return PlanBuilder{
		hooks: hooks,
		repo:  repo,
	}
}

type PlanBuilder struct {
	hooks config.Hooks
	repo  hep.HookRepository
}

func (p PlanBuilder) PlanForEntrypointStage(endpoint string) hep.Plan[stages.EntrypointHook] {
	return getMergedPlan(
		p.hooks,
		nil,
		endpoint,
		hep.StageEntrypoint,
		p.repo.GetEntrypointHook,
	)
}

func (p PlanBuilder) PlanForRawAuctionStage(endpoint string, account *config.Account) hep.Plan[stages.RawAuctionHook] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		hep.StageRawauction,
		p.repo.GetRawAuctionHook,
	)
}

func (p PlanBuilder) PlanForProcessedAuctionStage(endpoint string, account *config.Account) hep.Plan[stages.ProcessedAuctionHook] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		hep.StageProcauction,
		p.repo.GetProcessedAuctionHook,
	)
}

func (p PlanBuilder) PlanForBidRequestStage(endpoint string, account *config.Account) hep.Plan[stages.BidRequestHook] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		hep.StageBidrequest,
		p.repo.GetBidRequestHook,
	)
}

func (p PlanBuilder) PlanForRawBidResponseStage(endpoint string, account *config.Account) hep.Plan[stages.RawBidResponseHook] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		hep.StageRawbidresponse,
		p.repo.GetRawBidResponseHook,
	)
}

func (p PlanBuilder) PlanForAllProcBidResponsesStage(endpoint string, account *config.Account) hep.Plan[stages.AllProcBidResponsesHook] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		hep.StageAllprocbidresponses,
		p.repo.GetAllProcBidResponsesHook,
	)
}

func (p PlanBuilder) PlanForAuctionResponseStage(endpoint string, account *config.Account) hep.Plan[stages.AuctionResponseHook] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		hep.StageAuctionresponse,
		p.repo.GetAuctionResponseHook,
	)
}

type hookFn[T any] func(moduleName, hookCode string) (T, bool)

func getMergedPlan[T any](
	cfg config.Hooks,
	account *config.Account,
	endpoint, stage string,
	getHookFn hookFn[T],
) hep.Plan[T] {
	accountPlan := cfg.DefaultAccountExecutionPlan
	if account != nil && account.Hooks.Endpoints != nil {
		accountPlan = account.Hooks
	}

	plan := getPlan(getHookFn, cfg.HostExecutionPlan, endpoint, stage)
	plan = append(plan, getPlan(getHookFn, accountPlan, endpoint, stage)...)

	return plan
}

func getPlan[T any](getHookFn hookFn[T], cfg config.HookExecutionPlan, endpoint, stage string) hep.Plan[T] {
	plan := make(hep.Plan[T], 0, len(cfg.Endpoints[endpoint].Stages[stage].Groups))
	for _, groupCfg := range cfg.Endpoints[endpoint].Stages[stage].Groups {
		group := hep.Group[T]{Timeout: time.Duration(groupCfg.Timeout) * time.Millisecond}
		for _, hookCfg := range groupCfg.HookSequence {
			if h, ok := getHookFn(hookCfg.Module, hookCfg.Hook); ok {
				group.Hooks = append(group.Hooks, hep.HookWrapper[T]{Module: hookCfg.Module, Code: hookCfg.Hook, Hook: h})
			} else if !ok {
				glog.Warningf("Not found hook while building hook execution plan: %s %s", hookCfg.Module, hookCfg.Hook)
			}
		}

		if len(group.Hooks) > 0 {
			plan = append(plan, group)
		}
	}

	return plan
}
