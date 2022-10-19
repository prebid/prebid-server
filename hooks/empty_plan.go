package hooks

import (
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks/stages"
)

type EmptyPlanBuilder struct{}

func (e EmptyPlanBuilder) PlanForEntrypointStage(endpoint string) Plan[stages.EntrypointHook] {
	return nil
}

func (e EmptyPlanBuilder) PlanForRawAuctionStage(endpoint string, account *config.Account) Plan[stages.RawAuctionHook] {
	return nil
}

func (e EmptyPlanBuilder) PlanForProcessedAuctionStage(endpoint string, account *config.Account) Plan[stages.ProcessedAuctionHook] {
	return nil
}

func (e EmptyPlanBuilder) PlanForBidRequestStage(endpoint string, account *config.Account) Plan[stages.BidRequestHook] {
	return nil
}

func (e EmptyPlanBuilder) PlanForRawBidResponseStage(endpoint string, account *config.Account) Plan[stages.RawBidResponseHook] {
	return nil
}

func (e EmptyPlanBuilder) PlanForAllProcessedBidResponsesStage(endpoint string, account *config.Account) Plan[stages.AllProcBidResponsesHook] {
	return nil
}

func (e EmptyPlanBuilder) PlanForAuctionResponseStage(endpoint string, account *config.Account) Plan[stages.AuctionResponseHook] {
	return nil
}
