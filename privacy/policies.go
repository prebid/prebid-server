package privacy

import (
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
)

// Policies represents the privacy regulations for an OpenRTB bid request.
type Policies struct {
	GDPR gdpr.Policy
	CCPA ccpa.Policy
}
