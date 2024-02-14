package dsa

import (
	"github.com/prebid/prebid-server/v2/exchange/entities"
	"github.com/prebid/prebid-server/v2/openrtb_ext"

	"github.com/buger/jsonparser"
)

const (
	// Required - bid responses without DSA object will not be accepted
	Required = 2
	// RequiredOnlinePlatform - bid responses without DSA object will not be accepted, Publisher is an Online Platform
	RequiredOnlinePlatform = 3
)

// Validate determines whether a given bid is valid from a DSA perspective.
// A bid is considered valid unless the bid request indicates that a DSA object is required
// in bid responses and it the object happens to be missing from the specified bid.
func Validate(req *openrtb_ext.RequestWrapper, bid *entities.PbsOrtbBid) (valid bool) {
	if !dsaRequired(req) {
		return true
	}
	if bid == nil || bid.Bid == nil {
		return false
	}
	_, dataType, _, err := jsonparser.Get(bid.Bid.Ext, "dsa")
	if dataType == jsonparser.Object && err == nil {
		return true
	} else if err != nil && err != jsonparser.KeyPathNotFoundError {
		return true
	}
	return false
}

// dsaRequired examines the bid request to determine if the dsarequired field indicates
// that bid responses include a dsa object
func dsaRequired(req *openrtb_ext.RequestWrapper) bool {
	regExt, err := req.GetRegExt()
	if regExt == nil || err != nil {
		return false
	}
	regsDSA := regExt.GetDSA()
	if regsDSA == nil {
		return false
	}
	if regsDSA.Required == Required || regsDSA.Required == RequiredOnlinePlatform {
		return true
	}
	return false
}
