package privacy

import (
	"strings"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// UserInProhibitedGeoState returns true when the user or device geo.region matches
// one of the states in prohibitedStates where the sale of precise geo data is banned.
// Returns false when prohibitedStates is empty (feature is opt-in).
func UserInProhibitedGeoState(reqWrapper *openrtb_ext.RequestWrapper, prohibitedStates []string) bool {
	if len(prohibitedStates) == 0 {
		return false
	}

	req := reqWrapper.BidRequest
	if req == nil {
		return false
	}

	stateSet := make(map[string]struct{}, len(prohibitedStates))
	for _, s := range prohibitedStates {
		stateSet[strings.ToUpper(s)] = struct{}{}
	}

	if req.User != nil && req.User.Geo != nil {
		if _, ok := stateSet[strings.ToUpper(req.User.Geo.Region)]; ok {
			return true
		}
	}

	if req.Device != nil && req.Device.Geo != nil {
		if _, ok := stateSet[strings.ToUpper(req.Device.Geo.Region)]; ok {
			return true
		}
	}

	return false
}
