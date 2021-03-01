package ccpa

import (
	"encoding/json"
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
	req.ExtractRegExt()
	if req.RegExt != nil {
		usPrivacyJSON, hasUSPrivacy := req.RegExt.Ext["us_privacy"]
		if hasUSPrivacy {
			if err := json.Unmarshal(usPrivacyJSON, &consent); err != nil {
				return Policy{}, fmt.Errorf("error reading request.regs.ext: %s", err)
			}
		}
	}
	// Read no sale bidders from request.ext.prebid
	req.ExtractRequestExt()
	if req.RequestExt != nil && req.RequestExt.Prebid != nil {
		noSaleBidders = req.RequestExt.Prebid.NoSale
	}

	return Policy{consent, noSaleBidders}, nil
}

// Write mutates an OpenRTB bid request with the CCPA regulatory information.
func (p Policy) Write(req *openrtb_ext.RequestWrapper) error {
	if req == nil {
		return nil
	}

	req.ExtractRegExt()
	req.ExtractRequestExt()
	buildRegs(p.Consent, req.RegExt)

	buildExt(p.NoSaleBidders, req.RequestExt)
	return nil
}

// START HERE
// was regs == *openrtb.Regs
// No need to return RegExt as the containing struct should still exist. I don't
// think there was a need to make a new Regs.Ext when the Ext was modified.
func buildRegs(consent string, regs *openrtb_ext.RegExt) {
	if consent == "" {
		buildRegsClear(regs)
	} else {
		buildRegsWrite(consent, regs)
	}
}

func buildRegsClear(regs *openrtb_ext.RegExt) {
	if regs == nil || len(regs.Ext) == 0 {
		return
	}

	if _, hasUSPrivacy := regs.Ext["us_privacy"]; hasUSPrivacy {
		delete(regs.Ext, "us_privacy")
		regs.Dirty = true
	}
}

// buildRegsWrite becomes an almost a one liner
func buildRegsWrite(consent string, regs *openrtb_ext.RegExt) {
	regs.USPrivacy = consent
	regs.USPrivacyDirty = true
}

func buildExt(noSaleBidders []string, ext *openrtb_ext.RequestExt) {
	if len(noSaleBidders) == 0 {
		buildExtClear(ext)
	} else {
		buildExtWrite(noSaleBidders, ext)
	}
}

func buildExtClear(ext *openrtb_ext.RequestExt) {
	if ext.Prebid == nil {
		return
	}

	// Remove no sale member
	ext.Prebid.NoSale = []string{}
	ext.PrebidDirty = true
}

func buildExtWrite(noSaleBidders []string, ext *openrtb_ext.RequestExt) {
	if ext == nil {
		// This should hopefully not be possible. The only caller insures that this has been initialized
		return
	}

	if ext.Prebid == nil {
		ext.Prebid = &openrtb_ext.ExtRequestPrebid{}
	}
	ext.Prebid.NoSale = noSaleBidders
	ext.PrebidDirty = true
	return
}
