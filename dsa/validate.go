package dsa

import (
	"errors"

	"github.com/prebid/prebid-server/v3/exchange/entities"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// Required values representing whether a DSA object is required
const (
	Required               int8 = 2 // bid responses without DSA object will not be accepted
	RequiredOnlinePlatform int8 = 3 // bid responses without DSA object will not be accepted, Publisher is Online Platform
)

// PubRender values representing publisher rendering intentions
const (
	PubRenderCannotRender int8 = 0 // publisher can't render
	PubRenderWillRender   int8 = 2 // publisher will render
)

// AdRender values representing buyer/advertiser rendering intentions
const (
	AdRenderWillRender int8 = 1 // buyer/advertiser will render
)

var (
	ErrDsaMissing        = errors.New("DSA object missing when required")
	ErrBehalfTooLong     = errors.New("DSA behalf exceeds limit of 100 chars")
	ErrPaidTooLong       = errors.New("DSA paid exceeds limit of 100 chars")
	ErrNeitherWillRender = errors.New("DSA publisher and buyer both signal will not render")
	ErrBothWillRender    = errors.New("DSA publisher and buyer both signal will render")
)

const (
	behalfMaxLength = 100
	paidMaxLength   = 100
)

// Validate determines whether a given bid is valid from a DSA perspective.
// A bid is considered valid unless the bid request indicates that a DSA object is required
// in bid responses and the object happens to be missing from the specified bid, or if the bid
// DSA object contents are invalid
func Validate(req *openrtb_ext.RequestWrapper, bid *entities.PbsOrtbBid) error {
	reqDSA := getReqDSA(req)
	bidDSA := getBidDSA(bid)

	if dsaRequired(reqDSA) && bidDSA == nil {
		return ErrDsaMissing
	}
	if bidDSA == nil {
		return nil
	}
	if len(bidDSA.Behalf) > behalfMaxLength {
		return ErrBehalfTooLong
	}
	if len(bidDSA.Paid) > paidMaxLength {
		return ErrPaidTooLong
	}
	if reqDSA != nil && reqDSA.PubRender != nil && bidDSA.AdRender != nil {
		if *reqDSA.PubRender == PubRenderCannotRender && *bidDSA.AdRender != AdRenderWillRender {
			return ErrNeitherWillRender
		}
		if *reqDSA.PubRender == PubRenderWillRender && *bidDSA.AdRender == AdRenderWillRender {
			return ErrBothWillRender
		}
	}
	return nil
}

// dsaRequired examines the bid request to determine if the dsarequired field indicates
// that bid responses include a dsa object
func dsaRequired(dsa *openrtb_ext.ExtRegsDSA) bool {
	if dsa == nil || dsa.Required == nil {
		return false
	}
	return *dsa.Required == Required || *dsa.Required == RequiredOnlinePlatform
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
