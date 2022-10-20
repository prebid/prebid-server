package hep

import (
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks/stages"
)

const (
	stageEntrypoint          = "entrypoint"
	stageRawauction          = "rawauction"
	stageProcauction         = "procauction"
	stageBidrequest          = "bidrequest"
	stageRawbidresponse      = "rawbidresponse"
	stageProcbidresponse     = "procbidresponse"
	stageAllprocbidresponses = "allprocbidresponses"
	stageAuctionresponse     = "auctionresponse"
)

type HookExecutionPlanBuilder interface {
	PlanForEntrypointStage(endpoint string) Plan[stages.EntrypointHook]
	PlanForRawAuctionStage(endpoint string, account *config.Account) Plan[stages.RawAuctionHook]
	PlanForProcessedAuctionStage(endpoint string, account *config.Account) Plan[stages.ProcessedAuctionHook]
	PlanForBidRequestStage(endpoint string, account *config.Account) Plan[stages.BidRequestHook]
	PlanForRawBidResponseStage(endpoint string, account *config.Account) Plan[stages.RawBidResponseHook]
	PlanForProcessedBidResponseStage(endpoint string, account *config.Account) Plan[stages.ProcessedBidResponseHook]
	PlanForAllProcBidResponsesStage(endpoint string, account *config.Account) Plan[stages.AllProcBidResponsesHook]
	PlanForAuctionResponseStage(endpoint string, account *config.Account) Plan[stages.AuctionResponseHook]
	DebugModeEnabled() bool
}

func NewHookExecutionPlanBuilder(hooks config.Hooks, repo HookRepository) HookExecutionPlanBuilder {
	return ExecutionPlan{
		hooks:     hooks,
		repo:      repo,
		debugMode: false, //todo: implement
	}
}

type ExecutionPlan struct {
	hooks     config.Hooks
	repo      HookRepository
	debugMode bool
}

func (p ExecutionPlan) PlanForEntrypointStage(endpoint string) Plan[stages.EntrypointHook] {
	return getMergedPlan(
		p.hooks,
		nil,
		endpoint,
		stageEntrypoint,
		p.repo.GetEntrypointHook,
	)
}

func (p ExecutionPlan) PlanForRawAuctionStage(endpoint string, account *config.Account) Plan[stages.RawAuctionHook] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		stageRawauction,
		p.repo.GetRawAuctionHook,
	)
}

func (p ExecutionPlan) PlanForProcessedAuctionStage(endpoint string, account *config.Account) Plan[stages.ProcessedAuctionHook] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		stageProcauction,
		p.repo.GetProcessedAuctionHook,
	)
}

func (p ExecutionPlan) PlanForBidRequestStage(endpoint string, account *config.Account) Plan[stages.BidRequestHook] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		stageBidrequest,
		p.repo.GetBidRequestHook,
	)
}

func (p ExecutionPlan) PlanForRawBidResponseStage(endpoint string, account *config.Account) Plan[stages.RawBidResponseHook] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		stageRawbidresponse,
		p.repo.GetRawBidResponseHook,
	)
}

func (p ExecutionPlan) PlanForProcessedBidResponseStage(endpoint string, account *config.Account) Plan[stages.ProcessedBidResponseHook] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		stageProcbidresponse,
		p.repo.GetProcessedBidResponseHook,
	)
}

func (p ExecutionPlan) PlanForAllProcBidResponsesStage(endpoint string, account *config.Account) Plan[stages.AllProcBidResponsesHook] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		stageAllprocbidresponses,
		p.repo.GetAllProcBidResponsesHook,
	)
}

func (p ExecutionPlan) PlanForAuctionResponseStage(endpoint string, account *config.Account) Plan[stages.AuctionResponseHook] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		stageAuctionresponse,
		p.repo.GetAuctionResponseHook,
	)
}

func (p ExecutionPlan) DebugModeEnabled() bool {
	return p.debugMode
}

type Plan[T any] []Group[T]

type Group[T any] struct {
	Timeout time.Duration
	Hooks   []HookWrapper[T]
}

type HookWrapper[T any] struct {
	Module string
	Code   string
	Hook   T
}

type hookFn[T any] func(moduleName, hookCode string) (T, bool)

func getMergedPlan[T any](
	cfg config.Hooks,
	account *config.Account,
	endpoint, stage string,
	getHookFn hookFn[T],
) Plan[T] {
	accountPlan := cfg.DefaultAccountExecutionPlan
	if account != nil && account.Hooks.Endpoints != nil {
		accountPlan = account.Hooks
	}

	plan := getPlan(getHookFn, cfg.HostExecutionPlan, endpoint, stage)
	plan = append(plan, getPlan(getHookFn, accountPlan, endpoint, stage)...)

	return plan
}

func getPlan[T any](getHookFn hookFn[T], hep config.HookExecutionPlan, endpoint, stage string) Plan[T] {
	plan := make(Plan[T], 0, len(hep.Endpoints[endpoint].Stages[stage].Groups))
	for _, groupCfg := range hep.Endpoints[endpoint].Stages[stage].Groups {
		group := Group[T]{Timeout: time.Duration(groupCfg.Timeout) * time.Millisecond}
		for _, hookCfg := range groupCfg.HookSequence {
			if h, ok := getHookFn(hookCfg.Module, hookCfg.Hook); ok {
				group.Hooks = append(group.Hooks, HookWrapper[T]{Module: hookCfg.Module, Code: hookCfg.Hook, Hook: h})
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
