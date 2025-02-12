package errortypes

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
	return TimeoutErrorCode
}

func (err *Timeout) Severity() Severity {
	return SeverityFatal
}

// TmaxTimeout should be used to flag that remaining tmax duration is not enough to get response from bidder
//
// TmaxTimeout will not be written to the app log, since it's not an actionable item for the Prebid Server hosts.
type TmaxTimeout struct {
	Message string
}

func (err *TmaxTimeout) Error() string {
	return err.Message
}

func (err *TmaxTimeout) Code() int {
	return TmaxTimeoutErrorCode
}

func (err *TmaxTimeout) Severity() Severity {
	return SeverityFatal
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
	return BadInputErrorCode
}

func (err *BadInput) Severity() Severity {
	return SeverityFatal
}

// BlockedApp should be used when a request App.ID matches an entry in the BlockedApp configuration.
type BlockedApp struct {
	Message string
}

func (err *BlockedApp) Error() string {
	return err.Message
}

func (err *BlockedApp) Code() int {
	return BlockedAppErrorCode
}

func (err *BlockedApp) Severity() Severity {
	return SeverityFatal
}

// AccountDisabled should be used when a request an account is specifically disabled in account config.
type AccountDisabled struct {
	Message string
}

func (err *AccountDisabled) Error() string {
	return err.Message
}

func (err *AccountDisabled) Code() int {
	return AccountDisabledErrorCode
}

func (err *AccountDisabled) Severity() Severity {
	return SeverityFatal
}

// AcctRequired should be used when the environment variable ACCOUNT_REQUIRED has been set to not
// process requests that don't come with a valid account ID
//
// These errors will be written to  http.ResponseWriter before canceling execution
type AcctRequired struct {
	Message string
}

func (err *AcctRequired) Error() string {
	return err.Message
}

func (err *AcctRequired) Code() int {
	return AcctRequiredErrorCode
}

func (err *AcctRequired) Severity() Severity {
	return SeverityFatal
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
	return BadServerResponseErrorCode
}

func (err *BadServerResponse) Severity() Severity {
	return SeverityFatal
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
	return FailedToRequestBidsErrorCode
}

func (err *FailedToRequestBids) Severity() Severity {
	return SeverityFatal
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
	return BidderTemporarilyDisabledErrorCode
}

func (err *BidderTemporarilyDisabled) Severity() Severity {
	return SeverityWarning
}

// MalformedAcct should be used when the retrieved account config cannot be unmarshaled
// These errors will be written to http.ResponseWriter before canceling execution
type MalformedAcct struct {
	Message string
}

func (err *MalformedAcct) Error() string {
	return err.Message
}

func (err *MalformedAcct) Code() int {
	return MalformedAcctErrorCode
}

func (err *MalformedAcct) Severity() Severity {
	return SeverityFatal
}

// Warning is a generic non-fatal error. Throughout the codebase, an error can
// only be a warning if it's of the type defined below
type Warning struct {
	Message     string
	WarningCode int
}

func (err *Warning) Error() string {
	return err.Message
}

func (err *Warning) Code() int {
	return err.WarningCode
}

func (err *Warning) Severity() Severity {
	return SeverityWarning
}

// FailedToUnmarshal should be used to represent errors that occur when unmarshaling raw json.
type FailedToUnmarshal struct {
	Message string
}

func (err *FailedToUnmarshal) Error() string {
	return err.Message
}

func (err *FailedToUnmarshal) Code() int {
	return FailedToUnmarshalErrorCode
}

func (err *FailedToUnmarshal) Severity() Severity {
	return SeverityFatal
}

// FailedToMarshal should be used to represent errors that occur when marshaling to a byte slice.
type FailedToMarshal struct {
	Message string
}

func (err *FailedToMarshal) Error() string {
	return err.Message
}

func (err *FailedToMarshal) Code() int {
	return FailedToMarshalErrorCode
}

func (err *FailedToMarshal) Severity() Severity {
	return SeverityFatal
}

// DebugWarning is a generic non-fatal error used in debug mode. Throughout the codebase, an error can
// only be a warning if it's of the type defined below
type DebugWarning struct {
	Message     string
	WarningCode int
}

func (err *DebugWarning) Error() string {
	return err.Message
}

func (err *DebugWarning) Code() int {
	return err.WarningCode
}

func (err *DebugWarning) Severity() Severity {
	return SeverityWarning
}

func (err *DebugWarning) Scope() Scope {
	return ScopeDebug
}

// InvalidImpFirstPartyData should be used when the retrieved account config cannot be unmarshaled
// These errors will be written to http.ResponseWriter before canceling execution
type InvalidImpFirstPartyData struct {
	Message string
}

func (err *InvalidImpFirstPartyData) Error() string {
	return err.Message
}

func (err *InvalidImpFirstPartyData) Code() int {
	return InvalidImpFirstPartyDataErrorCode
}

func (err *InvalidImpFirstPartyData) Severity() Severity {
	return SeverityFatal
}
