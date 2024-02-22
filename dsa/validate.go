package dsa

import (
	"errors"

	"github.com/prebid/prebid-server/v2/exchange/entities"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/prebid/prebid-server/v2/util/jsonutil"
)

// ObjectSignal represents publisher DSA object required statuses
type ObjectSignal int8

const (
	ObjectSignalRequired               ObjectSignal = 2 // bid responses without DSA object will not be accepted
	ObjectSignalRequiredOnlinePlatform ObjectSignal = 3 // bid responses without DSA object will not be accepted, Publisher is Online Platform
)

// PubSignal represents publisher rendering intentions
type PubSignal int8

const (
	PubSignalCannotRender PubSignal = 0 // publisher can't render
	PubSignalCanRender    PubSignal = 1 // publisher could render depending on adrender
	PubSignalWillRender   PubSignal = 2 // publisher will render
)

// BuyerSignal represents buyer/advertiser rendering intentions
type BuyerSignal int8

const (
	BuyerSignalWontRender BuyerSignal = 0 // buyer/advertiser will not render
	BuyerSignalWillRender BuyerSignal = 1 // buyer/advertiser will render
)

var (
	ErrDsaMissing        = errors.New("DSA object missing when required")
	ErrBehalfTooLong     = errors.New("DSA behalf exceeds limit of 100 chars")
	ErrPaidTooLong       = errors.New("DSA paid exceeds limit of 100 chars")
	ErrNeitherWillRender = errors.New("DSA publisher and buyer both signal will not render")
	ErrBothWillRender    = errors.New("DSA publisher and buyer both signal will render")
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
	if len(bidDSA.Behalf) > 100 {
		return ErrBehalfTooLong
	}
	if len(bidDSA.Paid) > 100 {
		return ErrPaidTooLong
	}
	if reqDSA.PubRender == int8(PubSignalCannotRender) && bidDSA.AdRender != int8(BuyerSignalWillRender) {
		return ErrNeitherWillRender
	}
	if reqDSA.PubRender == int8(PubSignalWillRender) && bidDSA.AdRender == int8(BuyerSignalWillRender) {
		return ErrBothWillRender
	}
	return nil
}

// dsaRequired examines the bid request to determine if the dsarequired field indicates
// that bid responses include a dsa object
func dsaRequired(dsa *openrtb_ext.ExtRegsDSA) bool {
	if dsa == nil {
		return false
	}
	return dsa.Required == int8(ObjectSignalRequired) || dsa.Required == int8(ObjectSignalRequiredOnlinePlatform)
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
