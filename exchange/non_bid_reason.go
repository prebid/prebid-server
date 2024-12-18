package exchange

import (
	"errors"
	"net"
	"syscall"

	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func errorToNonBidReason(err error) openrtb_ext.NonBidReason {
	switch errortypes.ReadCode(err) {
	case errortypes.TimeoutErrorCode:
		return openrtb_ext.ErrorTimeout
	default:
		return openrtb_ext.ErrorGeneral
	}
}

// httpInfoToNonBidReason determines NoBidReason code (NBR)
// It will first try to resolve the NBR based on prebid's proprietary error code.
// If proprietary error code not found then it will try to determine NBR using
// system call level error code
func httpInfoToNonBidReason(httpInfo *httpCallInfo) openrtb_ext.NonBidReason {
	nonBidReason := errorToNonBidReason(httpInfo.err)
	if nonBidReason != openrtb_ext.ErrorGeneral {
		return nonBidReason
	}
	if isBidderUnreachableError(httpInfo) {
		return openrtb_ext.ErrorBidderUnreachable
	}
	return openrtb_ext.ErrorGeneral
}

// isBidderUnreachableError checks if the error is due to connection refused or no such host
func isBidderUnreachableError(httpInfo *httpCallInfo) bool {
	var dnsErr *net.DNSError
	isNoSuchHost := errors.As(httpInfo.err, &dnsErr) && dnsErr.IsNotFound
	return errors.Is(httpInfo.err, syscall.ECONNREFUSED) || isNoSuchHost
}
