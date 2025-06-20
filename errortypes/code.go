package errortypes

// Defines numeric codes for well-known errors.
const (
	UnknownErrorCode = 999
	TimeoutErrorCode = iota
	BadInputErrorCode
	BlockedAppErrorCode
	BadServerResponseErrorCode
	FailedToRequestBidsErrorCode
	BidderTemporarilyDisabledErrorCode
	AccountDisabledErrorCode
	AcctRequiredErrorCode
	NoConversionRateErrorCode
	MalformedAcctErrorCode
	ModuleRejectionErrorCode
	TmaxTimeoutErrorCode
	FailedToMarshalErrorCode
	FailedToUnmarshalErrorCode
	InvalidImpFirstPartyDataErrorCode
)

// Defines numeric codes for well-known warnings.
const (
	UnknownWarningCode               = 10999
	InvalidPrivacyConsentWarningCode = iota + 10000
	AccountLevelDebugDisabledWarningCode
	BidderLevelDebugDisabledWarningCode
	DisabledCurrencyConversionWarningCode
	AlternateBidderCodeWarningCode
	MultiBidWarningCode
	AdServerTargetingWarningCode
	BidAdjustmentWarningCode
	FloorBidRejectionWarningCode
	InvalidBidResponseDSAWarningCode
	SecCookieDeprecationLenWarningCode
	SecBrowsingTopicsWarningCode
	InvalidUserEIDsWarningCode
	InvalidUserUIDsWarningCode
	TooLongTargetingPrefixWarningCode
	TooShortTargetingPrefixWarningCode
)

// Coder provides an error or warning code with severity.
type Coder interface {
	Code() int
	Severity() Severity
}

// ReadCode returns the error or warning code, or UnknownErrorCode if unavailable.
func ReadCode(err error) int {
	if e, ok := err.(Coder); ok {
		return e.Code()
	}
	return UnknownErrorCode
}
