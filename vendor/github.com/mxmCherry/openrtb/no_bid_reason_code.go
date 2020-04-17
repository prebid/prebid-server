package openrtb

// 5.24 No-Bid Reason Codes
//
// Options for a bidder to signal the exchange as to why it did not offer a bid for the impression.
type NoBidReasonCode int8

const (
	NoBidReasonCodeUnknownError             NoBidReasonCode = 0  // Unknown Error
	NoBidReasonCodeTechnicalError           NoBidReasonCode = 1  // Technical Error
	NoBidReasonCodeInvalidRequest           NoBidReasonCode = 2  // Invalid Request
	NoBidReasonCodeKnownWebSpider           NoBidReasonCode = 3  // Known Web Spider
	NoBidReasonCodeSuspectedNonHumanTraffic NoBidReasonCode = 4  // Suspected Non-Human Traffic
	NoBidReasonCodeCloudDataCenterProxyIP   NoBidReasonCode = 5  // Cloud, Data center, or Proxy IP
	NoBidReasonCodeUnsupportedDevice        NoBidReasonCode = 6  // Unsupported Device
	NoBidReasonCodeBlockedPublisherOrSite   NoBidReasonCode = 7  // Blocked Publisher or Site
	NoBidReasonCodeUnmatchedUser            NoBidReasonCode = 8  // Unmatched User
	NoBidReasonCodeDailyReaderCapMet        NoBidReasonCode = 9  // Daily Reader Cap Met
	NoBidReasonCodeDailyDomainCapMet        NoBidReasonCode = 10 // Daily Domain Cap Met
)

// Ptr returns pointer to own value.
func (c NoBidReasonCode) Ptr() *NoBidReasonCode {
	return &c
}

// Val safely dereferences pointer, returning default value (NoBidReasonCodeUnknownError) for nil.
func (c *NoBidReasonCode) Val() NoBidReasonCode {
	if c == nil {
		return NoBidReasonCodeUnknownError
	}
	return *c
}
