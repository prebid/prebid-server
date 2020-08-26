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

func (NilPolicyWriter) Write(req *openrtb.BidRequest) error {
	return nil
}

// InvalidConsentError represents an error parsing or validating a consent string.
type InvalidConsentError struct {
	Message string
}

func (err *InvalidConsentError) Error() string {
	return err.Message
}
