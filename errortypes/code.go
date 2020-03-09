package errortypes

// Defines numeric codes for well-known errors in this package.
const (
	NoErrorCode = iota
	TimeoutCode
	BadInputCode
	BlacklistedAppCode
	BadServerResponseCode
	FailedToRequestBidsCode
	BidderTemporarilyDisabledCode
	BlacklistedAcctCode
	AcctRequiredCode
	InvalidPrivacyConsentCode
	UnknownErrorCode = 999
)

// Coder provides an error code.
type Coder interface {
	Code() int
}

// ReadErrorCode returns the error code, or UnknownErrorCode if unavailable.
func ReadErrorCode(err error) int {
	if e, ok := err.(Coder); ok {
		return e.Code()
	}
	return UnknownErrorCode
}
