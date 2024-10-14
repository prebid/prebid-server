package lmt

import "github.com/prebid/openrtb/v20/openrtb2"

const (
	trackingUnrestricted = 0
	trackingRestricted   = 1
)

// Policy represents the LMT (Limit Ad Tracking) policy for an OpenRTB bid request.
type Policy struct {
	Signal         int
	SignalProvided bool
}

// ReadFromRequest extracts the LMT (Limit Ad Tracking) policy from an OpenRTB bid request.
func ReadFromRequest(req *openrtb2.BidRequest) (policy Policy) {
	if req != nil && req.Device != nil && req.Device.Lmt != nil {
		policy.Signal = int(*req.Device.Lmt)
		policy.SignalProvided = true
	}
	return
}

// CanEnforce returns true the LMT (Limit Ad Tracking) signal is provided by the publisher.
func (p Policy) CanEnforce() bool {
	return p.SignalProvided
}

// ShouldEnforce returns true when the LMT (Limit Ad Tracking) policy is in effect.
func (p Policy) ShouldEnforce(bidder string) bool {
	return p.SignalProvided && p.Signal == trackingRestricted
}
