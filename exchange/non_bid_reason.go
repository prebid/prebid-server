package exchange

import (
	"net"
	"net/url"
	"os"
	"syscall"

	"github.com/prebid/prebid-server/v2/errortypes"
)

// import "github.com/prebid/prebid-server/errortypes"

// SeatNonBid list the reasons why bid was not resulted in positive bid
// reason could be either No bid, Error, Request rejection or Response rejection
// Reference:  https://github.com/InteractiveAdvertisingBureau/openrtb/blob/master/extensions/community_extensions/seat-non-bid.md#list-non-bid-status-codes
type NonBidReason int

const (
	NoBidUnknownError                      NonBidReason = 0 // No Bid - General
	ResponseRejectedGeneral                NonBidReason = 300
	ResponseRejectedBelowFloor             NonBidReason = 301 // Response Rejected - Below Floor
	ResponseRejectedCategoryMappingInvalid NonBidReason = 303 // Response Rejected - Category Mapping Invalid
	ResponseRejectedBelowDealFloor         NonBidReason = 304 // Response Rejected - Bid was Below Deal Floor
	ResponseRejectedCreativeSizeNotAllowed NonBidReason = 351 // Response Rejected - Invalid Creative (Size Not Allowed)
	ResponseRejectedCreativeNotSecure      NonBidReason = 352 // Response Rejected - Invalid Creative (Not Secure)
	ErrorTimeout                           NonBidReason = 101 // Error - Timeout
	ErrorGeneral                           NonBidReason = 100
	ErrorBidderUnreachable                 NonBidReason = 103 // Error - Bidder Unreachable
)

// Ptr returns pointer to own value.
func (n NonBidReason) Ptr() *NonBidReason {
	return &n
}

// Val safely dereferences pointer, returning default value (NoBidUnknownError) for nil.
func (n *NonBidReason) Val() NonBidReason {
	if n == nil {
		return NoBidUnknownError
	}
	return *n
}

func errorToNonBidReason(errorCode int) NonBidReason {
	switch errorCode {
	case errortypes.TimeoutErrorCode:
		return ErrorTimeout
	}
	// return 0
	return ErrorGeneral
}

func httpInfoToNonBidReason(httpInfo *httpCallInfo) NonBidReason {
	if uError, ok := httpInfo.err.(*url.Error); ok {
		if opError, ok := uError.Err.(*net.OpError); ok {
			if sysCallErr, ok := opError.Err.(*os.SyscallError); ok {
				// fmt.Println(sysCallErr.Err.(syscall.Errno))
				// fmt.Println(uError.Unwrap())
				// fmt.Println(sysCallErr.Err)
				sysErr := sysCallErr.Err.(syscall.Errno)
				switch sysErr {
				case syscall.ECONNREFUSED:
					return ErrorBidderUnreachable
				}
			}

		}
	}
	return ErrorGeneral
}

func HttpInfoToNonBidReason(httpInfo *httpCallInfo) NonBidReason {
	errorCode := errortypes.ReadCode(httpInfo.err)
	nonBidReason := errorToNonBidReason(errorCode)
	if nonBidReason == ErrorGeneral {
		nonBidReason = httpInfoToNonBidReason(httpInfo)
	}
	return nonBidReason
}
