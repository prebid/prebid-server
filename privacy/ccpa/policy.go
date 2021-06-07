package ccpa

import (
	"fmt"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// Policy represents the CCPA regulatory information from an OpenRTB bid request.
type Policy struct {
	Consent       string
	NoSaleBidders []string
}

// ReadFromRequest extracts the CCPA regulatory information from an OpenRTB bid request.
func ReadFromRequest(req *openrtb_ext.RequestWrapper) (Policy, error) {
	var consent string
	var noSaleBidders []string

	if req == nil {
		return Policy{}, nil
	}

	// Read consent from request.regs.ext
	err := req.ExtractRegExt()
	if err != nil {
		return Policy{}, fmt.Errorf("error reading request.regs.ext: %s", err)
	}
	if req.RegExt != nil {
		consent = req.RegExt.GetUSPrivacy()
	}
	// Read no sale bidders from request.ext.prebid
	err = req.ExtractRequestExt()
	if err != nil {
		return Policy{}, fmt.Errorf("error reading request.ext: %s", err)
	}
	reqPrebid := req.RequestExt.GetPrebid()
	if reqPrebid != nil {
		noSaleBidders = reqPrebid.NoSale
	}

	return Policy{consent, noSaleBidders}, nil
}

// Write mutates an OpenRTB bid request with the CCPA regulatory information.
func (p Policy) Write(req *openrtb_ext.RequestWrapper) error {
	if req == nil {
		return nil
	}

	err := req.ExtractRegExt()
	if err != nil {
		return err
	}
	req.RegExt.SetUSPrivacy(p.Consent)

	err = req.ExtractRequestExt()
	if err != nil {
		return err
	}
	buildExt(p.NoSaleBidders, req.RequestExt)
	return nil
}

func buildExt(noSaleBidders []string, ext *openrtb_ext.RequestExt) {
	if len(noSaleBidders) == 0 {
		buildExtClear(ext)
	} else {
		buildExtWrite(noSaleBidders, ext)
	}
}

func buildExtClear(ext *openrtb_ext.RequestExt) {
	prebid := ext.GetPrebid()
	if prebid == nil {
		return
	}

	// Remove no sale member
	prebid.NoSale = []string{}
	ext.SetPrebid(prebid)
}

func buildExtWrite(noSaleBidders []string, ext *openrtb_ext.RequestExt) {
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
	return
}
