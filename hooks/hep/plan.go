package hep

import (
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks/stages"
)

const (
	StageEntrypoint          = "entrypoint"
	StageRawauction          = "rawauction"
	StageProcauction         = "procauction"
	StageBidrequest          = "bidrequest"
	StageRawbidresponse      = "rawbidresponse"
	StageAllprocbidresponses = "allprocbidresponses"
	StageAuctionresponse     = "auctionresponse"
)

type HookExecutionPlanBuilder interface {
	PlanForEntrypointStage(endpoint string) Plan[stages.EntrypointHook]
	PlanForRawAuctionStage(endpoint string, account *config.Account) Plan[stages.RawAuctionHook]
	PlanForProcessedAuctionStage(endpoint string, account *config.Account) Plan[stages.ProcessedAuctionHook]
	PlanForBidRequestStage(endpoint string, account *config.Account) Plan[stages.BidRequestHook]
	PlanForRawBidResponseStage(endpoint string, account *config.Account) Plan[stages.RawBidResponseHook]
	PlanForAllProcBidResponsesStage(endpoint string, account *config.Account) Plan[stages.AllProcBidResponsesHook]
	PlanForAuctionResponseStage(endpoint string, account *config.Account) Plan[stages.AuctionResponseHook]
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
