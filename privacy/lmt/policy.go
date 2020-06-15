package lmt

import (
	"github.com/mxmCherry/openrtb"
)

const (
	trackingUnrestricted = 0
	trackingRestricted   = 1
)

// Policy represents the LMT (Limit Ad Tracking) policy for an OpenRTB bid request.
type Policy struct {
	Signal         int
	SignalProvided bool
}

// ReadPolicy extracts the LMT (Limit Ad Tracking) policy from an OpenRTB bid request.
func ReadPolicy(req *openrtb.BidRequest) Policy {
	policy := Policy{}

	if req != nil && req.Device != nil && req.Device.Lmt != nil {
		policy.Signal = int(*req.Device.Lmt)
		policy.SignalProvided = true
	}

	return policy
}

// ShouldEnforce returns true when the LMT (Limit Ad Tracking) policy is in effect.
func (p Policy) ShouldEnforce() bool {
	return p.SignalProvided && p.Signal == trackingRestricted
}
