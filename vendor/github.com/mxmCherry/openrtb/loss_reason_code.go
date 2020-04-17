package openrtb

// 5.25 Loss Reason Codes
//
// Options for an exchange to inform a bidder as to the reason why they did not win an impression.
type LossReasonCode int64

const (
	LossReasonCodeBidWon                                      LossReasonCode = 0   // Bid Won
	LossReasonCodeInternalError                               LossReasonCode = 1   // Internal Error
	LossReasonCodeImpressionOpportunityExpired                LossReasonCode = 2   // Impression Opportunity Expired
	LossReasonCodeInvalidBidResponse                          LossReasonCode = 3   // Invalid Bid Response
	LossReasonCodeInvalidDealID                               LossReasonCode = 4   // Invalid Deal ID
	LossReasonCodeInvalidAuctionID                            LossReasonCode = 5   // Invalid Auction ID
	LossReasonCodeInvalidAdvertiserDomain                     LossReasonCode = 6   // Invalid (i.e., malformed) Advertiser Domain
	LossReasonCodeMissingMarkup                               LossReasonCode = 7   // Missing Markup
	LossReasonCodeMissingCreativeID                           LossReasonCode = 8   // Missing Creative ID
	LossReasonCodeMissingBidPrice                             LossReasonCode = 9   // Missing Bid Price
	LossReasonCodeMissingMinimumCreativeApprovalData          LossReasonCode = 10  // Missing Minimum Creative Approval Data
	LossReasonCodeBidBelowAuctionFloor                        LossReasonCode = 100 // Bid was Below Auction Floor
	LossReasonCodeBidBelowDealFloor                           LossReasonCode = 101 // Bid was Below Deal Floor
	LossReasonCodeLostToHigherBid                             LossReasonCode = 102 // Lost to Higher Bid
	LossReasonCodeLostToBidForPMPDeal                         LossReasonCode = 103 // Lost to a Bid for a PMP Deal
	LossReasonCodeBuyerSeatBlocked                            LossReasonCode = 104 // Buyer Seat Blocked
	LossReasonCodeCreativeFilteredGeneral                     LossReasonCode = 200 // Creative Filtered – General; reason unknown.
	LossReasonCodeCreativeFilteredPendingProcessingByExchange LossReasonCode = 201 // Creative Filtered – Pending processing by Exchange (e.g., approval, transcoding, etc.)
	LossReasonCodeCreativeFilteredDisapprovedByExchange       LossReasonCode = 202 // Creative Filtered – Disapproved by Exchange
	LossReasonCodeCreativeFilteredSizeNotAllowed              LossReasonCode = 203 // Creative Filtered – Size Not Allowed
	LossReasonCodeCreativeFilteredIncorrectCreativeFormat     LossReasonCode = 204 // Creative Filtered – Incorrect Creative Format
	LossReasonCodeCreativeFilteredAdvertiserExclusions        LossReasonCode = 205 // Creative Filtered – Advertiser Exclusions
	LossReasonCodeCreativeFilteredAppBundleExclusions         LossReasonCode = 206 // Creative Filtered – App Bundle Exclusions
	LossReasonCodeCreativeFilteredNotSecure                   LossReasonCode = 207 // Creative Filtered – Not Secure
	LossReasonCodeCreativeFilteredLanguageExclusions          LossReasonCode = 208 // Creative Filtered – Language Exclusions
	LossReasonCodeCreativeFilteredCategoryExclusions          LossReasonCode = 209 // Creative Filtered – Category Exclusions
	LossReasonCodeCreativeFilteredCreativeAttributeExclusions LossReasonCode = 210 // Creative Filtered – Creative Attribute Exclusions
	LossReasonCodeCreativeFilteredAdTypeExclusions            LossReasonCode = 211 // Creative Filtered – Ad Type Exclusions
	LossReasonCodeCreativeFilteredAnimationTooLong            LossReasonCode = 212 // Creative Filtered – Animation Too Long
	LossReasonCodeCreativeFilteredNotAllowedInPMPDeal         LossReasonCode = 213 // Creative Filtered – Not Allowed in PMP Deal

	// ≥ 1000 Exchange specific (should be communicated to bidders a priori)
)
