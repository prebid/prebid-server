package plans

import (
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks/hep"
	"github.com/prebid/prebid-server/hooks/stages"
)

type EmptyPlanBuilder struct{}

func (e EmptyPlanBuilder) PlanForEntrypointStage(endpoint string) hep.Plan[stages.EntrypointHook] {
	return nil
}

func (e EmptyPlanBuilder) PlanForRawAuctionStage(endpoint string, account *config.Account) hep.Plan[stages.RawAuctionHook] {
	return nil
}

func (e EmptyPlanBuilder) PlanForProcessedAuctionStage(endpoint string, account *config.Account) hep.Plan[stages.ProcessedAuctionHook] {
	return nil
}

func (e EmptyPlanBuilder) PlanForBidRequestStage(endpoint string, account *config.Account) hep.Plan[stages.BidRequestHook] {
	return nil
}

func (e EmptyPlanBuilder) PlanForRawBidResponseStage(endpoint string, account *config.Account) hep.Plan[stages.RawBidResponseHook] {
	return nil
}

func (e EmptyPlanBuilder) PlanForAllProcBidResponsesStage(endpoint string, account *config.Account) hep.Plan[stages.AllProcBidResponsesHook] {
	return nil
}

func (e EmptyPlanBuilder) PlanForAuctionResponseStage(endpoint string, account *config.Account) hep.Plan[stages.AuctionResponseHook] {
	return nil
}
