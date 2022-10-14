package plans

import (
	"testing"

	"github.com/prebid/prebid-server/hooks/hep"
	"github.com/stretchr/testify/assert"
)

func TestEmptyPlanBuilder(t *testing.T) {
	planBuilder := EmptyPlanBuilder{}
	message := "Empty plan builder should always return empty hook execution plan."

	assert.Len(t, planBuilder.PlanForEntrypointStage(hep.StageEntrypoint), 0, message)
	assert.Len(t, planBuilder.PlanForRawAuctionStage(hep.StageRawauction, nil), 0, message)
	assert.Len(t, planBuilder.PlanForProcessedAuctionStage(hep.StageProcauction, nil), 0, message)
	assert.Len(t, planBuilder.PlanForBidRequestStage(hep.StageBidrequest, nil), 0, message)
	assert.Len(t, planBuilder.PlanForRawBidResponseStage(hep.StageRawbidresponse, nil), 0, message)
	assert.Len(t, planBuilder.PlanForAllProcBidResponsesStage(hep.StageAllprocbidresponses, nil), 0, message)
	assert.Len(t, planBuilder.PlanForAuctionResponseStage(hep.StageAuctionresponse, nil), 0, message)
}
