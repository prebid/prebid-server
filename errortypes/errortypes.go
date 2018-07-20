package errortypes

// These define the error codes for all the errors enumerated in this package
// NoErrorCode is to reserve 0 for non error states.
const (
	NoErrorCode = iota
	TimeoutCode
	BadInputCode
	BadServerResponseCode
)

// We should use this code for any Error interface that is not in this package
const UnknownErrorCode = 999

// PBSError provides an interface to use if we want to deal with any error type created in this package.
type PBSError interface {
	Error() string
	Code() int
}

// Timeout should be used to flag that a bidder failed to return a response because the PBS timeout timer
// expired before a result was recieved.
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
