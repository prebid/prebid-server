package hooks

import (
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks/hookstage"
)

const (
	StageEntrypoint               = "entrypoint"
	StageRawAuction               = "rawauction"
	StageProcessedAuction         = "procauction"
	StageBidRequest               = "bidrequest"
	StageRawBidResponse           = "rawbidresponse"
	StageAllProcessedBidResponses = "allprocbidresponses"
	StageAuctionResponse          = "auctionresponse"
)

type ExecutionPlanBuilder interface {
	PlanForEntrypointStage(endpoint string) Plan[hookstage.Entrypoint]
	PlanForRawAuctionStage(endpoint string, account *config.Account) Plan[hookstage.RawAuction]
	PlanForProcessedAuctionStage(endpoint string, account *config.Account) Plan[hookstage.ProcessedAuction]
	PlanForBidRequestStage(endpoint string, account *config.Account) Plan[hookstage.BidRequest]
	PlanForRawBidResponseStage(endpoint string, account *config.Account) Plan[hookstage.RawBidResponse]
	PlanForAllProcessedBidResponsesStage(endpoint string, account *config.Account) Plan[hookstage.AllProcessedBidResponses]
	PlanForAuctionResponseStage(endpoint string, account *config.Account) Plan[hookstage.AuctionResponse]
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

func NewExecutionPlanBuilder(hooks config.Hooks, repo HookRepository) ExecutionPlanBuilder {
	if hooks.Enabled {
		return PlanBuilder{
			hooks: hooks,
			repo:  repo,
		}
	}
	return EmptyPlanBuilder{}
}

type PlanBuilder struct {
	hooks config.Hooks
	repo  HookRepository
}

func (p PlanBuilder) PlanForEntrypointStage(endpoint string) Plan[hookstage.Entrypoint] {
	return getMergedPlan(
		p.hooks,
		nil,
		endpoint,
		StageEntrypoint,
		p.repo.GetEntrypointHook,
	)
}

func (p PlanBuilder) PlanForRawAuctionStage(endpoint string, account *config.Account) Plan[hookstage.RawAuction] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		StageRawAuction,
		p.repo.GetRawAuctionHook,
	)
}

func (p PlanBuilder) PlanForProcessedAuctionStage(endpoint string, account *config.Account) Plan[hookstage.ProcessedAuction] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		StageProcessedAuction,
		p.repo.GetProcessedAuctionHook,
	)
}

func (p PlanBuilder) PlanForBidRequestStage(endpoint string, account *config.Account) Plan[hookstage.BidRequest] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		StageBidRequest,
		p.repo.GetBidRequestHook,
	)
}

func (p PlanBuilder) PlanForRawBidResponseStage(endpoint string, account *config.Account) Plan[hookstage.RawBidResponse] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		StageRawBidResponse,
		p.repo.GetRawBidResponseHook,
	)
}

func (p PlanBuilder) PlanForAllProcessedBidResponsesStage(endpoint string, account *config.Account) Plan[hookstage.AllProcessedBidResponses] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		StageAllProcessedBidResponses,
		p.repo.GetAllProcessedBidResponsesHook,
	)
}

func (p PlanBuilder) PlanForAuctionResponseStage(endpoint string, account *config.Account) Plan[hookstage.AuctionResponse] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		StageAuctionResponse,
		p.repo.GetAuctionResponseHook,
	)
}

type hookFn[T any] func(moduleName string) (T, bool)

func getMergedPlan[T any](
	cfg config.Hooks,
	account *config.Account,
	endpoint, stage string,
	getHookFn hookFn[T],
) Plan[T] {
	accountPlan := cfg.AccountExecutionPlan
	if account != nil && account.Hooks.ExecutionPlan.Endpoints != nil {
		accountPlan = account.Hooks.ExecutionPlan
	}

	plan := getPlan(getHookFn, cfg.HostExecutionPlan, endpoint, stage)
	plan = append(plan, getPlan(getHookFn, accountPlan, endpoint, stage)...)

	return plan
}

func getPlan[T any](getHookFn hookFn[T], cfg config.HookExecutionPlan, endpoint, stage string) Plan[T] {
	plan := make(Plan[T], 0, len(cfg.Endpoints[endpoint].Stages[stage].Groups))
	for _, groupCfg := range cfg.Endpoints[endpoint].Stages[stage].Groups {
		group := getGroup(getHookFn, groupCfg)
		if len(group.Hooks) > 0 {
			plan = append(plan, group)
		}
	}

	return plan
}

func getGroup[T any](getHookFn hookFn[T], cfg config.HookExecutionGroup) Group[T] {
	group := Group[T]{
		Timeout: time.Duration(cfg.Timeout) * time.Millisecond,
		Hooks:   make([]HookWrapper[T], 0, len(cfg.HookSequence)),
	}

	for _, hookCfg := range cfg.HookSequence {
		if h, ok := getHookFn(hookCfg.Module); ok {
			group.Hooks = append(group.Hooks, HookWrapper[T]{Module: hookCfg.Module, Code: hookCfg.Hook, Hook: h})
		} else {
			glog.Warningf("Not found hook while building hook execution plan: %s %s", hookCfg.Module, hookCfg.Hook)
		}
	}

	return group
}
