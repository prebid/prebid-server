package hooks

import (
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
)

type Stage string

// Names of the available stages.
const (
	StageEntrypoint               Stage = "entrypoint"
	StageRawAuctionRequest        Stage = "raw_auction_request"
	StageProcessedAuctionRequest  Stage = "processed_auction_request"
	StageBidderRequest            Stage = "bidder_request"
	StageRawBidderResponse        Stage = "raw_bidder_response"
	StageAllProcessedBidResponses Stage = "all_processed_bid_responses"
	StageAuctionResponse          Stage = "auction_response"
)

func (s Stage) String() string {
	return string(s)
}

func (s Stage) IsRejectable() bool {
	return s != StageAllProcessedBidResponses &&
		s != StageAuctionResponse
}

// ExecutionPlanBuilder is the interface that provides methods
// for retrieving hooks grouped and sorted in the established order
// according to the hook execution plan intended for run at a certain stage.
type ExecutionPlanBuilder interface {
	PlanForEntrypointStage(endpoint string) Plan[hookstage.Entrypoint]
	PlanForRawAuctionStage(endpoint string, account *config.Account) Plan[hookstage.RawAuctionRequest]
	PlanForProcessedAuctionStage(endpoint string, account *config.Account) Plan[hookstage.ProcessedAuctionRequest]
	PlanForBidderRequestStage(endpoint string, account *config.Account) Plan[hookstage.BidderRequest]
	PlanForRawBidderResponseStage(endpoint string, account *config.Account) Plan[hookstage.RawBidderResponse]
	PlanForAllProcessedBidResponsesStage(endpoint string, account *config.Account) Plan[hookstage.AllProcessedBidResponses]
	PlanForAuctionResponseStage(endpoint string, account *config.Account) Plan[hookstage.AuctionResponse]
}

// Plan represents a slice of groups of hooks of a specific type grouped in the established order.
type Plan[T any] []Group[T]

// Group represents a slice of hooks sorted in the established order.
type Group[T any] struct {
	// Timeout specifies the max duration in milliseconds that a group of hooks is allowed to run.
	Timeout time.Duration
	// Hooks holds a slice of HookWrapper of a specific type.
	Hooks []HookWrapper[T]
}

// HookWrapper wraps Hook representing specific hook interface
// and holds additional meta information, such as Module name and hook Code.
type HookWrapper[T any] struct {
	// Module holds a name of the module that provides the Hook.
	// Specified in the format "vendor.module_name".
	Module string
	// Code is an arbitrary value assigned to hook via the hook execution plan
	// and is used when sending metrics, logging debug information, etc.
	Code string
	// Hook is an instance of the specific hook interface.
	Hook T
}

// NewExecutionPlanBuilder returns a new instance of the ExecutionPlanBuilder interface.
// Depending on the hooks' status, method returns a real PlanBuilder or the EmptyPlanBuilder.
func NewExecutionPlanBuilder(hooks config.Hooks, repo HookRepository) ExecutionPlanBuilder {
	if hooks.Enabled {
		return PlanBuilder{
			hooks: hooks,
			repo:  repo,
		}
	}
	return EmptyPlanBuilder{}
}

// PlanBuilder is a concrete implementation of the ExecutionPlanBuilder interface.
// Which returns hook execution plans for specific stage defined by the hook config.
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

func (p PlanBuilder) PlanForRawAuctionStage(endpoint string, account *config.Account) Plan[hookstage.RawAuctionRequest] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		StageRawAuctionRequest,
		p.repo.GetRawAuctionHook,
	)
}

func (p PlanBuilder) PlanForProcessedAuctionStage(endpoint string, account *config.Account) Plan[hookstage.ProcessedAuctionRequest] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		StageProcessedAuctionRequest,
		p.repo.GetProcessedAuctionHook,
	)
}

func (p PlanBuilder) PlanForBidderRequestStage(endpoint string, account *config.Account) Plan[hookstage.BidderRequest] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		StageBidderRequest,
		p.repo.GetBidderRequestHook,
	)
}

func (p PlanBuilder) PlanForRawBidderResponseStage(endpoint string, account *config.Account) Plan[hookstage.RawBidderResponse] {
	return getMergedPlan(
		p.hooks,
		account,
		endpoint,
		StageRawBidderResponse,
		p.repo.GetRawBidderResponseHook,
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
	endpoint string,
	stage Stage,
	getHookFn hookFn[T],
) Plan[T] {
	accountPlan := cfg.DefaultAccountExecutionPlan
	if account != nil && account.Hooks.ExecutionPlan.Endpoints != nil {
		accountPlan = account.Hooks.ExecutionPlan
	}

	plan := getPlan(getHookFn, cfg.HostExecutionPlan, endpoint, stage)
	plan = append(plan, getPlan(getHookFn, accountPlan, endpoint, stage)...)

	return plan
}

func getPlan[T any](getHookFn hookFn[T], cfg config.HookExecutionPlan, endpoint string, stage Stage) Plan[T] {
	plan := make(Plan[T], 0, len(cfg.Endpoints[endpoint].Stages[stage.String()].Groups))
	for _, groupCfg := range cfg.Endpoints[endpoint].Stages[stage.String()].Groups {
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
		if h, ok := getHookFn(hookCfg.ModuleCode); ok {
			group.Hooks = append(group.Hooks, HookWrapper[T]{Module: hookCfg.ModuleCode, Code: hookCfg.HookImplCode, Hook: h})
		} else {
			glog.Warningf("Not found hook while building hook execution plan: %s %s", hookCfg.ModuleCode, hookCfg.HookImplCode)
		}
	}

	return group
}
