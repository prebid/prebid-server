package hooks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptyPlanBuilder(t *testing.T) {
	planBuilder := EmptyPlanBuilder{}
	message := "Empty plan builder should always return empty hook execution plan for %s stage."

	assert.Len(t, planBuilder.PlanForEntrypointStage(StageEntrypoint), 0, message, StageEntrypoint)
	assert.Len(t, planBuilder.PlanForRawAuctionStage(StageRawAuction, nil), 0, message, StageRawAuction)
	assert.Len(t, planBuilder.PlanForProcessedAuctionStage(StageProcessedAuction, nil), 0, message, StageProcessedAuction)
	assert.Len(t, planBuilder.PlanForBidderRequestStage(StageBidderRequest, nil), 0, message, StageBidderRequest)
	assert.Len(t, planBuilder.PlanForRawBidderResponseStage(StageRawBidderResponse, nil), 0, message, StageRawBidderResponse)
	assert.Len(t, planBuilder.PlanForAllProcessedBidResponsesStage(StageAllProcessedBidResponses, nil), 0, message, StageAllProcessedBidResponses)
	assert.Len(t, planBuilder.PlanForAuctionResponseStage(StageAuctionResponse, nil), 0, message, StageAuctionResponse)
}
