package ccpa

import (
	"fmt"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Policy represents the CCPA regulatory information from an OpenRTB bid request.
type Policy struct {
	Consent       string
	NoSaleBidders []string
}

// ReadFromRequestWrapper extracts the CCPA regulatory information from an OpenRTB bid request.
func ReadFromRequestWrapper(req *openrtb_ext.RequestWrapper) (Policy, error) {
	var consent string
	var noSaleBidders []string

	if req == nil {
		return Policy{}, nil
	}

	// Read consent from request.regs.ext
	regsExt, err := req.GetRegExt()
	if err != nil {
		return Policy{}, fmt.Errorf("error reading request.regs.ext: %s", err)
	}
	if regsExt != nil {
		consent = regsExt.GetUSPrivacy()
	}
	// Read no sale bidders from request.ext.prebid
	reqExt, err := req.GetRequestExt()
	if err != nil {
		return Policy{}, fmt.Errorf("error reading request.ext: %s", err)
	}
	reqPrebid := reqExt.GetPrebid()
	if reqPrebid != nil {
		noSaleBidders = reqPrebid.NoSale
	}

	return Policy{consent, noSaleBidders}, nil
}

func ReadFromRequest(req *openrtb2.BidRequest) (Policy, error) {
	return ReadFromRequestWrapper(&openrtb_ext.RequestWrapper{BidRequest: req})
}

// Write mutates an OpenRTB bid request with the CCPA regulatory information.
func (p Policy) Write(req *openrtb_ext.RequestWrapper) error {
	if req == nil {
		return nil
	}

	regsExt, err := req.GetRegExt()
	if err != nil {
		return err
	}

	reqExt, err := req.GetRequestExt()
	if err != nil {
		return err
	}

	regsExt.SetUSPrivacy(p.Consent)
	setPrebidNoSale(p.NoSaleBidders, reqExt)
	return nil
}

func setPrebidNoSale(noSaleBidders []string, ext *openrtb_ext.RequestExt) {
	if len(noSaleBidders) == 0 {
		setPrebidNoSaleClear(ext)
	} else {
		setPrebidNoSaleWrite(noSaleBidders, ext)
	}
}

func setPrebidNoSaleClear(ext *openrtb_ext.RequestExt) {
	prebid := ext.GetPrebid()
	if prebid == nil {
		return
	}

	// Remove no sale member
	prebid.NoSale = []string{}
	ext.SetPrebid(prebid)
}

func setPrebidNoSaleWrite(noSaleBidders []string, ext *openrtb_ext.RequestExt) {
	if ext == nil {
		// This should hopefully not be possible. The only caller insures that this has been initialized
		return
	}

	prebid := ext.GetPrebid()
	if prebid == nil {
		prebid = &openrtb_ext.ExtRequestPrebid{}
	}
	prebid.NoSale = noSaleBidders
	ext.SetPrebid(prebid)
}
