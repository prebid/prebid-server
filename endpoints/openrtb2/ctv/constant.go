package ctv

type ErrorCode = int
type FilterReasonCode = int

const (
	PrebidCTVSeatName        = `prebid_ctv`
	CTVImpressionIDSeparator = `_`
	CTVImpressionIDFormat    = `%v` + CTVImpressionIDSeparator + `%v`
	CTVUniqueBidIDFormat     = `%v-%v`
	HTTPPrefix               = `http`

	//VAST Constants
	VASTDefaultVersion    = 2.0
	VASTMaxVersion        = 4.0
	VASTDefaultVersionStr = `2.0`
	VASTDefaultTag        = `<VAST version="` + VASTDefaultVersionStr + `"/>`
	VASTElement           = `VAST`
	VASTAdElement         = `Ad`
	VASTWrapperElement    = `Wrapper`
	VASTAdTagURIElement   = `VASTAdTagURI`
	VASTVersionAttribute  = `version`
	VASTSequenceAttribute = `sequence`

	CTVAdpod  = `adpod`
	CTVOffset = `offset`
)

var (
	VASTVersionsStr = []string{"0", "1.0", "2.0", "3.0", "4.0"}
)

const (
	CTVErrorNoValidImpressionsForAdPodConfig ErrorCode = 601

	//Filter Reason Code
	CTVRCDidNotGetChance   FilterReasonCode = 0
	CTVRCWinningBid        FilterReasonCode = 1
	CTVRCCategoryExclusion FilterReasonCode = 2
	CTVRCDomainExclusion   FilterReasonCode = 3
)
