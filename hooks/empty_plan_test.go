package hooks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptyPlanBuilder(t *testing.T) {
	planBuilder := EmptyPlanBuilder{}
	endpoint := "/openrtb2/auction"
	message := "Empty plan builder should always return empty hook execution plan for %s stage."

	assert.Len(t, planBuilder.PlanForEntrypointStage(endpoint), 0, message, StageEntrypoint)
	assert.Len(t, planBuilder.PlanForRawAuctionStage(endpoint, nil), 0, message, StageRawAuctionRequest)
	assert.Len(t, planBuilder.PlanForProcessedAuctionStage(endpoint, nil), 0, message, StageProcessedAuctionRequest)
	assert.Len(t, planBuilder.PlanForBidderRequestStage(endpoint, nil), 0, message, StageBidderRequest)
	assert.Len(t, planBuilder.PlanForRawBidderResponseStage(endpoint, nil), 0, message, StageRawBidderResponse)
	assert.Len(t, planBuilder.PlanForAllProcessedBidResponsesStage(endpoint, nil), 0, message, StageAllProcessedBidResponses)
	assert.Len(t, planBuilder.PlanForAuctionResponseStage(endpoint, nil), 0, message, StageAuctionResponse)
}
