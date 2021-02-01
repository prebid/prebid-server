package privacy

import (
	"github.com/mxmCherry/openrtb"
)

// PolicyWriter mutates an OpenRTB bid request with a policy's regulatory information.
type PolicyWriter interface {
	Write(req *openrtb.BidRequest) error
}

// NilPolicyWriter implements the PolicyWriter interface but performs no action.
type NilPolicyWriter struct{}

// Write is hardcoded to perform no action with the OpenRTB bid request.
func (NilPolicyWriter) Write(req *openrtb.BidRequest) error {
	return nil
}
