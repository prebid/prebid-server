package openrtb_ext

import (
	"fmt"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
)

func normalizeTo26(r *RequestWrapper) error {
	if err := migrateEnsureExt(r); err != nil {
		return err
	}

	migrateSupplyChainFrom24To25(r)
	migrateSupplyChainFrom25To26(r)

	migrateGDPRFrom25To26(r)

	return nil
}

// migrateEnsureExt gets all extension objects required for migration to verify there are no access errors.
func migrateEnsureExt(r *RequestWrapper) error {
	if _, err := r.GetRequestExt(); err != nil {
		return fmt.Errorf("req.ext is invalid: %v", err)
	}

	if _, err := r.GetSourceExt(); err != nil {
		return fmt.Errorf("req.source.ext is invalid: %v", err)
	}

	if _, err := r.GetRegExt(); err != nil {
		return fmt.Errorf("req.regs.ext is invalid: %v", err)
	}

	return nil
}

// migrateSupplyChainFrom24To25 modifies the request to move the OpenRTB 2.4 supply chain object (req.ext.schain)
// to the OpenRTB 2.5 location (req.source.ext.schain). If the OpenRTB 2.5 location is already present, the
// OpenRTB 2.4 supply chain object is dropped.
func migrateSupplyChainFrom24To25(r *RequestWrapper) {
	// read and clear 2.4 location
	reqExt, _ := r.GetRequestExt()
	schain24 := reqExt.GetSChain()
	reqExt.SetSChain(nil)

	// move to 2.5 location, if not already present
	sourceExt, _ := r.GetSourceExt()
	if sourceExt.GetSChain() == nil {
		sourceExt.SetSChain(schain24)
	}
}

// migrateSupplyChainFrom25To26 modifies the request to move the OpenRTB 2.5 supply chain object (req.source.ext.schain)
// to the OpenRTB 2.6 location (r.source.schain). If the OpenRTB 2.6 location is already present, the OpenRTB 2.5 supply
// chain object is dropped.
func migrateSupplyChainFrom25To26(r *RequestWrapper) {
	// read and clear 2.5 location
	sourceExt, _ := r.GetSourceExt()
	schain25 := sourceExt.GetSChain()
	sourceExt.SetSChain(nil)

	// move to 2.6 location, if not already present
	if schain25 != nil {
		if r.Source == nil {
			r.Source = &openrtb2.Source{}
		}
		if r.Source.SChain == nil {
			r.Source.SChain = schain25
		}
	}
}

func migrateGDPRFrom25To26(r *RequestWrapper) {
	// read and clear 2.5 location
	regsExt, _ := r.GetRegExt()
	gdpr25 := regsExt.GetGDPR()
	regsExt.SetGDPR(nil)

	// move to 2.6 location
	if gdpr25 != nil && r.Regs.GDPR == nil {
		r.Regs.GDPR = gdpr25
	}
}

// New field: $.regs.us_privacy. This data currently comes in on .regs.ext.us_privacy.
// New field: $.user.consent. This data currently comes in on .user.ext.consent.
// New object: $.user.eids. Contains the new ortb2 objects EIDs and UIDs. This data currently comes in on .user.ext.eids. It appears to be a lift-and-shift.
// New field: $.imp[].rwdd. This replaces the extension imp.ext.prebid.is_rewarded_inventory?
