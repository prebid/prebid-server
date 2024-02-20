package dsa

import (
	"errors"

	"github.com/prebid/prebid-server/v2/exchange/entities"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/prebid/prebid-server/v2/util/jsonutil"
)

const (
	Required               = 2 // bid responses without DSA object will not be accepted
	RequiredOnlinePlatform = 3 // bid responses without DSA object will not be accepted, Publisher is Online Platform

	PubCannotRender = 0 // publisher can't render
	PubCanRender    = 1 // publisher could render depending on adrender
	PubWillRender   = 2 // publisher will render

	BuyerWontRender = 0 // buyer/advertiser will not render
	BuyerWillRender = 1 // buyer/advertiser will render
)

// Validate determines whether a given bid is valid from a DSA perspective.
// A bid is considered valid unless the bid request indicates that a DSA object is required
// in bid responses and the object happens to be missing from the specified bid, or, if the bid
// DSA object exists, its contents are valid
func Validate(req *openrtb_ext.RequestWrapper, bid *entities.PbsOrtbBid) error {
	reqDSA := getReqDSA(req)
	bidDSA := getBidDSA(bid)

	if dsaRequired(reqDSA) && bidDSA == nil {
		return errors.New("object missing when required")
	}
	if bidDSA == nil {
		return nil
	}
	if len(bidDSA.Behalf) > 100 {
		return errors.New("behalf exceeds limit of 100 chars")
	}
	if len(bidDSA.Paid) > 100 {
		return errors.New("paid exceeds limit of 100 chars")
	}
	if reqDSA.PubRender == PubCannotRender && bidDSA.AdRender != BuyerWillRender {
		return errors.New("publisher and buyer both signal will not render")
	}
	if reqDSA.PubRender == PubWillRender && bidDSA.AdRender == BuyerWillRender {
		return errors.New("publisher and buyer both signal will render")
	}
	return nil
}

// dsaRequired examines the bid request to determine if the dsarequired field indicates
// that bid responses include a dsa object
func dsaRequired(dsa *openrtb_ext.ExtRegsDSA) bool {
	if dsa == nil {
		return false
	}
	if dsa.Required == Required || dsa.Required == RequiredOnlinePlatform {
		return true
	}
	return false
}

// getReqDSA retrieves the DSA object from the request
func getReqDSA(req *openrtb_ext.RequestWrapper) *openrtb_ext.ExtRegsDSA {
	if req == nil {
		return nil
	}
	regExt, err := req.GetRegExt()
	if regExt == nil || err != nil {
		return nil
	}
	return regExt.GetDSA()
}

// getBidDSA retrieves the DSA object from the bid
func getBidDSA(bid *entities.PbsOrtbBid) *openrtb_ext.ExtBidDSA {
	if bid == nil || bid.Bid == nil {
		return nil
	}
	var bidExt openrtb_ext.ExtBid
	if err := jsonutil.Unmarshal(bid.Bid.Ext, &bidExt); err != nil {
		return nil
	}
	return bidExt.DSA
}
