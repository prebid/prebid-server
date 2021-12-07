package privacy

import "github.com/mxmCherry/openrtb/v15/openrtb2"

// PolicyWriter mutates an OpenRTB bid request with a policy's regulatory information.
type PolicyWriter interface {
	Write(req *openrtb2.BidRequest) error
}

// NilPolicyWriter implements the PolicyWriter interface but performs no action.
type NilPolicyWriter struct{}

// Write is hardcoded to perform no action with the OpenRTB bid request.
func (NilPolicyWriter) Write(req *openrtb2.BidRequest) error {
	return nil
}
