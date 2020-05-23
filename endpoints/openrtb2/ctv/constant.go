package ctv

type ErrorCode = int
type FilterReasonCode = int

const (
	CTVErrorNoValidImpressionsForAdPodConfig ErrorCode = 601

	//Filter Reason Code
	CTVRCDidNotGetChance   FilterReasonCode = 700
	CTVRCWinningBid        FilterReasonCode = 701
	CTVRCCategoryExclusion FilterReasonCode = 702
	CTVRCDomainExclusion   FilterReasonCode = 703
)
