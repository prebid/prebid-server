package errortypes

// These define the error codes for all the errors enumerated in this package
// NoErrorCode is to reserve 0 for non error states.
const (
	NoErrorCode = iota
	TimeoutCode
	BadInputCode
	BlacklistedAppCode
	BadServerResponseCode
	FailedToRequestBidsCode
	BidderTemporarilyDisabledCode
)

// We should use this code for any Error interface that is not in this package
const UnknownErrorCode = 999

// Coder provides an interface to use if we want to check the code of an error type created in this package.
type Coder interface {
	Code() int
}

// Timeout should be used to flag that a bidder failed to return a response because the PBS timeout timer
// expired before a result was received.
//
// Timeouts will not be written to the app log, since it's not an actionable item for the Prebid Server hosts.
type Timeout struct {
	Message string
}

func (err *Timeout) Error() string {
	return err.Message
}

func (err *Timeout) Code() int {
	return TimeoutCode
}

// BadInput should be used when returning errors which are caused by bad input.
// It should _not_ be used if the error is a server-side issue (e.g. failed to send the external request).
//
// BadInputs will not be written to the app log, since it's not an actionable item for the Prebid Server hosts.
type BadInput struct {
	Message string
}

func (err *BadInput) Error() string {
	return err.Message
}

func (err *BadInput) Code() int {
	return BadInputCode
}

// BlacklistedApp should be used when a request App.ID matches an entry in the BlacklistedApps
// environment variable array
//
// These errors will be written to  http.ResponseWriter before canceling execution
type BlacklistedApp struct {
	Message string
}

func (err *BlacklistedApp) Error() string {
	return err.Message
}

func (err *BlacklistedApp) Code() int {
	return BlacklistedAppCode
}

// BadServerResponse should be used when returning errors which are caused by bad/unexpected behavior on the remote server.
//
// For example:
//
//   - The external server responded with a 500
//   - The external server gave a malformed or unexpected response.
//
// These should not be used to log _connection_ errors (e.g. "couldn't find host"),
// which may indicate config issues for the PBS host company
type BadServerResponse struct {
	Message string
}

func (err *BadServerResponse) Error() string {
	return err.Message
}

func (err *BadServerResponse) Code() int {
	return BadServerResponseCode
}

// FailedToRequestBids is an error to cover the case where an adapter failed to generate any http requests to get bids,
// but did not generate any error messages. This should not happen in practice and will signal that an adapter is poorly
// coded. If there was something wrong with a request such that an adapter could not generate a bid, then it should
// generate an error explaining the deficiency. Otherwise it will be extremely difficult to debug the reason why an
// adapter is not bidding.
type FailedToRequestBids struct {
	Message string
}

func (err *FailedToRequestBids) Error() string {
	return err.Message
}

func (err *FailedToRequestBids) Code() int {
	return FailedToRequestBidsCode
}

// BidderTemporarilyDisabled is used at the request validation step, where we want to continue processing as best we
// can rather than returning a 4xx, and still return an error message.
// The initial usecase is to flag deprecated bidders.
type BidderTemporarilyDisabled struct {
	Message string
}

func (err *BidderTemporarilyDisabled) Error() string {
	return err.Message
}

func (err *BidderTemporarilyDisabled) Code() int {
	return BidderTemporarilyDisabledCode
}

// DecodeError provides the error code for an error, as defined above
func DecodeError(err error) int {
	if ce, ok := err.(Coder); ok {
		return ce.Code()
	}
	return UnknownErrorCode
}
