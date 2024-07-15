package exchange

import (
	"errors"
	"strings"
	"syscall"

	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v2/errortypes"
)

// SeatNonBid list the reasons why bid was not resulted in positive bid
// reason could be either No bid, Error, Request rejection or Response rejection
// Reference:  https://github.com/InteractiveAdvertisingBureau/openrtb/blob/master/extensions/community_extensions/seat-non-bid.md#list-non-bid-status-codes

const (
	ResponseRejectedGeneral                openrtb3.NoBidReason = 300
	ResponseRejectedBelowFloor             openrtb3.NoBidReason = 301 // Response Rejected - Below Floor
	ResponseRejectedCategoryMappingInvalid openrtb3.NoBidReason = 303 // Response Rejected - Category Mapping Invalid
	ResponseRejectedBelowDealFloor         openrtb3.NoBidReason = 304 // Response Rejected - Bid was Below Deal Floor
	ResponseRejectedCreativeSizeNotAllowed openrtb3.NoBidReason = 351 // Response Rejected - Invalid Creative (Size Not Allowed)
	ResponseRejectedCreativeNotSecure      openrtb3.NoBidReason = 352 // Response Rejected - Invalid Creative (Not Secure)
	ErrorTimeout                           openrtb3.NoBidReason = 101 // Error - Timeout
	ErrorGeneral                           openrtb3.NoBidReason = 100 // Error - General
	ErrorBidderUnreachable                 openrtb3.NoBidReason = 103 // Error - Bidder Unreachable
)

func errorToNonBidReason(errorCode int) openrtb3.NoBidReason {
	switch errorCode {
	case errortypes.TimeoutErrorCode:
		return ErrorTimeout
	}
	// return 0
	return ErrorGeneral
}

// httpInfoToNonBidReason determines NoBidReason code (NBR)
// It will first try to resolve the NBR based on prebid's proprietary error code.
// If proprietary error code not found then it will try to determine NBR using
// system call level error code
func httpInfoToNonBidReason(httpInfo *httpCallInfo) openrtb3.NoBidReason {
	errorCode := errortypes.ReadCode(httpInfo.err)
	nonBidReason := errorToNonBidReason(errorCode)
	if nonBidReason == ErrorGeneral {
		if isBidderUnreachableError(httpInfo) {
			return ErrorBidderUnreachable
		}
	}
	return nonBidReason
}

// isBidderUnreachableError checks if the error is due to connection refused or no such host
func isBidderUnreachableError(httpInfo *httpCallInfo) bool {
	return errors.Is(httpInfo.err, syscall.ECONNREFUSED) || strings.Contains(httpInfo.err.Error(), "no such host")
}
