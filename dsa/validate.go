package dsa

import (
	"errors"

	"github.com/prebid/prebid-server/v2/exchange/entities"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/prebid/prebid-server/v2/util/jsonutil"
)

// ObjectSignal represents publisher DSA object required statuses
type ObjectSignal int

const (
	ObjectSignalRequired               = 2 // bid responses without DSA object will not be accepted
	ObjectSignalRequiredOnlinePlatform = 3 // bid responses without DSA object will not be accepted, Publisher is Online Platform
)

// PubSignal represents publisher rendering intentions
type PubSignal int

const (
	PubSignalCannotRender = 0 // publisher can't render
	PubSignalCanRender    = 1 // publisher could render depending on adrender
	PubSignalWillRender   = 2 // publisher will render
)

// BuyerSignal represents buyer/advertiser rendering intentions
type BuyerSignal int

const (
	BuyerSignalWontRender = 0 // buyer/advertiser will not render
	BuyerSignalWillRender = 1 // buyer/advertiser will render
)

const (
	ErrDsaMissing        = "object missing when required"
	ErrBehalfTooLong     = "behalf exceeds limit of 100 chars"
	ErrPaidTooLong       = "paid exceeds limit of 100 chars"
	ErrNeitherWillRender = "publisher and buyer both signal will not render"
	ErrBothWillRender    = "publisher and buyer both signal will render"
)

// Validate determines whether a given bid is valid from a DSA perspective.
// A bid is considered valid unless the bid request indicates that a DSA object is required
// in bid responses and the object happens to be missing from the specified bid, or if the bid
// DSA object contents are invalid
func Validate(req *openrtb_ext.RequestWrapper, bid *entities.PbsOrtbBid) error {
	reqDSA := getReqDSA(req)
	bidDSA := getBidDSA(bid)

	if dsaRequired(reqDSA) && bidDSA == nil {
		return errors.New(ErrDsaMissing)
	}
	if bidDSA == nil {
		return nil
	}
	if len(bidDSA.Behalf) > 100 {
		return errors.New(ErrBehalfTooLong)
	}
	if len(bidDSA.Paid) > 100 {
		return errors.New(ErrPaidTooLong)
	}
	if reqDSA.PubRender == PubSignalCannotRender && bidDSA.AdRender != BuyerSignalWillRender {
		return errors.New(ErrNeitherWillRender)
	}
	if reqDSA.PubRender == PubSignalWillRender && bidDSA.AdRender == BuyerSignalWillRender {
		return errors.New(ErrBothWillRender)
	}
	return nil
}

// dsaRequired examines the bid request to determine if the dsarequired field indicates
// that bid responses include a dsa object
func dsaRequired(dsa *openrtb_ext.ExtRegsDSA) bool {
	if dsa == nil {
		return false
	}
	return dsa.Required == ObjectSignalRequired || dsa.Required == ObjectSignalRequiredOnlinePlatform
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
