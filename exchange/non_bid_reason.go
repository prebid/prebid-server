package exchange

import (
	"errors"
	"net"
	"syscall"

	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v2/errortypes"
)

// SeatNonBid list the reasons why bid was not resulted in positive bid
// reason could be either No bid, Error, Request rejection or Response rejection
// Reference:  https://github.com/InteractiveAdvertisingBureau/openrtb/blob/master/extensions/community_extensions/seat-non-bid.md#list-non-bid-status-codes

const (
	ErrorGeneral                           openrtb3.NoBidReason = 100 // Error - General
	ErrorTimeout                           openrtb3.NoBidReason = 101 // Error - Timeout
	ErrorBidderUnreachable                 openrtb3.NoBidReason = 103 // Error - Bidder Unreachable
	ResponseRejectedGeneral                openrtb3.NoBidReason = 300
	ResponseRejectedBelowFloor             openrtb3.NoBidReason = 301 // Response Rejected - Below Floor
	ResponseRejectedCategoryMappingInvalid openrtb3.NoBidReason = 303 // Response Rejected - Category Mapping Invalid
	ResponseRejectedBelowDealFloor         openrtb3.NoBidReason = 304 // Response Rejected - Bid was Below Deal Floor
	ResponseRejectedCreativeSizeNotAllowed openrtb3.NoBidReason = 351 // Response Rejected - Invalid Creative (Size Not Allowed)
	ResponseRejectedCreativeNotSecure      openrtb3.NoBidReason = 352 // Response Rejected - Invalid Creative (Not Secure)
)

func errorToNonBidReason(err error) openrtb3.NoBidReason {
	switch errortypes.ReadCode(err) {
	case errortypes.TimeoutErrorCode:
		return ErrorTimeout
	default:
		return ErrorGeneral
	}
}

// httpInfoToNonBidReason determines NoBidReason code (NBR)
// It will first try to resolve the NBR based on prebid's proprietary error code.
// If proprietary error code not found then it will try to determine NBR using
// system call level error code
func httpInfoToNonBidReason(httpInfo *httpCallInfo) openrtb3.NoBidReason {
	// errorCode := errortypes.ReadCode(httpInfo.err)
	nonBidReason := errorToNonBidReason(httpInfo.err)
	// nonBidReason := errorToNonBidReason(errorCode)
	if nonBidReason != ErrorGeneral {
		return nonBidReason
	}
	if isBidderUnreachableError(httpInfo) {
		return ErrorBidderUnreachable
	}
	return ErrorGeneral
}

// isBidderUnreachableError checks if the error is due to connection refused or no such host
func isBidderUnreachableError(httpInfo *httpCallInfo) bool {
	var dnsErr *net.DNSError
	return errors.Is(httpInfo.err, syscall.ECONNREFUSED) || (errors.As(httpInfo.err, &dnsErr) && dnsErr.IsNotFound)
}
