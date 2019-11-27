package privacy

import (
	"github.com/mxmCherry/openrtb"

	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
)

// Policies represents the privacy policies and regulations of an OpenRTB bid request.
type Policies struct {
	GDPR gdpr.Policy
	CCPA ccpa.Policy
}

// Write mutates an OpenRTB bid request with the context of the policies.
func (p Policies) Write(req *openrtb.BidRequest) error {
	if err := p.GDPR.Write(req); err != nil {
		return err
	}

	if err := p.CCPA.Write(req); err != nil {
		return err
	}

	return nil
}
