package errortypes

// Defines numeric codes for non-specific errors or warnings.
const (
	UnknownCode = 999
)

// Defines numeric codes for well-known errors.
const (
	TimeoutCode = iota + 1
	BadInputCode
	BlacklistedAppCode
	BadServerResponseCode
	FailedToRequestBidsCode
	BidderTemporarilyDisabledCode
	BlacklistedAcctCode
	AcctRequiredCode
)

// Defines numeric codes for well-known warnings.
const (
	InvalidPrivacyConsentWarningCode = iota + 10001
)

// Coder provides an error or warning code.
type Coder interface {
	Code() int
}

// ReadErrorCode returns the error or warning code, or UnknownCode if unavailable.
func ReadErrorCode(err error) int {
	if e, ok := err.(Coder); ok {
		return e.Code()
	}
	return UnknownCode
}
