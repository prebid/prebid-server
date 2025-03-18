package exchange

import (
	"errors"
	"net"
	"syscall"

	"github.com/prebid/prebid-server/v3/errortypes"
)

// SeatNonBid list the reasons why bid was not resulted in positive bid
// reason could be either No bid, Error, Request rejection or Response rejection
// Reference:  https://github.com/InteractiveAdvertisingBureau/openrtb/blob/master/extensions/community_extensions/seat-non-bid.md#list-non-bid-status-codes
type NonBidReason int64

const (
	ErrorGeneral                           NonBidReason = 100 // Error - General
	ErrorTimeout                           NonBidReason = 101 // Error - Timeout
	ErrorBidderUnreachable                 NonBidReason = 103 // Error - Bidder Unreachable
	ResponseRejectedGeneral                NonBidReason = 300
	ResponseRejectedBelowFloor             NonBidReason = 301 // Response Rejected - Below Floor
	ResponseRejectedCategoryMappingInvalid NonBidReason = 303 // Response Rejected - Category Mapping Invalid
	ResponseRejectedBelowDealFloor         NonBidReason = 304 // Response Rejected - Bid was Below Deal Floor
	ResponseRejectedCreativeSizeNotAllowed NonBidReason = 351 // Response Rejected - Invalid Creative (Size Not Allowed)
	ResponseRejectedCreativeNotSecure      NonBidReason = 352 // Response Rejected - Invalid Creative (Not Secure)
)

func errorToNonBidReason(err error) NonBidReason {
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
func httpInfoToNonBidReason(httpInfo *httpCallInfo) NonBidReason {
	nonBidReason := errorToNonBidReason(httpInfo.err)
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
	isNoSuchHost := errors.As(httpInfo.err, &dnsErr) && dnsErr.IsNotFound
	return errors.Is(httpInfo.err, syscall.ECONNREFUSED) || isNoSuchHost
}
