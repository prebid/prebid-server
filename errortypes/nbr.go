package errortypes

import "github.com/prebid/openrtb/v20/openrtb3"

func GetNBRCodeFromError(err error) openrtb3.NoBidReason {
	switch ReadCode(err) {
	case TimeoutErrorCode, TmaxTimeoutErrorCode:
		return openrtb3.NoBidInsufficientTime
	case BadInputErrorCode, InvalidImpFirstPartyDataErrorCode:
		return openrtb3.NoBidInvalidRequest
	case BlockedAppErrorCode, AccountDisabledErrorCode:
		return openrtb3.NoBidBlockedPublisher
	case AcctRequiredErrorCode, MalformedAcctErrorCode:
		fallthrough
	case BadServerResponseErrorCode, FailedToRequestBidsErrorCode, BidderTemporarilyDisabledErrorCode, NoConversionRateErrorCode:
		fallthrough
	case ModuleRejectionErrorCode:
		fallthrough
	case FailedToUnmarshalErrorCode, FailedToMarshalErrorCode:
		return openrtb3.NoBidTechnicalError
	default:
		return openrtb3.NoBidUnknownError
	}
}
